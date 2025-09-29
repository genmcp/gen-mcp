#!/usr/bin/env bash

set -e
set -o pipefail

source "$(dirname "$0")/lib/common.sh"
source "$(dirname "$0")/lib/keycloak.sh"

CLUSTER_SUFFIX=${CLUSTER_SUFFIX:-"cluster.local"}
NODE_VERSION=${NODE_VERSION:-"v1.34.0"}
NODE_SHA=${NODE_SHA:-"sha256:7416a61b42b1662ca6ca89f02028ac133a309a2a30ba309614e8ec94d976dc5a"}

KEYCLOAK_SVC_NAME="external-keycloak"
KEYCLOAK_SVC_NAMESPACE="default"

# create keycloak container unless it already exists
if [ "$(docker inspect -f '{{.State.Running}}' "${KEYCLOAK_CONTAINER_NAME}" 2>/dev/null || true)" != 'true' ]; then
  header_text "No keycloak container found. Will create one..."
  "$(dirname "$0")"/keycloak.sh --init "DNS:${KEYCLOAK_CONTAINER_NAME},DNS:${KEYCLOAK_SVC_NAME},DNS:${KEYCLOAK_SVC_NAME}.${KEYCLOAK_SVC_NAMESPACE}.svc,DNS:${KEYCLOAK_SVC_NAME}.${KEYCLOAK_SVC_NAMESPACE}.svc.${CLUSTER_SUFFIX}" --start
else
  header_text "Keycloak container exists already. Skipping Keycloak setup..."
fi

# Create KinD cluster
cat <<EOF | kind create cluster --config=-
apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster

# This is needed in order to support projected volumes with service account tokens.
# See: https://kubernetes.slack.com/archives/CEKK1KTN2/p1600268272383600
kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        "service-account-issuer": "https://kubernetes.default.svc"
        "service-account-signing-key-file": "/etc/kubernetes/pki/sa.key"
    networking:
      dnsDomain: "${CLUSTER_SUFFIX}"
nodes:
- role: control-plane
  image: kindest/node:${NODE_VERSION}@${NODE_SHA}
  extraMounts:
  - hostPath: hack/keycloak-certs/ca.crt
    containerPath: /usr/local/share/ca-certificates/keycloak-ca.crt
- role: worker
  image: kindest/node:${NODE_VERSION}@${NODE_SHA}
  extraMounts:
  - hostPath: hack/keycloak-certs/ca.crt
    containerPath: /usr/local/share/ca-certificates/keycloak-ca.crt
EOF

header_text "Connecting keycloak to the cluster network if not already connected..."
if [ "$($CONTAINER_RUNTIME inspect -f='{{json .NetworkSettings.Networks.kind}}' "${KEYCLOAK_CONTAINER_NAME}")" = 'null' ]; then
  $CONTAINER_RUNTIME network connect "kind" "${KEYCLOAK_CONTAINER_NAME}"
fi

header_text "Creating service ${KEYCLOAK_SVC_NAME} in ${KEYCLOAK_SVC_NAMESPACE} namespace to expose Keycloak service..."
readonly KEYCLOAK_IP="$($CONTAINER_RUNTIME inspect "${KEYCLOAK_CONTAINER_NAME}" --format='{{(index .NetworkSettings.Networks "kind").IPAddress}}')"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: ${KEYCLOAK_SVC_NAME}
  namespace: ${KEYCLOAK_SVC_NAMESPACE}
spec:
  ports:
    - name: http
      protocol: TCP
      port: 80
      targetPort: 8080
    - name: https
      protocol: TCP
      port: 443
      targetPort: 8443
---
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: ${KEYCLOAK_SVC_NAME}-slice-1
  labels:
    # This label is the magic link! It MUST match the service name.
    kubernetes.io/service-name: ${KEYCLOAK_SVC_NAME}
addressType: IPv4
ports:
  - name: http
    protocol: TCP
    port: 8080
  - name: https
    protocol: TCP
    port: 8443
endpoints:
  - addresses:
      - "${KEYCLOAK_IP}"
    conditions:
      ready: true
EOF

header_text "Update KinD nodes to reload the CA certificates"
for no in $(kind get nodes); do
  # this is only for the nodes. The Pods need to mount the hostpath with the keycloak CA cert
  $CONTAINER_RUNTIME exec -it "$no" update-ca-certificates;
done
