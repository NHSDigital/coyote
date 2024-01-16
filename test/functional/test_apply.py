import os
from utils import *

def test_trying_to_apply_not_in_a_project_fails():
    with CoyoteTestContext() as ctx:
        package_path = build_package(ctx.path(), 'test-package-root', 'test')

        assert(unchecked_coyote('apply', package_path).returncode == 1)

def test_apply_extracts_files():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .add_file('canary', 'This file should be extracted') \
            .commit() \
            .build(ctx.path(), 'test')

        with NewProjectContext('target'):
            print(coyote('apply', package_path))
            assert(Path('canary').is_file())


def test_apply_excludes_original_metadata():
    with CoyoteTestContext() as ctx:
        package_path = build_package(ctx.path(), 'test-package-root', 'test')

        with NewProjectContext('target'):
            coyote('apply', package_path)
            assert(not Path('.cypkg').is_dir())


def test_apply_excludes_cymeta():
    with CoyoteTestContext() as ctx:
        package_path = build_package(ctx.path(), 'test-package-root', 'test')

        with NewProjectContext('target'):
            coyote('apply', package_path)
            assert(not Path('.CYMETA').is_dir())


def test_apply_records_package_as_installed():
    with CoyoteTestContext() as ctx:
        package_path = build_package(ctx.path(), 'test-package-root', 'test')

        with NewProjectContext('target'):
            coyote('apply', package_path)
            assert(Path('.coyote/installed').is_file())
            assert(Path('.coyote/installed').read_text().strip() == 'test=v1.42.0')

def test_apply_two_packages_records_both_as_installed():
    with CoyoteTestContext() as ctx:
        package_path1 = build_package(ctx.path(), 'test-package-root', 'test')
        package_path2 = build_package(ctx.path(), 'test-package-root2', 'test2')

        with NewProjectContext('target'):
            coyote('apply', package_path1)
            coyote('apply', package_path2)
            assert(Path('.coyote/installed').is_file())
            assert(Path('.coyote/installed').read_text().strip() == 'test=v1.42.0\ntest2=v1.42.0')

def test_refuses_to_apply_conflicting_packages():
    with CoyoteTestContext() as ctx:
        pkg1 = PackageTemplate('test-package-root1') \
            .version('1.0.0') \
            .build(ctx.path(), 'incompatible')
        pkg2 = PackageTemplate('test-package-root2') \
            .version('1.0.0') \
            .add_conflict('try-to-install', 'incompatible') \
            .build(ctx.path(), 'try-to-install')

        with NewProjectContext('target'):
            assert(coyote('apply', pkg1).returncode == 0)
            second = unchecked_coyote('apply', pkg2)
            assert(second.returncode == 1)

def test_apply_templates_values_within_files():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-with-template') \
            .add_file('templated-file', 'The name of the project is {{.ProjectName}}') \
            .commit() \
            .build(ctx.path(), 'templated-package')

        with NewProjectContext('my-project'):
            coyote('apply', package_path)
            assert(Path('templated-file').read_text() == 'The name of the project is my-project')

def test_apply_templates_filenames():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-filename-template') \
            .add_file('{{.ProjectName}}.txt', 'some arbitrary file contents') \
            .commit() \
            .build(ctx.path(), 'templated-package')

        with NewProjectContext('my-project'):
            coyote('apply', package_path)
            assert(Path('my-project.txt').is_file())

# NB: we don't need to test creating empty directories, because git
# doesn't track them.  They can't exist as long as our model is that
# packages are git repositories.
def test_apply_templates_directory_names():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-directory-template') \
            .add_file('{{.ProjectName}}/templated-file', 'some arbitrary file contents') \
            .commit() \
            .build(ctx.path(), 'templated-package')

        with NewProjectContext('my-project'):
            coyote('apply', package_path)
            assert(Path('my-project/templated-file').is_file())


def test_apply_preserves_file_modes():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-file-mode') \
            .add_file('executable', 'something executable', executable=True) \
            .add_file('nonexecutable', 'something else') \
            .build(ctx.path(), 'executable-package')

        with NewProjectContext('my-project'):
            coyote('apply', package_path)
            assert(Path('executable').stat().st_mode & 0o111 == 0o111)
            assert(Path('nonexecutable').stat().st_mode & 0o111 == 0)


def test_apply_preserves_symlinks():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-symlink') \
            .add_symlink('link', 'target') \
            .build(ctx.path(), 'symlink-package')

        with NewProjectContext('my-project'):
            coyote('apply', package_path)
            assert(Path('link').is_symlink())
            assert(Path('link').resolve() == Path('target').resolve())

def test_apply_runs_on_install_script():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .on_install('my-package', '#!/bin/sh\ntouch canary\n') \
            .build(ctx.path(), 'my-package')

        with NewProjectContext('target'):
            assert(coyote('apply', package_path).returncode == 0)
            assert(Path('canary').is_file())
