from utils import *


def test_upgrade_package_to_latest_version():
    with CoyoteTestContext() as ctx:
        # Build two versions of the same package
        package_v1 = PackageTemplate('test-package-v1') \
            .add_file('canary', 'version 1') \
            .commit('add canary v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-dependency')
        package_v2 = PackageTemplate('test-package-v2') \
            .add_file('canary', 'version 2') \
            .commit('add canary v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'my-dependency')
        index = Indexer(package_v1, package_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            # Install v1.0.0 explicitly
            coyote('install', 'my-dependency@v1.0.0', '--index', index.target_path)
            assert(Path('canary').read_text() == 'version 1')
            # Upgrade to latest
            coyote('upgrade', 'my-dependency', '--index', index.target_path)
            assert(Path('canary').read_text() == 'version 2')


def test_upgrade_not_installed_package_fails():
    with CoyoteTestContext() as ctx:
        package_v1 = PackageTemplate('test-package-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-dependency')
        index = Indexer(package_v1).build(ctx)
        with NewProjectContext('my-new-project'):
            cmd = unchecked_coyote('upgrade', 'my-dependency', '--index', index.target_path)
            assert(cmd.returncode != 0)
            assert('not installed' in cmd.stderr.decode('utf-8').lower())


def test_upgrade_already_at_latest_version():
    with CoyoteTestContext() as ctx:
        package_v1 = PackageTemplate('test-package-v1') \
            .add_file('canary', 'version 1') \
            .commit('add canary') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-dependency')
        index = Indexer(package_v1).build(ctx)
        with NewProjectContext('my-new-project'):
            coyote('install', 'my-dependency', '--index', index.target_path)
            # Upgrade when already at latest should succeed silently
            result = coyote('upgrade', 'my-dependency', '--index', index.target_path)
            # File should still be there unchanged
            assert(Path('canary').read_text() == 'version 1')


def test_upgrade_runs_on_install_script():
    with CoyoteTestContext() as ctx:
        package_v1 = PackageTemplate('test-package-v1') \
            .on_install('my-dependency', '#!/bin/sh\necho -n "v1" >> install-log\n') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-dependency')
        package_v2 = PackageTemplate('test-package-v2') \
            .on_install('my-dependency', '#!/bin/sh\necho -n "v2" >> install-log\n') \
            .version('v2.0.0') \
            .build(ctx.path(), 'my-dependency')
        index = Indexer(package_v1, package_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            coyote('install', 'my-dependency@v1.0.0', '--index', index.target_path)
            assert(Path('install-log').read_text() == 'v1')
            coyote('upgrade', 'my-dependency', '--index', index.target_path)
            assert(Path('install-log').read_text() == 'v1v2')


def test_upgrade_fails_if_new_version_conflicts_with_installed_package():
    with CoyoteTestContext() as ctx:
        # package-a v1.0.0 has no conflicts
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        # package-a v2.0.0 declares a conflict with package-b
        pkg_a_v2 = PackageTemplate('pkg-a-v2') \
            .add_conflict('package-a', 'package-b') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-a')
        # package-b v1.0.0
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')

        # First, build an index with only v1 versions (no conflicts)
        index_v1 = Indexer(pkg_a_v1, pkg_b_v1).build(ctx)

        with NewProjectContext('my-new-project'):
            # Install package-a v1.0.0 and package-b v1.0.0 using v1 index (no conflict)
            coyote('install', 'package-a@v1.0.0', '--index', index_v1.target_path)
            coyote('install', 'package-b@v1.0.0', '--index', index_v1.target_path)

            # Now build a new index that includes v2.0.0 with the conflict
            index_v2 = Indexer(pkg_a_v1, pkg_a_v2, pkg_b_v1).build(ctx)

            # Upgrading package-a should fail because v2.0.0 conflicts with package-b
            cmd = unchecked_coyote('upgrade', 'package-a', '--index', index_v2.target_path)
            assert(cmd.returncode != 0)
            assert('conflict' in cmd.stderr.decode('utf-8').lower())


def test_upgrade_updates_installed_version():
    with CoyoteTestContext() as ctx:
        package_v1 = PackageTemplate('test-package-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-dependency')
        package_v2 = PackageTemplate('test-package-v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'my-dependency')
        index = Indexer(package_v1, package_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            coyote('install', 'my-dependency@v1.0.0', '--index', index.target_path)
            assert('my-dependency=v1.0.0' in Path('.coyote/installed').read_text())
            coyote('upgrade', 'my-dependency', '--index', index.target_path)
            installed = Path('.coyote/installed').read_text()
            assert('my-dependency=v2.0.0' in installed)
            # Should not have duplicate entries
            assert(installed.count('my-dependency=') == 1)


def test_upgrade_not_in_project_fails():
    with CoyoteTestContext() as ctx:
        cmd = unchecked_coyote('upgrade', 'anything')
        assert(cmd.returncode != 0)
        assert('not in a coyote project' in cmd.stderr.decode('utf-8').lower())


def test_upgrade_package_not_in_index_fails():
    with CoyoteTestContext() as ctx:
        package_v1 = PackageTemplate('test-package-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-dependency')
        package_other = PackageTemplate('test-other') \
            .version('v1.0.0') \
            .build(ctx.path(), 'other-package')
        # Only include my-dependency in the index, not other-package
        index = Indexer(package_v1).build(ctx)
        with NewProjectContext('my-new-project'):
            # Manually install other-package using apply (bypasses index)
            coyote('apply', str(package_other))
            # Try to upgrade other-package which is installed but not in the index
            cmd = unchecked_coyote('upgrade', 'other-package', '--index', index.target_path)
            assert(cmd.returncode != 0)
            assert('not found' in cmd.stderr.decode('utf-8').lower())


def test_upgrade_all_packages():
    with CoyoteTestContext() as ctx:
        # Build two packages, each with two versions
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .add_file('file-a', 'a version 1') \
            .commit('add file a v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_a_v2 = PackageTemplate('pkg-a-v2') \
            .add_file('file-a', 'a version 2') \
            .commit('add file a v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .add_file('file-b', 'b version 1') \
            .commit('add file b v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')
        pkg_b_v2 = PackageTemplate('pkg-b-v2') \
            .add_file('file-b', 'b version 2') \
            .commit('add file b v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-b')
        index = Indexer(pkg_a_v1, pkg_a_v2, pkg_b_v1, pkg_b_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            # Install v1.0.0 of both packages
            coyote('install', 'package-a@v1.0.0', '--index', index.target_path)
            coyote('install', 'package-b@v1.0.0', '--index', index.target_path)
            assert(Path('file-a').read_text() == 'a version 1')
            assert(Path('file-b').read_text() == 'b version 1')
            # Upgrade all
            coyote('upgrade', '--index', index.target_path)
            assert(Path('file-a').read_text() == 'a version 2')
            assert(Path('file-b').read_text() == 'b version 2')


def test_upgrade_all_updates_installed_file():
    with CoyoteTestContext() as ctx:
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_a_v2 = PackageTemplate('pkg-a-v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')
        pkg_b_v2 = PackageTemplate('pkg-b-v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-b')
        index = Indexer(pkg_a_v1, pkg_a_v2, pkg_b_v1, pkg_b_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            coyote('install', 'package-a@v1.0.0', '--index', index.target_path)
            coyote('install', 'package-b@v1.0.0', '--index', index.target_path)
            coyote('upgrade', '--index', index.target_path)
            installed = Path('.coyote/installed').read_text()
            assert('package-a=v2.0.0' in installed)
            assert('package-b=v2.0.0' in installed)


def test_upgrade_all_skips_packages_not_in_index():
    with CoyoteTestContext() as ctx:
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .add_file('file-a', 'a version 1') \
            .commit('add file a v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_a_v2 = PackageTemplate('pkg-a-v2') \
            .add_file('file-a', 'a version 2') \
            .commit('add file a v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .add_file('file-b', 'b version 1') \
            .commit('add file b v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')
        # Only package-a is in the index
        index = Indexer(pkg_a_v1, pkg_a_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            coyote('install', 'package-a@v1.0.0', '--index', index.target_path)
            # Install package-b via apply (bypasses index)
            coyote('apply', str(pkg_b_v1))
            # Upgrade all should succeed, upgrading package-a but skipping package-b
            coyote('upgrade', '--index', index.target_path)
            assert(Path('file-a').read_text() == 'a version 2')
            assert(Path('file-b').read_text() == 'b version 1')


def test_upgrade_all_with_nothing_installed():
    with CoyoteTestContext() as ctx:
        pkg_v1 = PackageTemplate('pkg-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'my-package')
        index = Indexer(pkg_v1).build(ctx)
        with NewProjectContext('my-new-project'):
            # Upgrade all with nothing installed should succeed silently
            coyote('upgrade', '--index', index.target_path)


def test_upgrade_multiple_specific_packages():
    with CoyoteTestContext() as ctx:
        # Build three packages, each with two versions
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .add_file('file-a', 'a version 1') \
            .commit('add file a v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_a_v2 = PackageTemplate('pkg-a-v2') \
            .add_file('file-a', 'a version 2') \
            .commit('add file a v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .add_file('file-b', 'b version 1') \
            .commit('add file b v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')
        pkg_b_v2 = PackageTemplate('pkg-b-v2') \
            .add_file('file-b', 'b version 2') \
            .commit('add file b v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-b')
        pkg_c_v1 = PackageTemplate('pkg-c-v1') \
            .add_file('file-c', 'c version 1') \
            .commit('add file c v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-c')
        pkg_c_v2 = PackageTemplate('pkg-c-v2') \
            .add_file('file-c', 'c version 2') \
            .commit('add file c v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-c')
        index = Indexer(pkg_a_v1, pkg_a_v2, pkg_b_v1, pkg_b_v2, pkg_c_v1, pkg_c_v2).build(ctx)
        with NewProjectContext('my-new-project'):
            # Install v1.0.0 of all packages
            coyote('install', 'package-a@v1.0.0', '--index', index.target_path)
            coyote('install', 'package-b@v1.0.0', '--index', index.target_path)
            coyote('install', 'package-c@v1.0.0', '--index', index.target_path)
            # Upgrade only package-a and package-c (not package-b)
            coyote('upgrade', 'package-a', 'package-c', '--index', index.target_path)
            assert(Path('file-a').read_text() == 'a version 2')
            assert(Path('file-b').read_text() == 'b version 1')  # Not upgraded
            assert(Path('file-c').read_text() == 'c version 2')


def test_upgrade_multiple_packages_one_not_installed_fails():
    with CoyoteTestContext() as ctx:
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')
        index = Indexer(pkg_a_v1, pkg_b_v1).build(ctx)
        with NewProjectContext('my-new-project'):
            # Only install package-a
            coyote('install', 'package-a', '--index', index.target_path)
            # Try to upgrade both - should fail because package-b is not installed
            cmd = unchecked_coyote('upgrade', 'package-a', 'package-b', '--index', index.target_path)
            assert(cmd.returncode != 0)
            assert('not installed' in cmd.stderr.decode('utf-8').lower())


def test_upgrade_is_atomic_no_changes_on_conflict():
    """When upgrading multiple packages and one has a conflict, NO packages should be upgraded."""
    with CoyoteTestContext() as ctx:
        # Create package-a v1.0.0 (no conflicts)
        pkg_a_v1 = PackageTemplate('pkg-a-v1') \
            .add_file('file-a', 'a version 1') \
            .commit('add file a v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-a')
        # Create package-a v2.0.0 (no conflicts)
        pkg_a_v2 = PackageTemplate('pkg-a-v2') \
            .add_file('file-a', 'a version 2') \
            .commit('add file a v2') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-a')
        # Create package-b v1.0.0 (no conflicts)
        pkg_b_v1 = PackageTemplate('pkg-b-v1') \
            .add_file('file-b', 'b version 1') \
            .commit('add file b v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-b')
        # Create package-b v2.0.0 that conflicts with package-c
        pkg_b_v2 = PackageTemplate('pkg-b-v2') \
            .add_file('file-b', 'b version 2') \
            .add_conflict('package-b', 'package-c') \
            .commit('add file b v2 with conflict') \
            .version('v2.0.0') \
            .build(ctx.path(), 'package-b')
        # Create package-c (this will conflict with package-b v2.0.0)
        pkg_c_v1 = PackageTemplate('pkg-c-v1') \
            .add_file('file-c', 'c version 1') \
            .commit('add file c v1') \
            .version('v1.0.0') \
            .build(ctx.path(), 'package-c')

        # Build index in two phases to avoid build-time conflict checking
        # First build without v2.0.0 of package-b
        index_phase1 = Indexer(pkg_a_v1, pkg_a_v2, pkg_b_v1, pkg_c_v1).build(ctx)

        with NewProjectContext('my-new-project'):
            # Install v1.0.0 of all packages using initial index
            coyote('install', 'package-a@v1.0.0', '--index', index_phase1.target_path)
            coyote('install', 'package-b@v1.0.0', '--index', index_phase1.target_path)
            coyote('install', 'package-c@v1.0.0', '--index', index_phase1.target_path)

            # Now build index with package-b v2.0.0 included
            index_phase2 = Indexer(pkg_a_v1, pkg_a_v2, pkg_b_v1, pkg_b_v2, pkg_c_v1).build(ctx)

            # Verify initial state
            assert(Path('file-a').read_text() == 'a version 1')
            assert(Path('file-b').read_text() == 'b version 1')

            # Try to upgrade both package-a and package-b
            # package-b v2.0.0 conflicts with package-c, so the entire operation should fail
            cmd = unchecked_coyote('upgrade', 'package-a', 'package-b', '--index', index_phase2.target_path)
            assert(cmd.returncode != 0)
            assert('conflict' in cmd.stderr.decode('utf-8').lower())

            # CRITICAL: package-a should NOT have been upgraded because the operation was atomic
            assert(Path('file-a').read_text() == 'a version 1'), "package-a should not have been upgraded"
            assert(Path('file-b').read_text() == 'b version 1'), "package-b should not have been upgraded"
