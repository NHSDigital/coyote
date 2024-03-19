from pathlib import Path
import os

from utils import *

def test_package_new():
    with CoyoteTestContext() as ctx:
        package_org = "test-github-org"
        config=f"package_org = \"{package_org}\"\nindex = \"/dev/null\""
        output = coyote('--fake-github', 'package', 'new', 'test', config=config)
        assert 'NullSourceControl.CreateRepo( cypkg-test , test-github-org )' in output.stdout.decode('utf-8')

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
