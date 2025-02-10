from pathlib import Path
import os

from utils import *

def test_release_file():
    with CoyoteTestContext() as ctx:
        git("init")
        git("remote", "add", "origin", "https://github.com/org/repo.git")
        file_path = ctx.path() / "canary"
        file_path.write_text("This is a test release file")
        output = coyote('--fake-github', 'release', 'dummytag', file_path)
        assert f"NullSourceControl.CreateRelease( repo , org , dummytag , [{file_path}] )" in output.stdout.decode('utf-8')