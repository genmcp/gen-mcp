#!/usr/bin/env bash
set -euo pipefail

# NetEdge ingress break/repair helper
# Supports four reversible ingress/network scenarios drawn from NET_DIAGNOSTIC_SCENARIOS.md

NAMESPACE="${NAMESPACE:-test-ingress}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"

SCENARIO2_BAD_HOST="does-not-exist.netedge.test"
SCENARIO2_ORIG_HOST_ANNOTATION="netedge-tools-original-host"
SCENARIO3_POLICY_NAME="netedge-deny-router"
CURRENT_ROUTE_HOST=""
SCENARIO4_TLS_ANNOTATION="netedge-tools-original-tls"

log() { printf '%s\n' "$*" >&2; }

need_oc() {
  command -v oc >/dev/null 2>&1 || { log "oc not found in path"; exit 127; }
}

ensure_ns() {
  if ! oc get ns "${NAMESPACE}" >/dev/null 2>&1; then
    log "creating namespace ${NAMESPACE}"
    oc create ns "${NAMESPACE}"
  fi
}

deploy_healthy() {
  ensure_ns
  log "applying deployment, service and route in ${NAMESPACE}"
  cat <<'YAML' | APP_NAME="${APP_NAME}" APP_LABEL="${APP_LABEL}" IMAGE="${IMAGE}" PORT="${PORT}" envsubst | oc -n "${NAMESPACE}" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${APP_NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${APP_LABEL}
  template:
    metadata:
      labels:
        app: ${APP_LABEL}
    spec:
      containers:
      - name: ${APP_NAME}
        image: ${IMAGE}
        ports:
        - containerPort: ${PORT}
---
apiVersion: v1
kind: Service
metadata:
  name: ${APP_NAME}
spec:
  selector:
    app: ${APP_LABEL}
  ports:
  - name: http
    port: ${PORT}
    targetPort: ${PORT}
---
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: ${APP_NAME}
spec:
  to:
    kind: Service
    name: ${APP_NAME}
  port:
    targetPort: http
YAML

  log "waiting for deployment to be available"
  oc -n "${NAMESPACE}" rollout status deploy/"${APP_NAME}" --timeout=120s

  log "waiting for endpoints to be populated"
  oc -n "${NAMESPACE}" wait --for=jsonpath='{.subsets[0].addresses[0].ip}' endpoints/"${APP_NAME}" --timeout=120s

  show_status
  refresh_route_host
}

ensure_workload() {
  if ! oc -n "${NAMESPACE}" get svc "${APP_NAME}" >/dev/null 2>&1; then
    log "baseline objects missing; deploying healthy baseline first"
    deploy_healthy
  fi
}

show_status() {
  log "route summary"
  oc -n "${NAMESPACE}" get route "${APP_NAME}" -o custom-columns=NAME:.metadata.name,HOST:.spec.host,ADMITTED:'{.status.ingress[0].conditions[?(@.type=="Admitted")].status}' --no-headers || true

  local tls_term
  tls_term="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.tls.termination}' 2>/dev/null || true)"
  if [ -n "${tls_term}" ]; then
    log "route TLS termination: ${tls_term}"
  else
    log "route TLS termination: (none)"
  fi

  log "service selector"
  oc -n "${NAMESPACE}" get svc "${APP_NAME}" -o jsonpath='{.spec.selector}{"\n"}' || true

  log "endpoints detail"
  oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o yaml || true

  if oc -n "${NAMESPACE}" get networkpolicy "${SCENARIO3_POLICY_NAME}" >/dev/null 2>&1; then
    log "networkpolicy ${SCENARIO3_POLICY_NAME} status"
    oc -n "${NAMESPACE}" get networkpolicy "${SCENARIO3_POLICY_NAME}" -o yaml || true
  fi
}

refresh_route_host() {
  local status_host spec_host chosen_host
  status_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.status.ingress[0].host}' 2>/dev/null || true)"
  spec_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
  if [ -n "${status_host}" ]; then
    chosen_host="${status_host}"
  else
    chosen_host="${spec_host}"
  fi
  if [ -n "${chosen_host}" ]; then
    log "current route host from cluster: ${chosen_host}"
    CURRENT_ROUTE_HOST="${chosen_host}"
  else
    log "could not determine current route host from cluster"
    CURRENT_ROUTE_HOST=""
  fi
}

scenario1_break() {
  ensure_workload
  log "breaking service selector to create empty endpoints"
  oc -n "${NAMESPACE}" patch svc "${APP_NAME}" --type=merge -p '{"spec":{"selector":{"app":"broken-mismatch"}}}'
  log "waiting for endpoints to become empty"
  for _ in $(seq 1 24); do
    if ! oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o jsonpath='{.subsets}' | grep -q '[^[:space:]]'; then
      log "endpoints are empty"
      break
    fi
    sleep 5
  done
  show_status
  refresh_route_host
  log "note: route should now return 503 due to no backends"
}

