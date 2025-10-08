#!/usr/bin/env bash
set -euo pipefail

# simple, safe, reversible ingress breakage scenario
# scenario 1: service selector mismatch leads to empty endpoints and router 503s

# prefs
NAMESPACE="${NAMESPACE:-test-ingress}"
APP_NAME="${APP_NAME:-hello}"
APP_LABEL="${APP_LABEL:-hello}"
IMAGE="${IMAGE:-quay.io/openshift/origin-hello-openshift:latest}"
PORT="${PORT:-8080}"

# helpers

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
  cat <<'EOF' | APP_NAME="${APP_NAME}" APP_LABEL="${APP_LABEL}" IMAGE="${IMAGE}" PORT="${PORT}" envsubst | oc -n "${NAMESPACE}" apply -f -
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
EOF

  log "waiting for deployment to be available"
  oc -n "${NAMESPACE}" rollout status deploy/"${APP_NAME}" --timeout=120s

  log "waiting for endpoints to be populated"
  oc -n "${NAMESPACE}" wait --for=jsonpath='{.subsets[0].addresses[0].ip}' endpoints/"${APP_NAME}" --timeout=120s

  show_status
}

show_status() {
  log "route summary"
  oc -n "${NAMESPACE}" get route "${APP_NAME}" -o custom-columns=NAME:.metadata.name,HOST:.spec.host,ADMITTED:'{.status.ingress[0].conditions[?(@.type=="Admitted")].status}' --no-headers || true

  log "service selector"
  oc -n "${NAMESPACE}" get svc "${APP_NAME}" -o jsonpath='{.spec.selector}{"\n"}' || true

  log "endpoints detail"
  oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o yaml || true
}

break_selector() {
  log "breaking service selector to create empty endpoints"
  oc -n "${NAMESPACE}" patch svc "${APP_NAME}" --type=merge -p '{"spec":{"selector":{"app":"broken-mismatch"}}}'
  log "waiting for endpoints to become empty"
  # wait loop since oc wait cannot express empty subsets directly
  for i in $(seq 1 24); do
    if ! oc -n "${NAMESPACE}" get endpoints "${APP_NAME}" -o jsonpath='{.subsets}' | grep -q '[^[:space:]]'; then
      log "endpoints are empty"
      break
    fi
    sleep 5
  done
  show_status
  log "note: route should now return 503 due to no backends"
}

repair_selector() {
  log "restoring service selector to match deployment labels"
  oc -n "${NAMESPACE}" patch svc "${APP_NAME}" --type=merge -p '{"spec":{"selector":{"app":"'"${APP_LABEL}"'"}}}'
  log "waiting for endpoints to repopulate"
  oc -n "${NAMESPACE}" wait --for=jsonpath='{.subsets[0].addresses[0].ip}' endpoints/"${APP_NAME}" --timeout=120s
  show_status
  log "note: route should now succeed"
}

route_url() {
  oc -n "${NAMESPACE}" get route "${APP_NAME}" -o jsonpath='http://{.spec.host}{"\n"}'
}

curl_route() {
  url="$(route_url || true)"
  if [ -n "${url:-}" ]; then
    log "curling ${url}"
    curl -fsS "${url}" | head -n 3 || { log "curl failed or route not resolvable from here"; return 1; }
  else
    log "could not determine route host"
  fi
}

cleanup() {
  log "deleting objects in ${NAMESPACE}"
  oc -n "${NAMESPACE}" delete route,svc,deploy "${APP_NAME}" --ignore-not-found
  log "namespace retained to avoid surprises, delete manually if desired"
}

usage() {
  cat <<EOF
usage: $(basename "$0") [--setup|--break|--repair|--status|--curl|--cleanup]

  --setup   create namespace if needed and deploy healthy app, service and route
  --break   patch service selector to nonmatching value leading to empty endpoints
  --repair  restore service selector to match deployment labels
  --status  show route host, service selector and endpoints detail
  --curl    attempt a curl to the route host from this machine
  --cleanup delete the app, service and route in the namespace

env vars:
  NAMESPACE  target namespace default test-ingress
  APP_NAME   base name for deployment, service and route default hello
  APP_LABEL  label used by deployment and service selector default hello
  IMAGE      container image default quay.io/openshift/origin-hello-openshift:latest
  PORT       container and service port default 8080
EOF
}

main() {
  need_oc
  case "${1:-}" in
    --setup) deploy_healthy ;;
    --break) break_selector ;;
    --repair) repair_selector ;;
    --status) show_status ;;
    --curl) curl_route ;;
    --cleanup) cleanup ;;
    *) usage ;;
  esac
}

main "$@"
