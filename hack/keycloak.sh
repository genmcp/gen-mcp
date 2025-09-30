#!/usr/bin/env bash

set -e
set -o pipefail

source "$(dirname "$0")/lib/keycloak.sh"

function main() {
  INITIALIZE=0
  SUBJECT_ALT_NAME=""
  START=0
  STOP=0
  LOGS=0
  REALM=""
  CLIENT_REALM=""
  CLIENT_ID=""
  SCOPE_REALM=""
  SCOPE_NAME=""
  ASSIGN_SCOPE_REALM=""
  ASSIGN_SCOPE_CLIENT=""
  ASSIGN_SCOPE_NAME=""
  DISABLE_TRUSTED_HOSTS=0
  TRUSTED_HOSTS_REALM=""

  while [[ $# -ne 0 ]]; do
    parameter=$1
    case ${parameter} in
      --init)
        INITIALIZE=1
        # Check if the next argument exists and doesn't start with --
        if [[ $# -gt 1 && "$2" != --* ]]; then
          shift
          SUBJECT_ALT_NAME="$1"
        fi
        ;;
      --start) START=1 ;;
      --stop) STOP=1 ;;
      --logs) LOGS=1 ;;
      --add-realm) shift; REALM="$1" ;;
      --add-client) shift; CLIENT_REALM="$1"; shift; CLIENT_ID="$1" ;;
      --add-scope) shift; SCOPE_REALM="$1"; shift; SCOPE_NAME="$1" ;;
      --assign-scope) shift; ASSIGN_SCOPE_REALM="$1"; shift; ASSIGN_SCOPE_CLIENT="$1"; shift; ASSIGN_SCOPE_NAME="$1" ;;
      --disable-trusted-hosts) shift; TRUSTED_HOSTS_REALM="$1" ;;
      --help) show_keycloak_help; exit 0 ;;
      *) abort "error: unknown option ${parameter}. Check the usage via --help" ;;
    esac
    shift
  done

  if [[ $INITIALIZE == 1 ]]; then
    initialize_keycloak_setup "$SUBJECT_ALT_NAME"
  fi

  if [[ $START == 1 ]]; then
    start_keycloak
  fi

  if [[ $LOGS == 1 ]]; then
    keycloak_logs
  fi

  if [[ -n "$REALM" ]]; then
    add_realm "$REALM"
  fi

  if [[ -n "$CLIENT_REALM" && -n "$CLIENT_ID" ]]; then
    add_client "$CLIENT_REALM" "$CLIENT_ID"
  fi

  if [[ -n "$SCOPE_REALM" && -n "$SCOPE_NAME" ]]; then
    add_scope "$SCOPE_REALM" "$SCOPE_NAME"
  fi

  if [[ -n "$ASSIGN_SCOPE_REALM" && -n "$ASSIGN_SCOPE_CLIENT" && -n "$ASSIGN_SCOPE_NAME" ]]; then
    assign_scope "$ASSIGN_SCOPE_REALM" "$ASSIGN_SCOPE_CLIENT" "$ASSIGN_SCOPE_NAME"
  fi

  if [[ -n "$TRUSTED_HOSTS_REALM" ]]; then
    disable_trusted_hosts "$TRUSTED_HOSTS_REALM"
  fi

  if [[ $STOP == 1 ]]; then
    stop_keycloak
  fi
}

main "$@"