scenario1_repair() {
  ensure_workload
  log "restoring service selector to match deployment labels"
  oc -n "${NAMESPACE}" patch svc "${APP_NAME}" --type=merge -p '{"spec":{"selector":{"app":"'"${APP_LABEL}"'"}}}'
  log "waiting for endpoints to repopulate"
  oc -n "${NAMESPACE}" wait --for=jsonpath='{.subsets[0].addresses[0].ip}' endpoints/"${APP_NAME}" --timeout=120s
  show_status
  refresh_route_host
  log "note: route should now succeed"
}

scenario2_break() {
  ensure_workload
  local stored_host
  local annotation_query
  annotation_query="{.metadata.annotations['${SCENARIO2_ORIG_HOST_ANNOTATION}']}"
  stored_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o "jsonpath=${annotation_query}" 2>/dev/null || true)"
  if [ -z "${stored_host}" ]; then
    stored_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.host}' 2>/dev/null || true)"
  fi
  if [ -z "${stored_host}" ]; then
    stored_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.status.ingress[0].host}' 2>/dev/null || true)"
  fi
  if [ -z "${stored_host}" ]; then
    log "could not determine current route host to store"
    exit 1
  fi
  log "patching route host to trigger NXDOMAIN, original host: ${stored_host}"
  local patch
  patch=$(printf '{"spec":{"host":"%s"},"metadata":{"annotations":{"%s":"%s"}}}' "${SCENARIO2_BAD_HOST}" "${SCENARIO2_ORIG_HOST_ANNOTATION}" "${stored_host}")
  oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=merge -p "${patch}"
  show_status
  refresh_route_host
  log "note: external DNS lookups for ${SCENARIO2_BAD_HOST} should now fail with NXDOMAIN"
}

scenario2_repair() {
  ensure_workload
  local stored_host
  local annotation_query
  annotation_query="{.metadata.annotations['${SCENARIO2_ORIG_HOST_ANNOTATION}']}"
  stored_host="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o "jsonpath=${annotation_query}" 2>/dev/null || true)"
  if [ -z "${stored_host}" ]; then
    log "original host annotation ${SCENARIO2_ORIG_HOST_ANNOTATION} not found; cannot repair automatically"
    exit 1
  fi
  log "restoring route host to ${stored_host}"
  local patch
  patch=$(printf '{"spec":{"host":"%s"}}' "${stored_host}")
  oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=merge -p "${patch}"
  show_status
  refresh_route_host
  log "note: route host restored; DNS should resolve again if reachable from this environment"
}

scenario3_break() {
  ensure_workload
  log "applying default-deny ingress NetworkPolicy ${SCENARIO3_POLICY_NAME}"
  cat <<YAML | oc -n "${NAMESPACE}" apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ${SCENARIO3_POLICY_NAME}
  labels:
    app.kubernetes.io/managed-by: netedge-tools
spec:
  podSelector: {}
  policyTypes:
  - Ingress
YAML
  show_status
  refresh_route_host
  log "note: router traffic should now be blocked by the NetworkPolicy"
}

scenario3_repair() {
  log "removing NetworkPolicy ${SCENARIO3_POLICY_NAME}"
  oc -n "${NAMESPACE}" delete networkpolicy "${SCENARIO3_POLICY_NAME}" --ignore-not-found
  show_status
  refresh_route_host
  log "note: router traffic should now be allowed"
}

