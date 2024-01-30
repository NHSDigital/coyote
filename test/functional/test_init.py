from utils import *

def test_init_creates_project_dir():
    with CoyoteTestContext() as ctx:
        # TODO: are there any places where we don't support hyphens that this is going to matter?
        # Assume not for now
        coyote('init', 'empty', 'my-project')
        assert ctx.path('my-project').exists()

def test_init_saves_project_name():
    with CoyoteTestContext() as ctx:
        with NewProjectContext('my-project') as project:
            assert project.path('.coyote/project-name').read_text() == 'my-project'

def test_init_from_package():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'my-chosen-tech-stack')
        index = Indexer(package_path).build(ctx)
        # this is equivalent to:
        #  coyote init empty my-new-project
        #  cd my-new-project
        #  coyote install my-chosen-tech-stack --index ../index.cyi
        coyote('init', 'my-chosen-tech-stack', 'my-new-project', '--index', index.target_path)
        print(list(ctx.path('my-new-project').glob('**/*')))
        assert(Path('my-new-project', 'canary').is_file())

def test_init_fails_if_project_already_exists():
    with CoyoteTestContext() as ctx:
        coyote('init', 'empty', 'my-project')
        assert(unchecked_coyote('init', 'empty', 'my-project').returncode == 1)


def test_init_fails_if_no_index_specified():
    with CoyoteTestContext() as ctx:
        assert(unchecked_coyote('init', 'any-package', 'anything').returncode == 1)

def test_init_fails_if_index_doesnt_exist():
    with CoyoteTestContext() as ctx:
        assert(unchecked_coyote('init', 'anything', 'anything', '--index', 'doesnt-exist.cyi').returncode == 1)


def test_init_fails_if_index_isnt_a_file():
    with CoyoteTestContext() as ctx:
        assert(unchecked_coyote('init', 'anything', 'anything', '--index', '.').returncode == 1)

def test_init_fails_if_index_isnt_a_valid_index():
    with CoyoteTestContext() as ctx:
        ctx.path('not-an-index.cyi').touch()
        cmd = unchecked_coyote('init', 'anything', 'anything', '--index', 'not-an-index.cyi')
        assert(cmd.returncode == 1)


def test_init_fails_if_index_doesnt_contain_package():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'my-chosen-tech-stack')
        index = Indexer(package_path).build(ctx)
        cmd = unchecked_coyote('init', 'some-other-package', 'my-new-project', '--index', index.target_path)

        assert(cmd.returncode == 1)

def test_init_fails_if_index_contains_package_but_package_isnt_there():
    with CoyoteTestContext() as ctx:
        package_path = PackageTemplate('test-package-root') \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'my-chosen-tech-stack')
        index = Indexer(package_path).build(ctx)

        package_path.unlink()
        cmd = unchecked_coyote('init', 'my-chosen-tech-stack', 'my-new-project', '--index', index.target_path)

        assert(cmd.returncode == 1)

def test_init_installs_package_dependencies():
    with CoyoteTestContext() as ctx:
        package_a_path = PackageTemplate('package-a-root') \
            .add_dependency('my-chosen-tech-stack', 'upstream-dependency') \
            .build(ctx.path(), 'my-chosen-tech-stack')

        package_b_path = PackageTemplate('package-b-root') \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'upstream-dependency')

        index = Indexer(package_a_path, package_b_path).build(ctx)

        coyote('init', 'my-chosen-tech-stack', 'my-new-project', '--index', index.target_path)

        assert(Path('my-new-project', 'canary').is_file())

def test_init_installs_package_dependencies_recursively():
    with CoyoteTestContext() as ctx:
        package_a_path = PackageTemplate('base-root') \
            .add_dependency('my-chosen-tech-stack', 'dependency') \
            .build(ctx.path(), 'my-chosen-tech-stack')

        package_b_path = PackageTemplate('dependency-root') \
            .add_dependency('dependency', 'grand-dependency') \
            .build(ctx.path(), 'dependency')

        package_c_path = PackageTemplate('grand-dependency-root') \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'grand-dependency')

        index = Indexer(package_a_path, package_b_path, package_c_path).build(ctx)

        coyote('init', 'my-chosen-tech-stack', 'my-new-project', '--index', index.target_path)

        assert(Path('my-new-project', 'canary').is_file())
