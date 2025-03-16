from pathlib import Path
import os

from utils import *

def test_package_new():
    with CoyoteTestContext() as ctx:
        package_org = "test-github-org"
        config=f"package_org = \"{package_org}\"\nindex = \"/dev/null\""
        output = coyote('--fake-github', 'package', 'new', 'test', config=config)
        assert f"NullSourceControl.CreateRepo( cypkg-test , {package_org} )" in output.stdout.decode('utf-8')

def test_package_new_sets_origin():
    with CoyoteTestContext() as ctx:
        package_org = "test-github-org"
        config=f"package_org = \"{package_org}\"\nindex = \"/dev/null\""
        coyote('--fake-github', 'package', 'new', 'test', config=config)
        with DirContext("cypkg-test") as dir:
            output = git("remote", "get-url", "origin")
            assert f"fake-remote-url" in output.stdout.decode('utf-8')

# TODO Tests for package release.
# The following is from the readme:

# Sanity checks the version.  We check whether there
# is already a release on github with that version. If so, we fail.  Otherwise, we
# check whether the version already exists as a git tag.  Otherwise we tag HEAD with
# the version.  Then we push the tag to github.  Then we build the package and publish
# it as a release.
# If the command succeeds, it prints the URL of the release on stdout.
def test_sanity_check_version():
    # We can't impose version number constraints, because we always want to allow
    # 1.1.0 after 2.0.0, for instance.
    # So we just check that the version number is a valid git tag according to `git check-ref-format`
    with CoyoteTestContext() as ctx:
        bad_version = "1 whoops no spaces allowed"
        package_path = ctx.path() / "test-repo"
        package_path.mkdir()
        with DirContext(package_path):
            create_package("test-package")
            git("remote", "add", "origin", "whatever")

            output = unchecked_coyote('--fake-github', 'package', 'release', 'test', bad_version)
            assert "invalid version" in output.stderr.decode('utf-8')

def test_package_version():
    with CoyoteTestContext() as ctx:
        package_path = ctx.path() / "test-repo"
        package_path.mkdir()
        with DirContext(package_path):
            create_package("test-package")
            git("tag", "coyote-v1.42.0")
            output = coyote('package', 'version')
            assert "v1.42.0" in output.stdout.decode('utf-8')
