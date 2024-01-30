from pathlib import Path
import os

from utils import *

def test_package_new():
    with CoyoteTestContext() as ctx:
        package_org = "test-github-org"
        config=f"package_org = \"{package_org}\"\nindex = \"/dev/null\""
        output = coyote('--fake-github', 'package', 'new', 'test', config=config)
        assert 'NullSourceControl.CreateRepo( cypkg-test , test-github-org )' in output.stdout.decode('utf-8')