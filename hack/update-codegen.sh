#!/usr/bin/env bash

set -e
set -o pipefail

source "$(dirname "$0")/lib/common.sh"

header_text "Running mcpfile schema generator"
pushd "$REPO_ROOT/hack/jsonschemagen" > /dev/null
go run main.go
popd > /dev/null

header_text "Finished generating code"