scenario4_break() {
  ensure_workload
  log "patching Route to use reencrypt TLS (backend remains plain HTTP) to force router/backend TLS handshake failures"
  oc -n "${NAMESPACE}" annotate route "${APP_NAME}" "${SCENARIO4_TLS_ANNOTATION}=plain-http" --overwrite >/dev/null 2>&1 || true

  local patch_file
  patch_file="$(mktemp)"
  cat <<'JSON' > "${patch_file}"
{"spec":{"tls":{"termination":"reencrypt","insecureEdgeTerminationPolicy":"Redirect","destinationCACertificate":"-----BEGIN CERTIFICATE-----\nMIICwjCCAaqgAwIBAgIUPdPBiiv4lqmx+J/0VDP0FmSkL5AwDQYJKoZIhvcNAQEL\nBQAwFDESMBAGA1UEAwwJbmV0ZWRnZS1jYTAeFw0yNDAxMDEwMDAwMDBaFw0zNDAx\nMDEwMDAwMDBaMBQxEjAQBgNVBAMMCW5ldGVkZ2UtY2EwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQDJF5vQ2p8sVh6a4l4BhFJj3ZdtmLEl3gHtP3nTzmYB\njLfzMOlHFelgW8Trgzzwf1JmHz1DScz3H1JdYaiN2rFbV2jwV3X7hJ2P60+g2PdV\ng9gi3b3tu5C3fYpt3YaKulHytQ+Oqv5x0zDfai+qyDCqgQzfwzEkbg0AoLvvy0F9\nWPzPj6xnEwSa5t1FUbYQwplpHk2nw/TYe8LSF+6T1Vn/ZRQlBIjXopB11Mh17l5b\n7ouC2vhV3ZNiHPJJR7MY3GacsbGu9zwCfn12MBYc4B1YwDWwP6R6jo5fKUn19+ih\nq6B4Tn3FQilds3n3qNNc3blJPoBsXSV6XdJ+2l4O/E/1AgMBAAGjUzBRMB0GA1Ud\nDgQWBBSmU/lVYG5HZM3Xq2bpuASr9PBRGjAfBgNVHSMEGDAWgBSmU/lVYG5HZM3X\nq2bpuASr9PBRGjAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCH\n0W7jhBP6tQyctXWZfqs169uAHSrmwU0aQAn2j2ad3rZ4IE+u2ASfrmVii5wjNe4E\ntRjS0+QWaEoZwJBGtJ6r/TqlTr/7zQa4XzdX8yEbfL33Gt20lfO1nU/KcqbPjhQi\n6JFpj0YvdwfukuvseIgCyPCYTDWJrrRJbwI62I5qnK0iePvyplTkU1qpLVgv2wP6\nUZUqXYy3eLKUIKwoJs5eX1Po9rIrFHG8wbHPJUBaVuZQgDrw9WdDPQ6YgB1nc8fN\nWHGakUWTynVro4/+Y/5Dn09iuPhR2JrDogFm6uaSYzBeAnkw1GOEiUv7J8wo6P0R\nQrJjapdStc3bVh7usNuh\n-----END CERTIFICATE-----\n"}}}
JSON

  oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=merge --patch-file "${patch_file}"
  rm -f "${patch_file}"

  show_status
  refresh_route_host
  log "note: router now expects TLS reencrypt to backend but pods still serve plain HTTP → handshake fails and the Route returns 503"
  log "tip: use query_prometheus to inspect haproxy_server_ssl_verify_result_total for this route to quantify TLS failures"
}

scenario4_repair() {
  log "restoring Route TLS configuration to plain HTTP"
  local tls_term
  tls_term="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.tls.termination}' 2>/dev/null || true)"
  if [ -n "${tls_term}" ]; then
    oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=json -p '[{"op":"remove","path":"/spec/tls"}]' || true
  fi
  oc -n "${NAMESPACE}" annotate route "${APP_NAME}" "${SCENARIO4_TLS_ANNOTATION}-" >/dev/null 2>&1 || true
  show_status
  refresh_route_host
  log "note: Route reverted to HTTP; TLS handshake failures should cease"
}

route_url() {
  refresh_route_host
  if [ -n "${CURRENT_ROUTE_HOST}" ]; then
    printf 'http://%s\n' "${CURRENT_ROUTE_HOST}"
  fi
}

curl_route() {
  refresh_route_host
  if [ -z "${CURRENT_ROUTE_HOST}" ]; then
    log "could not determine route host"
    return 1
  fi
  local tls_term
  tls_term="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.tls.termination}' 2>/dev/null || true)"
  local scheme="http"
  local curl_opts=(-fsS)
  if [ -n "${tls_term}" ]; then
    scheme="https"
    curl_opts+=(-k)
  fi
  local url="${scheme}://${CURRENT_ROUTE_HOST}"
  log "curling ${url}"
  curl "${curl_opts[@]}" "${url}" | head -n 3 || { log "curl failed or route not resolvable from here"; return 1; }
}

cleanup_workload() {
  log "deleting deployment, service and route in ${NAMESPACE}"
  oc -n "${NAMESPACE}" delete route,svc,deploy "${APP_NAME}" --ignore-not-found
  log "namespace retained to avoid surprises, delete manually if desired"
}

scenario_cleanup() {
  cleanup_workload
  if oc -n "${NAMESPACE}" get networkpolicy "${SCENARIO3_POLICY_NAME}" >/dev/null 2>&1; then
    oc -n "${NAMESPACE}" delete networkpolicy "${SCENARIO3_POLICY_NAME}" --ignore-not-found
  fi
  oc -n "${NAMESPACE}" annotate route "${APP_NAME}" "${SCENARIO4_TLS_ANNOTATION}-" >/dev/null 2>&1 || true
  CURRENT_ROUTE_HOST=""
}

