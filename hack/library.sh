#!/usr/bin/env bash

DIR="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
ROOT_DIR="$(dirname "$DIR")"
readonly ROOT_DIR

# Simple header for logging purposes.
function header() {
  local header_text="$1"
  local line_length=${#header_text}
  local border_line=$(printf '=%.0s' $(seq 1 $((line_length + 4))))

  echo "$border_line"
  echo "| $header_text |"
  echo "$border_line"
}
