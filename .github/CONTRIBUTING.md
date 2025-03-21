# Contributing to coyote <!-- omit in toc -->

Thank you for investing your time in contributing to this project!

## When...

### I have found a problem

If you have found a problem - a bug, a defect, a flaw - please open an issue.  Please include as much information as you can, and if you can include an example of the problem, that would be even better.

### I have fixed a problem

If you have fixed a problem, this is excellent.  Please open a pull request.  Include a description of the problem and the solution, and please do make sure there's a test that demonstrates the problem and the fix.

### I have an idea for a new feature

If you have an idea for a new feature, please open an issue.  Please include as much information as you can, and if you can include an example of how you would like the feature to work, that would be even better.

### I have implemented a new feature

If you have implemented a new feature, this is also excellent.  Please open a pull request.  Include a description of the feature, and please do make sure there's a test that demonstrates the feature.  But be aware that there may need to be some discussion before it is can be merged.

## How to navigate this repository

The application is approximately structured as a Ports and Adapters architecture.  The core of the application is defined in `internal/core`.  The contents of `internal/core` have no dependencies outside of the standard library.

The `internal/adapters` directory contains the implementations of the interfaces defined in `internal/core`.  The `internal/adapters` directory may depend on third-party libraries.

The command-line interface is defined in `internal/adapters/cobra_cli`.

The `cmd` directory contains the entrypoint for the application.

There is a set of functional tests in `tests/functional`.  These tests are written in Python, and use the `pytest` library.  They exercise the built binary, and have no knowledge of the internal workings of the application.

At the root of the repository there is a `Makefile` that contains some useful commands for building and testing the application.  There are also two demo files: `demo.sh` and `demo-github.sh` that show how to use the application.  `demo.sh` is a demonstration of the complete workflow of defining packages and using them in a project.