scenario_name() {
  case "$1" in
    1) printf 'Route → Service selector mismatch';;
    2) printf 'Route host without DNS';;
    3) printf 'Router blocked by NetworkPolicy';;
    4) printf 'Route reencrypt without backend TLS';;
    *) printf 'unknown';;
  esac
}

scenario_setup() {
  case "$1" in
    1|2)
      deploy_healthy
      ;;
    3)
      deploy_healthy
      oc -n "${NAMESPACE}" delete networkpolicy "${SCENARIO3_POLICY_NAME}" --ignore-not-found >/dev/null 2>&1 || true
      ;;
    4)
      deploy_healthy
      local tls_term
      tls_term="$(oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='{.spec.tls.termination}' 2>/dev/null || true)"
      if [ -n "${tls_term}" ]; then
        oc -n "${NAMESPACE}" patch route "${APP_NAME}" --type=json -p '[{"op":"remove","path":"/spec/tls"}]' || true
      fi
      oc -n "${NAMESPACE}" annotate route "${APP_NAME}" "${SCENARIO4_TLS_ANNOTATION}-" >/dev/null 2>&1 || true
      ;;
    *)
      log "unsupported scenario $1"
      exit 1
      ;;
  esac
}

scenario_break() {
  case "$1" in
    1) scenario1_break ;;
    2) scenario2_break ;;
    3) scenario3_break ;;
    4) scenario4_break ;;
    *) log "unsupported scenario $1"; exit 1 ;;
  esac
}

scenario_repair() {
  case "$1" in
    1) scenario1_repair ;;
    2) scenario2_repair ;;
    3) scenario3_repair ;;
    4) scenario4_repair ;;
    *) log "unsupported scenario $1"; exit 1 ;;
  esac
}

scenario_status() {
  ensure_workload
  show_status
  refresh_route_host
}

remind_scenario() {
  log "Reminder: include --scenario=$1 on follow-up commands to stay in the same scenario"
}

usage() {
  cat <<'HELP'
usage: netedge-break-repair.sh [--scenario=<1|2|3|4>] [--setup|--break|--repair|--status|--curl|--cleanup]

scenarios:
  1  Route → Service selector mismatch (default)
  2  Route host without DNS record (NXDOMAIN)
  3  NetworkPolicy blocking router → service traffic
  4  Route reencrypt without backend TLS

actions:
  --setup    deploy healthy baseline objects for the chosen scenario
  --break    introduce the scenario-specific failure condition
  --repair   restore the healthy state for the chosen scenario
  --status   show route, service selector, endpoints (and policy) state
  --curl     curl the route host from this machine (best-effort)
  --cleanup  remove route/service/deployment (and policy if present)

env vars (optional overrides):
  NAMESPACE  target namespace (default: test-ingress)
  APP_NAME   base name for deployment, service and route (default: hello)
  APP_LABEL  label used by deployment and service selector (default: hello)
  IMAGE      container image (default: quay.io/openshift/origin-hello-openshift:latest)
  PORT       container and service port (default: 8080)
HELP
}

main() {
  need_oc
  local scenario="1"
  local action=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --scenario=*)
        scenario="${1#*=}"
        ;;
      --scenario)
        shift || { log "--scenario requires a value"; exit 1; }
        scenario="${1:-}"
        if [ -z "${scenario}" ]; then
          log "--scenario requires a value"
          exit 1
        fi
        ;;
      --setup|--break|--repair|--status|--curl|--cleanup)
        if [ -n "${action}" ]; then
          log "only one action may be specified per invocation"
          exit 1
        fi
        action="${1}"
        ;;
      --help|-h)
        usage
        return 0
        ;;
      *)
        log "unknown argument: ${1}"
        usage
        exit 1
        ;;
    esac
    shift
  done

  if [ -z "${action}" ]; then
    usage
    exit 1
  fi

  case "${scenario}" in
    1|2|3|4)
      ;;
    *)
      log "unsupported scenario ${scenario}; choose 1, 2, 3 or 4"
      exit 1
      ;;
  esac

  log "scenario ${scenario}: $(scenario_name "${scenario}")"

  case "${action}" in
    --setup)
      scenario_setup "${scenario}"
      ;;
    --break)
      scenario_break "${scenario}"
      ;;
    --repair)
      scenario_repair "${scenario}"
      ;;
    --status)
      scenario_status "${scenario}"
      ;;
    --curl)
      curl_route
      ;;
    --cleanup)
      scenario_cleanup
      ;;
  esac

  remind_scenario "${scenario}"
}

main "$@"
