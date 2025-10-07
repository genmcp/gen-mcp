#!/usr/bin/env bash

# exit on error
set -e

source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")"/library.sh

header "Running mcpfile schema generator"
pushd "$ROOT_DIR/hack/jsonschemagen" > /dev/null
go run main.go
popd > /dev/null

header "Finished generating code"
