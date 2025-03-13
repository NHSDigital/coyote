# coyote

Package management for source code repositories.

The problem this solves is supplying repository structures in a
manageable way, to supply pre-canned functionality that allows reuse,
upgrading, versioning, and all that goodness.

## Very quick introduction (local operation)

This may seem verbose, but until we have some more of the
infrastructure set up, we need to manually put some config files in
place.

### 0. Build the `coyote` binary

If you have `go` already installed, to build `coyote` you run:

```sh
 $ make
```

That will put a `coyote` executable at `build/bin/coyote`, which from
here I will refer to as `$COYOTE`.

### 1. Setup

You will need a `.coyoterc` file. It should look like this:

```toml
index = "~/.coyote-index"
package_org = "NHSDigital"
```

The path to `index` can be anything you like, as long as you can put a
text file there. It doesn't have to be in your home directory.

Now, open an `index-src` file, and add the following contents:

```
https://github.com/NHSDigital/cypkg-repository-template/releases/download/coyote-0.0.3/default.repository-template-0.0.3.cypkg
https://github.com/NHSDigital/cypkg-python-hello-world/releases/download/coyote-0.0.1/default.python-hello-world-0.0.1.cypkg
```

Ensure that the `GITHUB_TOKEN` environment variable is a valid
auth token with rights to see download internal repositories in the
`NHSDigital` org.

Now run this command, substituting the `index` path if you changed it
above:

```
 $ $COYOTE index build ./index-src ~/.coyote-index
```

[Note: `coyote` can accept a remote index URL which avoids this step,
but I haven't set up hosting it yet.]

### 2. Make a new project

Run the following:

```
 $ $COYOTE init python-hello-world my-shiny-project
 $ cd my-shiny-project
 $ ls
```

You will see that `main.py` is listed.  Looking at the
https://github.com/NHSDigital/cypkg-python-hello-world repository, you
will see where it has come from.  When you run `$COYOTE init
<base-package> <project-name>` it uses that `<base-package>` as the
basis for your project.

However you will also see other files: a README.md, for instance.  If
you look at https://github.com/NHSDigital/cypkg-repository-template
you will see where these other files have come from.  The
`python-hello-world` package lists `repository-template` as a
dependency
[here](https://github.com/NHSDigital/cypkg-python-hello-world/blob/3f01c3ee4850fb4aac309a047d84ee1ebc0a67b2/.cypkg/python-hello-world/DEPENDS#L1),
and when you ran `$COYOTE index build... ` it stored the information
it needed to install the dependency for you.

### 3. Running the tests

The functional tests are in `tests/functional`.  You can run them like so:

```
 $ make test
```

Alternatively if you want more control you can run:

```
 $ make sh
 $ cd test/functional
 $ pytest
```

## The concepts

### Source repositories

A github repo that contains source we want to use, potentially in
combination with other source repositories.

### Package

Each source repository will publish a number of packages. Each package
is a released tarball; they are versioned. A source repository can
publish more than one package.

### Package list

A list of packages, and versions.

### Package index

A place to find a package list.

## Assumptions

The only assumption we make of any system that our tools execute on is
`make`, `bash`, and (for the moment) `wget`.

## Sources and packages

If you put a `.cypkg` directory in the root of a repository, that
makes it a valid source repository.

Each directory under `.cypkg` defines a package, named after the
directory: `.cypkg/my-shiny-package` defines the `my-shiny-package`
package.

If you put a `DEPENDS` file in the `.cypkg/my-shiny-package`
directory, the tools expect that to declare dependencies on other
packages.  The format is one name per line.  You can omit this file,
and the package will have no dependencies.

Similarly if you put a `CONFLICTS` file in `.cypkg/my-shiny-package`,
`coyote` will refuse to install a package to a repository that has a
package listed in this file already installed.  The format is one name
per line.  You can omit this file, and the package will have no
conflicts.

If you put a `on-install` file in `.cypkg/my-shiny-package` directory,
that file will be executed on the target after the package has been
unpacked.

The package will have an added `.CYPKG` directory at the top
level. That directory will contain a `VERSION` file so that the
filename is not constrained to contain version information, the
`DEPENDS` file, and `on-install`. This directory is not copied to the
target.

If you put a `build` file in `.cypkg/my-shiny-package`, that file will
be executed to build the tarfile. It is expected to dump a tarfile to
stdout. The default, if it is not supplied, is the equivalent of `tar
cf - --exclude=.git`.  It is *not* responsible for the `.CYPKG`
directory.

### Package versions

Version numbers are expected to sort asciibetically. Otherwise there
are no constraints.  They are stored as git tags, with an expected
pattern of `coyote-WHATEVER`.  You can pass an explicit version to
`coyote package build` if you want to build a version that is not the
last in the sorted list.

## Usage

`coyote package init my-shiny-package`

Create the `.cypkg/my-shiny-package` directory.  Call from the root of
your repository.

`coyote package new my-shiny-package`

Do a `package init`, and push the empty package as a new github repo
called `NHSDigital/cypkg-my-shiny-package`.

`coyote package delete my-shiny-package`

Delete the github repo `nhs-england-tools/cypkg-my-shiny-package`.

`coyote open`

Open the browser at the current git origin remote.

`coyote package build my-shiny-package`

Copies non-git and non-coyote files to a temporary directory, and runs
`build`.  The output has the `.CYPKG` directory added, and the
resulting tarball is zipped.

The file will be written to the current directory as
`my-shiny-package_<version>.cypkg`.

`coyote package release [version] [package...]`

Sanity checks the version, does a `coyote package build`, tags the
repo if it's not already tagged at that version and that commit is
currently checked out, pushes the tag (and if necessary any local
patches).  Pushes each `<package>_<version>.cypkg` as release to the
source repository, identified as the git `origin` remote.

`coyote install <package>`

Grabs the latest version of the package index. Fleshes out the
dependency list for the named package, building an ordered list that
are then installed without invalidating the dependency requirements of
any package installed.

`coyote apply <package-file>`

Just install the package file.  This will ignore the source
repository, and will install no dependencies.  It *will* obey
`CONFLICTS` and refuse to install if there is already a conflicting
dependency installed.

`coyote release <version> <files...>`

Tag the version and create a release on GitHub with the given files.
If the tag does not yet exist, it will be created.  The tag will be
pushed to the remote.

## Where Do We Go From Here

As it stands, the upstream repository template is monolithic, and we
want to split it up so that onboarding is easier.  The hope is that by
breaking the template into individual packages we can make it less
intimidating, but the `coyote` tool gives us an opportunity to be a bit
smarter about what code runs when and where in the project lifecycle.

The intention is that there will be a starter package for each tech
stack we might want to support.  For instance, we might have a
`python-aws-lambda-stack` package which might depend on a
`terraform-base` package.  That same `terraform-base` package might
also be a dependency of a `next-ts-aws-stack` package.  Upstream of
`terraform-base`, it could depend on `github-actions` so that it was
integrated with our preferred CD platform.

This is not the only way we could cut the problem up, but hopefully this
tool gives us the flexibility to land on a solution that keeps everyone
moving.

## Author

[Alex Young](mailto:alex.young12@nhs.net)
