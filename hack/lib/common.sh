#!/usr/bin/env bash

REPO_ROOT="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

function abort() {
  echo "$@" > /dev/stderr
  exit 1
}

header=$'\e[1;33m'
reset=$'\e[0m'
function header_text {
	echo "$header$*$reset"
}

if command -v docker &> /dev/null; then
	header_text "using docker to interact with keycloak containers"
	CONTAINER_RUNTIME="docker"
elif command -v podman &> /dev/null; then
	header_text "using podman to interact with keycloak containers"
	CONTAINER_RUNTIME="podman"
else
	header_text "neither docker nor podman installed on system, unable to interact with containers"
	exit 1
fi
