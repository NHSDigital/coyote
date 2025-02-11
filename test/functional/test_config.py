from utils import *

import re

def test_get_default_config_path():
    with CoyoteTestContext() as ctx:
        # We need to do something kinda nasty here because we don't know if the
        # user has a config file in their home directory. So we need to override $HOME
        # and build a config file there to make sure that we get the default config path.
        # We can't just use a temporary directory because the config path is relative to $HOME.
        home = ctx.path()/"home"
        home.mkdir()
        config_path = home/".coyoterc"
        config_path.write_text("index = \"/dev/null\"\n")

        output = coyote('config', 'path', config=None, env={'HOME': str(home)}).stdout.strip()
        assert(output.decode("utf-8") == str(config_path))

def test_get_override_config_path():
    with CoyoteTestContext() as ctx:
        config_path = unchecked_coyote('config', 'path', '--config', 'foo').stderr.strip()
        assert(re.search(r'foo.*does not exist', config_path.decode("utf-8")))

# TODO test that we can pass the config path as an environment variable
def test_environment_variable_config_path():
    with CoyoteTestContext() as ctx:
        config_path = ctx.path()/"config"
        config_path.write_text("index = \"/dev/null\"\n")

        output = coyote('config', 'path', config=None, env={'COYOTE_CONFIG': str(config_path)}).stdout.strip()
        assert(output.decode("utf-8") == str(config_path))