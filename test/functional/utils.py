from pathlib import Path
import os
import tempfile
import subprocess

def unchecked_coyote(*args):
    """Run the coyote command, ignoring the return code. Use this if you are testing for an error."""
    coyote_path = Path(__file__).resolve().parent / '..' / '..' / 'build' / 'bin' / 'coyote'
    return subprocess.run([str(coyote_path)] + list(args),
                          stdout=subprocess.PIPE,
                          stderr=subprocess.PIPE)

def coyote(*args):
    """Run the coyote command, returning the result."""
    result = unchecked_coyote(*args)
    if result.returncode != 0:
        raise Exception(result.stderr.decode('utf-8', 'ignore'))
    return result


def git(*args):
    return subprocess.run(['git'] + list(args),
                          stdout=subprocess.PIPE,
                          stderr=subprocess.PIPE)


def create_package(name):
    coyote('package', 'init', name)
    with open('example-file', 'w') as f: f.write('Yes, this file exists.')
    git('init')
    git('add', '.')
    git('commit', '-m', 'initial commit')


class PackageTemplate:
    def __init__(self, repo_root_name):
        self.ops = []
        self.repo_root_name = repo_root_name
        self.is_version_set = False
        self.need_commit = False

    def version(self, v):
        self.ops.append(('set_version', v))
        self.is_version_set = True
        return self

    def add_file(self, path, contents, executable=False):
        self.ops.append(('add_file', path, contents, executable))
        self.need_commit = True
        return self

    def add_symlink(self, path, target):
        self.ops.append(('add_symlink', path, target))
        self.need_commit = True
        return self

    def add_dependency(self, pkg, dep):
        self.ops.append(('add_dependency', pkg, dep))
        self.need_commit = True
        return self

    def add_conflict(self, pkg, conflict):
        self.ops.append(('add_conflict', pkg, conflict))
        self.need_commit = True
        return self

    def on_install(self, pkg, script):
        self.ops.append(('on_install', pkg, script))
        self.need_commit = True
        return self

    def commit(self, msg="No message given"):
        self.ops.append(('commit', msg))
        self.need_commit = False
        return self

    def build(self, workdir, pkg):
        with NewDirContext(workdir / self.repo_root_name):
            create_package(pkg)
            if not self.is_version_set:
                self.version('1.42.0')
            if self.need_commit:
                self.commit()
            for op in self.ops:
                if op[0] == 'set_version':
                    git('tag', 'coyote-v' + op[1])
                elif op[0] == 'add_file':
                    target = Path(op[1])
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_text(op[2])
                    if op[3]:
                        target.chmod(target.stat().st_mode | 0o111)
                    git('add', op[1])
                elif op[0] == 'add_symlink':
                    target = Path(op[1])
                    target.parent.mkdir(parents=True, exist_ok=True)
                    os.symlink(op[2], op[1])
                    git('add', op[1])
                elif op[0] == 'add_dependency':
                    Path(f".cypkg/{op[1]}/DEPENDS").write_text(op[2] + '\n')
                elif op[0] == 'add_conflict':
                    Path(f".cypkg/{op[1]}/CONFLICTS").write_text(op[2] + '\n')
                elif op[0] == 'on_install':
                    Path(f".cypkg/{op[1]}/on-install").write_text(op[2] + '\n')
                    Path(f".cypkg/{op[1]}/on-install").chmod(0o755)
                elif op[0] == 'commit':
                    git('add', '.')
                    git('commit', '-m', op[1])
            package_name = coyote('package', 'build', pkg).stdout.decode('utf-8').strip()
            os.rename(package_name, workdir / package_name)
            return workdir / package_name


def build_package(workdir, repo_root, name):
    return PackageTemplate(repo_root) \
        .build(workdir, name)


class DirContext:
    def __init__(self, path):
        self.path = Path(path)

    def __enter__(self):
        self.currentdir = Path.cwd()
        os.chdir(self.path)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        os.chdir(self.currentdir)


def unpack(package):
    return subprocess.run(['tar', 'xzf', str(package)],
                          stdout=subprocess.PIPE,
                          stderr=subprocess.PIPE)


class NewDirContext(DirContext):
    def __init__(self, path):
        super().__init__(path)
        self.path.mkdir()

class CoyoteTestContext:
    def __enter__(self):
        self.currentdir = Path.cwd()
        self.tmpdir = tempfile.TemporaryDirectory()
        os.chdir(self.tmpdir.name)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.tmpdir.cleanup()
        os.chdir(self.currentdir)

    def path(self, *args):
        return Path(self.tmpdir.name, *args)

class NewProjectContext(CoyoteTestContext):
    def __init__(self, name, tech_stack='empty'):
        super().__init__()
        self.name = name
        self.tech_stack = tech_stack

    def __enter__(self):
        self.parent_path = Path.cwd()
        coyote('init', self.tech_stack, self.name)
        os.chdir(self.name)
        self.project_path = Path.cwd()
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        os.chdir(self.parent_path)

    def path(self, *args):
        return Path(self.project_path, *args)

class Index:
    def __init__(self, source_path, target_path):
        self.source_path = source_path
        self.target_path = target_path

class Indexer:
    def __init__(self, *packages):
        self.packages = packages

    def add_package(self, package):
        self.packages.append(package)
        return self

    def build(self, ctx):
        index = Index(ctx.path()/"index-source", ctx.path()/"index.cyi")
        index.source_path.write_text('\n'.join(str(package) for package in self.packages))
        coyote('index', 'build', index.source_path, index.target_path)
        return index
