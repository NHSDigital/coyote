#!/bin/bash

# This is a demo of the typical workflow using the github-published packages and index.
# Run it from the repo root.

set -e
set -u
set -x
set -o pipefail

export PATH=$(pwd)/build/bin:$PATH

# Our project is going to be called my-new-project
export PROJECT=my-new-project

# Let's make a working directory
mkdir -p tmp
cd tmp

rm -rf $PROJECT

# We need a config file and an index path, otherwise nothing else works
export INDEX="https://github.com/NHSDigital/coyote-index/releases/latest/download/index"
export COYOTE_CONFIG=$(pwd)/coyoterc # This sets it for subsequent coyote invocations

# Note that the package org is set to NHSDigital, so this *will* upload packages if
# you let it.
cat > $COYOTE_CONFIG <<EOF
index = "${INDEX}"
package_org = "NHSDigital"
EOF

coyote init template-readme $PROJECT --config $COYOTE_CONFIG