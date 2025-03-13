from pathlib import Path
import os

from utils import *

def test_package_init():
    with CoyoteTestContext() as ctx:
        coyote('package', 'init', 'test')
        assert os.path.isdir('.cypkg')
        assert os.path.isdir('.cypkg/test')
        assert os.path.isfile('.cypkg/test/DEPENDS')
        assert os.path.isfile('.cypkg/test/CONFLICTS')


def test_package_build():
    with CoyoteTestContext():
        create_package('test')
        git('tag', 'coyote-v1.42.0')

        coyote('package', 'build', 'test')

        assert os.path.isfile('test-v1.42.0.cypkg')


def test_package_builds_highest_tag_by_default():
    with CoyoteTestContext():
        create_package('test')
        git('tag', 'coyote-v1.43.0')
        git('tag', 'coyote-v1.42.0')

        coyote('package', 'build', 'test')

        assert os.path.isfile('test-v1.43.0.cypkg')


def test_package_build_includes_directory_content():
    with CoyoteTestContext() as ctx:
        workdir = ctx.path()

        pkgdir = workdir / 'test-package-root'
        with NewDirContext(pkgdir):
            create_package('test')
            git('tag', 'coyote-v1.42.0')
            package_name = coyote('package', 'build', 'test').stdout.decode('utf-8').strip()

            os.rename(package_name, workdir / package_name)

        extractdir = workdir / 'extract'
        with NewDirContext(extractdir):
            assert(unpack(workdir / package_name).returncode == 0)
            assert(Path('example-file').is_file())


def test_package_build_includes_package_metadata():
    with CoyoteTestContext() as ctx:
        workdir = ctx.path()

        pkgdir = workdir / 'test-package-root'
        with NewDirContext(pkgdir):
            create_package('test')
            git('tag', 'coyote-v1.42.0')
            package_name = coyote('package', 'build', 'test').stdout.decode('utf-8').strip()

            os.rename(package_name, workdir / package_name)

        extractdir = workdir / 'extract'
        with NewDirContext(extractdir):
            assert(unpack(workdir / package_name).returncode == 0)
            cymeta = Path('.CYMETA')
            assert((cymeta / 'DEPENDS').is_file())
            assert((cymeta / 'CONFLICTS').is_file())
            assert((cymeta / 'VERSION').is_file())
            assert((cymeta / 'NAME').is_file())

            assert((cymeta / 'VERSION').read_text().strip() == 'v1.42.0')
            assert((cymeta / 'NAME').read_text().strip() == 'test')

def test_package_build_works_without_depends_and_conflicts_files():
    with CoyoteTestContext() as ctx:
        workdir = ctx.path()

        pkgdir = workdir / 'test-package-root'
        with NewDirContext(pkgdir):
            create_package('test')
            os.remove(pkgdir / '.cypkg' / 'test' / 'DEPENDS')
            os.remove(pkgdir / '.cypkg' / 'test' / 'CONFLICTS')
            git('tag', 'coyote-v1.42.0')
            package_name = coyote('package', 'build', 'test').stdout.decode('utf-8').strip()

            os.rename(package_name, workdir / package_name)

        extractdir = workdir / 'extract'
        with NewDirContext(extractdir):
            assert(unpack(workdir / package_name).returncode == 0)
            cymeta = Path('.CYMETA')
            assert((cymeta / 'DEPENDS').is_file())
            assert((cymeta / 'CONFLICTS').is_file())

def test_package_build_excludes_cypkg_directory():
    with CoyoteTestContext() as ctx:
        workdir = ctx.path()

        pkgdir = workdir / 'test-package-root'
        with NewDirContext(pkgdir):
            create_package('test')
            git('tag', 'coyote-v1.42.0')
            package_name = coyote('package', 'build', 'test').stdout.decode('utf-8').strip()

            os.rename(package_name, workdir / package_name)

        extractdir = workdir / 'extract'
        with NewDirContext(extractdir):
            assert(unpack(workdir / package_name).returncode == 0)
            t = extractdir / '.cypkg'
            print(list(t.glob("**/*")))
            assert(not (extractdir / '.cypkg').is_dir())


