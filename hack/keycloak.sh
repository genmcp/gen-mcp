#!/usr/bin/env bash

set -e
set -o pipefail

readonly REPO_ROOT="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"
readonly KEYCLOAK_CERTS="${REPO_ROOT}/hack/keycloak-certs"
readonly KEYCLOAK_ADMIN="admin"
readonly KEYCLOAK_ADMIN_PASSWORD="admin"
readonly KEYCLOAK_CONTAINER_NAME="keycloak"
readonly TRUSTSTORE_PASS="password"

if command -v docker &> /dev/null; then
	echo "using docker to interact with keycloak containers"
	CONTAINER_RUNTIME="docker"
elif command -v podman &> /dev/null; then
	echo "using podman to interact with keycloak containers"
	CONTAINER_RUNTIME="podman"
else
	echo "neither docker nor podman installed on system, unable to interact with keycloak containers"
	exit 1
fi

function abort() {
  echo "$@" > /dev/stderr
  exit 1
}

function show_help() {
  cat << EOF
Usage: $0 [OPTIONS]

Keycloak management script for development environment.

OPTIONS:
  --init                          Initialize Keycloak setup (create certificates)
  --start                         Start Keycloak container with TLS
  --stop                          Stop and remove Keycloak container
  --logs                          Show Keycloak container logs (follow mode)
  --add-realm REALM               Add a new realm to Keycloak
  --add-client REALM CLIENT       Add a new client to the specified realm
  --add-scope REALM SCOPE          Add a custom scope to the specified realm
  --assign-scope REALM CLIENT SCOPE  Assign a scope to a specific client
  --disable-trusted-hosts REALM   Disable trusted hosts policy for specified realm
  --help                          Show this help message

EXAMPLES:
  $0 --init                           # Create certificates
  $0 --start                          # Start Keycloak
  $0 --add-realm myrealm              # Add realm 'myrealm'
  $0 --add-client myrealm myclient    # Add client 'myclient' to 'myrealm'
  $0 --add-scope myrealm read         # Add scope 'read' to 'myrealm'
  $0 --assign-scope myrealm myclient read  # Assign scope 'read' to client 'myclient'
  $0 --stop                           # Stop Keycloak

NOTES:
  - Run --init first to create TLS certificates
  - Keycloak will be available at: https://localhost:8443
  - Admin console: https://localhost:8443/admin (admin/admin)
  - Health endpoint: https://localhost:9000/health
  - Use with curl: curl --cacert ${KEYCLOAK_CERTS}/ca.crt https://localhost:8443
EOF
}

function initialize_setup() {
  echo "Initializing Keycloak setup..."
  
  # Remove existing certificates directory if it exists
  rm -rf "${KEYCLOAK_CERTS}"
  
  # Create certificates directory
  mkdir -p "${KEYCLOAK_CERTS}"
  
  # Generate CA private key and certificate
  openssl genrsa -out "${KEYCLOAK_CERTS}/ca.key" 4096
  openssl req -x509 -new -key "${KEYCLOAK_CERTS}/ca.key" -out "${KEYCLOAK_CERTS}/ca.crt" -days 365 -nodes \
    -subj "/C=US/ST=State/L=City/O=Organization/OU=CA/CN=Local CA"
  
  # Generate Keycloak private key and certificate signing request
  openssl genrsa -out "${KEYCLOAK_CERTS}/keycloak.key" 4096
  openssl req -new -key "${KEYCLOAK_CERTS}/keycloak.key" -out "${KEYCLOAK_CERTS}/keycloak.csr" -nodes \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
  
  # Sign the Keycloak certificate with the CA
  openssl x509 -req -in "${KEYCLOAK_CERTS}/keycloak.csr" -CA "${KEYCLOAK_CERTS}/ca.crt" -CAkey "${KEYCLOAK_CERTS}/ca.key" \
    -CAcreateserial -out "${KEYCLOAK_CERTS}/keycloak.crt" -days 365
  
  # Clean up CSR file
  rm "${KEYCLOAK_CERTS}/keycloak.csr"
  
  # Create Java truststore from CA certificate
  keytool -import -alias ca -file "${KEYCLOAK_CERTS}/ca.crt" -keystore "${KEYCLOAK_CERTS}/truststore.jks" \
    -storepass "${TRUSTSTORE_PASS}" -noprompt
  
  # Set proper permissions for Keycloak container access
  chmod 644 "${KEYCLOAK_CERTS}/ca.crt"
  chmod 644 "${KEYCLOAK_CERTS}/ca.key"
  chmod 644 "${KEYCLOAK_CERTS}/keycloak.crt"
  chmod 644 "${KEYCLOAK_CERTS}/keycloak.key"
  chmod 644 "${KEYCLOAK_CERTS}/truststore.jks"
  
  echo "TLS certificates created in ${KEYCLOAK_CERTS}/"
}

