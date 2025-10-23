#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "usage: exec_dns_in_pod.sh <namespace> <server> <name> <type>" >&2
  exit 64
fi

namespace="$1"
server="$2"
record_name="$3"
record_type="$4"

pod_name="dnsquery-$(date +%s%3N)"
manifest_file=""

cleanup() {
  if [ -n "$manifest_file" ] && [ -f "$manifest_file" ]; then
    rm -f "$manifest_file"
  fi
  oc -n "$namespace" delete pod "$pod_name" --ignore-not-found >/dev/null 2>&1 || true
}
trap cleanup EXIT

manifest_file="$(mktemp)"
cat >"$manifest_file" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${pod_name}
  namespace: ${namespace}
spec:
  restartPolicy: Never
  securityContext:
    runAsNonRoot: true
  containers:
  - name: dnsquery
    image: registry.redhat.io/openshift4/network-tools-rhel9:latest
    command: ["sh", "-c", "dig @${server} ${record_name} ${record_type} +noall +answer"]
    securityContext:
      runAsNonRoot: true
      runAsUser: 1000
      allowPrivilegeEscalation: false
      seccompProfile:
        type: RuntimeDefault
      capabilities:
        drop:
        - "ALL"
EOF

oc -n "$namespace" apply -f "$manifest_file" >/dev/null
rm -f "$manifest_file"

oc -n "$namespace" wait --for=condition=PodScheduled "pod/$pod_name" --timeout=30s >/dev/null 2>&1 || true
oc -n "$namespace" wait --for=condition=Ready "pod/$pod_name" --timeout=90s >/dev/null 2>&1 || true
oc -n "$namespace" logs -f "$pod_name" || oc -n "$namespace" logs "$pod_name"
