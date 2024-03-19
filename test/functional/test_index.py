from utils import *
import json

from http_server import PackageServer

def test_index_creates_file():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'my-chosen-tech-stack')
        index_source_path = ctx.path()/"index-source"
        index_source_path.write_text(str(package_path))
        target_path = ctx.path()/"index.cyi"

        coyote('index', 'build', index_source_path, target_path)

        assert(target_path.is_file())

def test_index_lists_conflicts():
    with CoyoteTestContext() as ctx:
        package_path1 = PackageTemplate('test-package-root') \
            .add_conflict('conflicted-package', 'other-package') \
            .build(ctx.path(), 'conflicted-package')
        package_path2 = PackageTemplate('test-other-package') \
            .build(ctx.path(), 'other-package')

        index_source_path = ctx.path()/"index-source"
        index_source_path.write_text('\n'.join(str(package_path) for package_path in [package_path1, package_path2]))
        target_path = ctx.path()/"index.cyi"

        coyote('index', 'build', index_source_path, target_path)

        index = json.loads(target_path.read_text())
        assert(index['packages']['conflicted-package']['conflicts'] == ['other-package'])


def test_conflicts_are_reflected():
    with CoyoteTestContext() as ctx:
        package_path1 = PackageTemplate('test-package-root') \
            .add_conflict('conflicted-package', 'other-package') \
            .build(ctx.path(), 'conflicted-package')
        package_path2 = PackageTemplate('test-other-package') \
            .build(ctx.path(), 'other-package')

        index_source_path = ctx.path()/"index-source"
        index_source_path.write_text('\n'.join(str(package_path) for package_path in [package_path1, package_path2]))
        target_path = ctx.path()/"index.cyi"

        coyote('index', 'build', index_source_path, target_path)

        index = json.loads(target_path.read_text())
        assert(index['packages']['other-package']['conflicts'] == ['conflicted-package'])

# Temporarily disabled while I figure out how to test this properly, given that github downloads are involved
def notest_index_lists_remote_package():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .build(ctx.path(), 'my-chosen-tech-stack')

        with PackageServer(ctx.path()) as server:
            location = server.url(package_path.name)
            index_source_path = ctx.path()/"index-source"
            index_source_path.write_text(location)
            target_path = ctx.path()/"index.cyi"

            coyote('index', 'build', index_source_path, target_path)

            index = json.loads(target_path.read_text())
            assert(index['packages']['my-chosen-tech-stack']['location'] == location)

def test_index_location_relative_to_index_file():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .build(ctx.path(), 'my-chosen-tech-stack')
        index_source_path = ctx.path()/"subdir"/"index-source"
        index_source_path.parent.mkdir(parents=True, exist_ok=True)
        index_source_path.write_text(f"../{str(package_path.name)}")
        target_path = ctx.path()/"index.cyi"
        # now what should end up in the index for the location of the package is
        # the resolved absolute path to the package, not the relative path

        coyote('index', 'build', index_source_path, target_path)
        index = json.loads(target_path.read_text())
        assert(index['packages']['my-chosen-tech-stack']['location'] == str(package_path.absolute()))

def test_error_if_file_doesnt_exist():
    with CoyoteTestContext() as ctx:
        index_source_path = ctx.path()/"index-source"
        index_source_path.write_text("doesnt-exist")
        target_path = ctx.path()/"index.cyi"

        cmd = unchecked_coyote('index', 'build', index_source_path, target_path)
        assert("package file missing: " in cmd.stderr.decode('utf-8'))

def test_index_release_fails_with_no_source_file():
    with CoyoteTestContext() as ctx:
        cmd = unchecked_coyote('index', 'release', 'no-such-file', "doesntmatter")
        assert("index source file not found: no-such-file" in cmd.stderr.decode('utf-8'))

# coyote index release needs to be run in a checkout of the index repo so that it can make a release commit
# and upload the built index as a release
def test_index_release_fails_if_not_in_git_repo():
    with CoyoteTestContext() as ctx:
        src_path = ctx.path()/'index-src'
        src_path.write_text("anything")
        cmd = unchecked_coyote('index', 'release', 'index-src', "doesntmatter")
        assert("not in a git repository" in cmd.stderr.decode('utf-8'))