function start_keycloak() {
  echo "Starting Keycloak with TLS enabled..."
  
  # Check if certificates exist
  if [[ ! -f "${KEYCLOAK_CERTS}/keycloak.crt" || ! -f "${KEYCLOAK_CERTS}/keycloak.key" ]]; then
    abort "Error: TLS certificates not found. Run with --init first to create certificates."
  fi
  
  # Start Keycloak container with TLS and HTTP
  $CONTAINER_RUNTIME run -d --name ${KEYCLOAK_CONTAINER_NAME} \
    -p 8443:8443 \
    -p 8080:8080 \
    -p 9000:9000 \
    -v "${KEYCLOAK_CERTS}:/opt/keycloak/conf/certs" \
    -e KC_BOOTSTRAP_ADMIN_USERNAME=${KEYCLOAK_ADMIN} \
    -e KC_BOOTSTRAP_ADMIN_PASSWORD=${KEYCLOAK_ADMIN_PASSWORD} \
    -e KC_HOSTNAME=localhost \
    -e KC_HTTPS_CERTIFICATE_FILE=/opt/keycloak/conf/certs/keycloak.crt \
    -e KC_HTTPS_CERTIFICATE_KEY_FILE=/opt/keycloak/conf/certs/keycloak.key \
    -e KC_HTTP_ENABLED=true \
    -e KC_HEALTH_ENABLED=true \
    quay.io/keycloak/keycloak:26.3 \
    start --hostname=localhost \
    --https-certificate-file=/opt/keycloak/conf/certs/keycloak.crt \
    --https-certificate-key-file=/opt/keycloak/conf/certs/keycloak.key \
    --http-enabled=true \
    --health-enabled=true

  echo "Keycloak starting with TLS at https://localhost:8443"
  echo "Admin console: https://localhost:8443/admin (${KEYCLOAK_ADMIN}/${KEYCLOAK_ADMIN_PASSWORD})"
  echo "Health endpoint: https://localhost:9000/health"
  
  echo "Waiting for Keycloak to be ready..."
  until curl -k -s https://localhost:9000/health/ready > /dev/null 2>&1; do
    printf "."
    sleep 2
  done
  echo " Keycloak is ready!"
}

function stop_keycloak() {
  echo "Stopping Keycloak..."
  $CONTAINER_RUNTIME stop "${KEYCLOAK_CONTAINER_NAME}" || true
  $CONTAINER_RUNTIME rm "${KEYCLOAK_CONTAINER_NAME}" || true
}

function keycloak_logs() {
  echo "Receiving Keycloak logs..."
  $CONTAINER_RUNTIME logs -f "${KEYCLOAK_CONTAINER_NAME}"
}

function add_realm() {
  local realm_name="$1"
  
  if [[ -z "$realm_name" ]]; then
    abort "Error: Realm name is required for --add-realm"
  fi
  
  echo "Adding realm: $realm_name"
  
  # Check if container is running
  if ! $CONTAINER_RUNTIME ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Add realm using Keycloak admin CLI
  $CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh create realms \
    -s realm="$realm_name" \
    -s enabled=true \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}"
    
  echo "Realm '$realm_name' created successfully"
}

function add_client() {
  local realm_name="$1"
  local client_id="$2"
  
  if [[ -z "$realm_name" || -z "$client_id" ]]; then
    abort "Error: Both realm name and client ID are required for --add-client"
  fi
  
  echo "Adding client '$client_id' to realm '$realm_name'"
  
  # Check if container is running
  if ! $CONTAINER_RUNTIME ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Add client using Keycloak admin CLI with direct access grant enabled
  $CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh create clients \
    -r "$realm_name" \
    -s clientId="$client_id" \
    -s enabled=true \
    -s publicClient=true \
    -s directAccessGrantsEnabled=true \
    -s 'redirectUris=["http://localhost:*"]' \
    -s 'webOrigins=["*"]' \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}"
    
  echo "Client '$client_id' created successfully in realm '$realm_name'"
}

