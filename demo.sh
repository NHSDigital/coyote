#!/bin/bash

# This is a demo of the workflow from zero to a functional application that has
# more than one package installed.
# Run it from the package root.

set -e
set -u
set -x

export PATH=$(pwd)/build/bin:$PATH

# Let's make a working directory
mkdir tmp
cd tmp

# We need a config file and an index path, otherwise nothing else works
export INDEX=$(pwd)/index
export INDEX_SRC=$(pwd)/index-src
export COYOTE_CONFIG=$(pwd)/coyoterc # This sets it for subsequent coyote invocations

# Note that the package org is set to NHSDigital, so this *will* upload packages if
# you let it.
cat > $COYOTE_CONFIG <<EOF
index = "${INDEX}"
package_org = "NHSDigital"
EOF

# Now we want a package
mkdir dependabot
(
    cd dependabot;
    git init;
    coyote package init dependabot;
    mkdir -p .github/;
    # This is the default from the repository template
    cat > .github/dependabot.yaml <<EOF
version: 2

updates:

  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "daily"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "daily"

  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "daily"

  - package-ecosystem: "pip"
    directory: "/"
    schedule:
      interval: "daily"

  - package-ecosystem: "terraform"
    directory: "/"
    schedule:
      interval: "daily"
EOF
    git add .;
    git commit -m "Initial commit";
    git tag coyote-v0.0.1;

    # Now we want to build it
    coyote package build dependabot --output ../packages;
)

# Let's have another one
mkdir editorconfig
(
    cd editorconfig;
    git init;
    coyote package init editorconfig;
    mkdir -p .github/;
    # This is the default from the repository template
    cat > .editorconfig <<EOF
root = true

[*]
charset = utf-8
end_of_line = lf
indent_size = 2
indent_style = space
insert_final_newline = true
trim_trailing_whitespace = true

[*.md]
indent_size = unset

[*.py]
indent_size = 4

[{Dockerfile,Dockerfile.}*]
indent_size = 4

[{Makefile,*.mk,go.mod,go.sum,*.go,.gitmodules}]
indent_style = tab
EOF
    git add .;
    git commit -m "Initial commit";
    git tag coyote-v0.0.1;

    # Now we want to build it
    coyote package build editorconfig --output ../packages;
)

# Let's have something a little more interesting: a python script that runs in docker
mkdir python-docker
(
    cd python-docker;
    git init;
    coyote package init python-docker;
    mkdir -p .github/;
    # This is the default from the repository template
    cat > Dockerfile <<EOF
FROM python:3.12-slim-buster

WORKDIR /app
COPY ./requirements.txt /app/requirements.txt
RUN pip install -r requirements.txt

COPY ./main.py /app/main.py
CMD ["python", "main.py"]
EOF

    touch requirements.txt

    cat > main.py <<EOF
print("Hello, world!")
EOF

    git add .;
    git commit -m "Initial commit";
    git tag coyote-v0.0.1;

    # Now we want to build it
    coyote package build python-docker --output ../packages;
)

# Now we want a metapackage that depends on all three that we've built
mkdir python-app
(
    cd python-app;
    git init;
    coyote package init python-app;
    cat > .cypkg/python-app/DEPENDS <<EOF
dependabot
editorconfig
python-docker
EOF
    git add .;
    git commit -m "Initial commit";
    git tag coyote-v0.0.1;

    # Now we want to build it
    coyote package build python-app --output ../packages;
)

# NOW we build the index
find packages -name '*.cypkg' > $INDEX_SRC
coyote index build $INDEX_SRC $INDEX

# Ok, now we can make our new project based on the python-app package
coyote init python-app my-python-app