def test_package_build_includes_on_install_script():
    with CoyoteTestContext() as ctx:
        workdir = ctx.path()

        pkgdir = workdir / 'test-package-root'
        with NewDirContext(pkgdir):
            create_package('test')
            on_install_path = pkgdir / '.cypkg' / 'test' / 'on-install'
            with open(on_install_path, 'w') as f:
                f.write('#!/bin/sh\n')
                f.write('echo "Hello, world!"\n')
            on_install_path.chmod(0o755)

            git('tag', 'coyote-v1.42.0')
            package_name = coyote('package', 'build', 'test').stdout.decode('utf-8').strip()

            os.rename(package_name, workdir / package_name)

        extractdir = workdir / 'extract'
        with NewDirContext(extractdir):
            assert(unpack(workdir / package_name).returncode == 0)
            on_install_extracted = Path('.CYMETA') / 'on-install'
            assert(on_install_extracted.is_file())
            assert(on_install_extracted.stat().st_mode & 0o755 != 0)

def test_set_build_output_directory():
    with CoyoteTestContext() as ctx:
        workdir = ctx.path()

        pkgdir = workdir / 'test-package-root'
        outdir = workdir / 'build'
        with NewDirContext(pkgdir):
            create_package('test')
            git('tag', 'coyote-v1.42.0')
            result = coyote('package', 'build', 'test', '--output', str(outdir))
            package_name = result.stdout.decode('utf-8').strip()

            assert((outdir / package_name).is_file())

def test_package_template_works():
    with CoyoteTestContext() as ctx:
        template = PackageTemplate('test-package-root')
        package_path = template \
            .add_file('canary', 'This is a test package') \
            .build(ctx.path(), 'my-chosen-tech-stack')

        assert os.path.isfile(ctx.path() / 'test-package-root' / 'canary')
        assert os.path.isfile(package_path)

def test_build_prior_version():
    with CoyoteTestContext() as ctx:
        template = PackageTemplate('test-package-root')
        package_path = template \
            .add_file('canary', 'This is a prior version') \
            .commit() \
            .version('v1.43.0') \
            .add_file('budgie', 'This is a later version') \
            .commit() \
            .version('v1.44.0') \
            .build(ctx.path(), 'my-chosen-tech-stack', version='v1.43.0')

        # Now we check that when we unpack the package, we get the prior version
        with NewDirContext(ctx.path() / 'extract'):
            assert(unpack(package_path).returncode == 0)
            assert(Path('canary').is_file())
            assert(not Path('budgie').is_file())

def test_build_head():
    with CoyoteTestContext() as ctx:
        template = PackageTemplate('test-package-root')
        package_path = template \
            .add_file('canary', "This is a test file that isn't in a version-tagged commit") \
            .commit() \
            .build(ctx.path(), 'my-chosen-tech-stack', version='HEAD')

        # Now we check that when we unpack the package, we get the file that's in the prior version
        with NewDirContext(ctx.path() / 'extract'):
            assert(unpack(package_path).returncode == 0)
            assert(Path('canary').is_file())

def test_runs_build_file():
    with CoyoteTestContext() as ctx:
        template = PackageTemplate('test-package-root')
        package_path = template \
            .add_file('canary', 'This file gets included') \
            .add_file('dead_canary', 'This file should get skipped') \
            .add_build_file('my-package', '#!/bin/sh\ntar -cf - canary') \
            .build(ctx.path(), 'my-package')

        with NewDirContext(ctx.path() / 'extract'):
            assert(unpack(package_path).returncode == 0)
            assert(Path('canary').is_file())
            assert(not Path('dead_canary').exists())