function add_scope() {
  local realm_name="$1"
  local scope_name="$2"
  
  if [[ -z "$realm_name" || -z "$scope_name" ]]; then
    abort "Error: Both realm name and scope name are required for --add-scope"
  fi
  
  echo "Adding scope '$scope_name' to realm '$realm_name'"
  
  # Check if container is running
  if ! $CONTAINER_RUNTIME ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Add scope using Keycloak admin CLI
  $CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh create client-scopes \
    -r "$realm_name" \
    -s name="$scope_name" \
    -s description="Custom scope: $scope_name" \
    -s protocol=openid-connect \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}"
    
  echo "Scope '$scope_name' created successfully in realm '$realm_name'"
}

function assign_scope() {
  local realm_name="$1"
  local client_id="$2"
  local scope_name="$3"
  
  if [[ -z "$realm_name" || -z "$client_id" || -z "$scope_name" ]]; then
    abort "Error: Realm name, client ID, and scope name are required for --assign-scope"
  fi
  
  echo "Assigning scope '$scope_name' to client '$client_id' in realm '$realm_name'"
  
  # Check if container is running
  if ! $CONTAINER_RUNTIME ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Get the client's internal ID
  local internal_client_id
  internal_client_id=$($CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh get clients \
    -r "$realm_name" \
    --fields id,clientId \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}" | \
     jq -r ".[] | select(.clientId==\"$client_id\") | .id")
     
  if [[ -z "$internal_client_id" ]]; then
    abort "Error: Client '$client_id' not found in realm '$realm_name'"
  fi
  
  # Get the scope's internal ID
  local scope_id
  scope_id=$($CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh get client-scopes \
    -r "$realm_name" \
    --fields id,name \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}" | \
     jq -r ".[] | select(.name==\"$scope_name\") | .id")
     
  if [[ -z "$scope_id" ]]; then
    abort "Error: Scope '$scope_name' not found in realm '$realm_name'"
  fi
  
  # Assign the scope to the client as optional
  $CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh update clients/"$internal_client_id"/optional-client-scopes/"$scope_id" \
    -r "$realm_name" \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}"
    
  echo "Scope '$scope_name' assigned successfully to client '$client_id'"
}

function disable_trusted_hosts() {
  local realm_name="$1"

  if [[ -z "$realm_name" ]]; then
    abort "Error: Realm name is required for --disable-trusted-hosts"
  fi

  echo "Disabling trusted hosts policy for dynamic client registration in realm '$realm_name'..."

  # Check if container is running
  if ! $CONTAINER_RUNTIME ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi

  # Configure admin CLI credentials
  $CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh config credentials \
    --server http://localhost:8080 \
    --realm ${realm_name} \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}"

  # Find and delete the trusted hosts policy component
  local trusted_hosts_id
  trusted_hosts_id=$($CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh get components \
    --realm "$realm_name" \
    --query 'providerType=org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy' \
    --fields id,providerId | \
     jq -r '.[] | select(.providerId=="trusted-hosts") | .id')

  if [[ -n "$trusted_hosts_id" ]]; then
    $CONTAINER_RUNTIME exec "${KEYCLOAK_CONTAINER_NAME}" \
      /opt/keycloak/bin/kcadm.sh delete components/"$trusted_hosts_id" -r "$realm_name"
    echo "Trusted hosts policy removed successfully from realm '$realm_name' (ID: $trusted_hosts_id)"
  else
    echo "No trusted hosts policy found in realm '$realm_name' - it may already be disabled"
  fi
}

function main() {
  INITIALIZE=0
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
      --init) INITIALIZE=1 ;;
      --start) START=1 ;;
      --stop) STOP=1 ;;
      --logs) LOGS=1 ;;
      --add-realm) shift; REALM="$1" ;;
      --add-client) shift; CLIENT_REALM="$1"; shift; CLIENT_ID="$1" ;;
      --add-scope) shift; SCOPE_REALM="$1"; shift; SCOPE_NAME="$1" ;;
      --assign-scope) shift; ASSIGN_SCOPE_REALM="$1"; shift; ASSIGN_SCOPE_CLIENT="$1"; shift; ASSIGN_SCOPE_NAME="$1" ;;
      --disable-trusted-hosts) shift; TRUSTED_HOSTS_REALM="$1" ;;
      --help) show_help; exit 0 ;;
      *) abort "error: unknown option ${parameter}. Check the usage via --help" ;;
    esac
    shift
  done

  if [[ $INITIALIZE == 1 ]]; then
    initialize_setup
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
