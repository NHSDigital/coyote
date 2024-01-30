These runners are small scripts that exercise adapters where we don't want to do that
in tests.  For instance, `github/github_main.go` will create a new repository and upload
it.  We don't want to put that in our test loop.

They need to be in their own directories so that the go compiler doesn't have a fit about
more than one `main` function in package `main`.