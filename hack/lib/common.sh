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

if [ -z "$CONTAINER_ENGINE" ]; then
  if command -v podman &> /dev/null; then
  		CONTAINER_ENGINE="podman"
	elif command -v docker &> /dev/null; then
		CONTAINER_ENGINE="docker"
	else
		abort "neither docker nor podman installed on system, unable to interact with containers"
	fi
fi
