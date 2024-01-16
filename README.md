# coyote

Package management for source code repositories.

The problem this solves is supplying repository structures in
a manageable way, to supply pre-canned functionality that
allows reuse, upgrading, versioning, and all that goodness.

## The concepts

### Source repositories

A github repo that contains source we want to use, potentially in combination
with other source repositories.

### Package

Each source repository will publish a number of packages. Each package is a
released tarball; they are versioned. A source repository can publish more than
one package.

### Package list

A list of packages, and versions.

### Package index

A place to find a package list.

## Assumptions

The only assumption we make of any system that our tools execute on is Make and
Bash (or other shell - might decide to tighten this up later, but I don't want
to be too prescriptive to soon.)

## Sources and packages

If you put a `.cypkg` directory in the root of a repository, that makes it a
valid source repository.

Each directory under `.cypkg` defines a package, named after the directory:
`.cypkg/my-shiny-package` defines the `my-shiny-package` package.

If you put a `DEPENDS` file in the `.cypkg/my-shiny-package` directory, the
tools expect that to declare dependencies on other packages.

Similarly if you put a `CONFLICTS` file in `.cypkg/my-shiny-package`, `coyote`
will refuse to install a package to a repository that has a package listed in
this file already installed.

If you put a `on-install` file in `.cypkg/my-shiny-package` directory, that file
will be executed on the target after the package has been unpacked.

The package will have an added `.CYPKG` directory at the top level. That
directory will contain a `VERSION` file so that the filename is not constrained
to contain version information, the `DEPENDS` file, and `on-install`. This
directory is not copied to the target.

If you put a `build` file in `.cypkg/my-shiny-package`, that file will be
executed to build the tarfile. It is expected to dump a tarfile to stdout. The
default, if it is not supplied, is the equivalent of `tar cf - --exclude=.git`.
It is *not* responsible for the `.CYPKG` directory.

### Package versions

Version numbers are expected to sort asciibetically. Otherwise there are no
constraints.  They are stored as git tags, with an expected pattern of
`coyote-vWHATEVER`.  You can pass an explicit version to `coyote package build` if
you want to build a version that is not the last in the sorted list.

## Usage

`coyote package init my-shiny-package`

Create the `.cypkg/my-shiny-package` directory.

`coyote package build my-shiny-package`

Copies non-git and non-coyote files to a temporary directory, and runs `build`.
The output has the `.CYPKG` directory added, and the resulting tarball is
zipped.

The file will be written to the current directory as
`my-shiny-package_<version>.cypkg`.

`coyote package publish [filename]`

Pushes `filename`, defaulting to `my-shiny-package_<version>.cypkg`, as a
release to the source repository, identified as the git `origin` remote.

`coyote install <package>`

Grabs the latest version of the package index. Fleshes out the dependency list
for the named package, building an ordered list that can be installed without
invalidating the dependency requirements of any package installed.

`coyote apply <package-file>`

Just install the package file.  This will ignore the source repository, and will
instal no dependencies.  It *will* obey `CONFLICTS` and refuse to install if
there is already a conflicting dependency installed.

