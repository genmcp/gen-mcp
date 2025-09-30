#!/usr/bin/env bash

source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/common.sh"

readonly KEYCLOAK_CERTS="${REPO_ROOT}/hack/keycloak-certs"
readonly KEYCLOAK_ADMIN="admin"
readonly KEYCLOAK_ADMIN_PASSWORD="admin"
readonly KEYCLOAK_CONTAINER_NAME="keycloak"
readonly TRUSTSTORE_PASS="password"
# Set volume mount flags conditionally based on OS
# On macOS with podman, don't set the SELinux label
if [[ "$(uname -s)" == "Darwin" ]]; then
  readonly VOLUME_MOUNT_FLAGS=""
else
  readonly VOLUME_MOUNT_FLAGS=":z"
fi

function show_keycloak_help() {
  cat << EOF
Usage: $0 [OPTIONS]

Keycloak management script for development environment.

OPTIONS:
  --init [SUBJECT_ALT_NAME]       Initialize Keycloak setup (create certificates)
                                  Optional: specify subjectAltName for certificate
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
  $0 --init                           # Create certificates with default CN
  $0 --init "DNS:example.com,IP:127.0.0.1"  # Create certificates with custom SAN
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

function initialize_keycloak_setup() {
  local subject_alt_name="$1"
  header_text "Initializing Keycloak setup..."
  
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
  if [[ -n "$subject_alt_name" ]]; then
    openssl x509 -req -in "${KEYCLOAK_CERTS}/keycloak.csr" -CA "${KEYCLOAK_CERTS}/ca.crt" -CAkey "${KEYCLOAK_CERTS}/ca.key" \
      -CAcreateserial -out "${KEYCLOAK_CERTS}/keycloak.crt" -days 365 \
      -extensions v3_req -extfile <(printf "[v3_req]\nsubjectAltName=%s" "$subject_alt_name")
  else
    openssl x509 -req -in "${KEYCLOAK_CERTS}/keycloak.csr" -CA "${KEYCLOAK_CERTS}/ca.crt" -CAkey "${KEYCLOAK_CERTS}/ca.key" \
      -CAcreateserial -out "${KEYCLOAK_CERTS}/keycloak.crt" -days 365
  fi
  
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
  
  header_text "TLS certificates created in ${KEYCLOAK_CERTS}/"
}

function start_keycloak() {
  header_text "Starting Keycloak with TLS enabled..."
  
  # Check if certificates exist
  if [[ ! -f "${KEYCLOAK_CERTS}/keycloak.crt" || ! -f "${KEYCLOAK_CERTS}/keycloak.key" ]]; then
    abort "Error: TLS certificates not found. Run with --init first to create certificates."
  fi
  
  # Start Keycloak container with TLS and HTTP
  $CONTAINER_ENGINE run -d --name ${KEYCLOAK_CONTAINER_NAME} \
    -p 8443:8443 \
    -p 8081:8080 \
    -p 9000:9000 \
    -v "${KEYCLOAK_CERTS}:/opt/keycloak/conf/certs${VOLUME_MOUNT_FLAGS}" \
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
  local timeout=180
  local elapsed=0
  until curl -k -s https://localhost:9000/health/ready > /dev/null 2>&1; do
    if [[ $elapsed -ge $timeout ]]; then
      echo ""
      abort "Error: Keycloak failed to become ready within ${timeout} seconds"
    fi

    printf "."
    sleep 2
    elapsed=$((elapsed + 2))
  done
  header_text "Keycloak is ready!"
}

function stop_keycloak() {
  header_text "Stopping Keycloak..."
  $CONTAINER_ENGINE stop "${KEYCLOAK_CONTAINER_NAME}" || true
  $CONTAINER_ENGINE rm "${KEYCLOAK_CONTAINER_NAME}" || true
}

function keycloak_logs() {
  header_text "Receiving Keycloak logs..."
  $CONTAINER_ENGINE logs -f "${KEYCLOAK_CONTAINER_NAME}"
}

function add_realm() {
  local realm_name="$1"
  
  if [[ -z "$realm_name" ]]; then
    abort "Error: Realm name is required for --add-realm"
  fi
  
  header_text "Adding realm: $realm_name"
  
  # Check if container is running
  if ! $CONTAINER_ENGINE ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Add realm using Keycloak admin CLI
  $CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh create realms \
    -s realm="$realm_name" \
    -s enabled=true \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}"
    
  header_text "Realm '$realm_name' created successfully"
}

function add_client() {
  local realm_name="$1"
  local client_id="$2"
  
  if [[ -z "$realm_name" || -z "$client_id" ]]; then
    abort "Error: Both realm name and client ID are required for --add-client"
  fi
  
  header_text "Adding client '$client_id' to realm '$realm_name'"
  
  # Check if container is running
  if ! $CONTAINER_ENGINE ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Add client using Keycloak admin CLI with direct access grant enabled
  $CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
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
  
  header_text "Adding scope '$scope_name' to realm '$realm_name'"
  
  # Check if container is running
  if ! $CONTAINER_ENGINE ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Add scope using Keycloak admin CLI
  $CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
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
    
  header_text "Scope '$scope_name' created successfully in realm '$realm_name'"
}

function assign_scope() {
  local realm_name="$1"
  local client_id="$2"
  local scope_name="$3"
  
  if [[ -z "$realm_name" || -z "$client_id" || -z "$scope_name" ]]; then
    abort "Error: Realm name, client ID, and scope name are required for --assign-scope"
  fi
  
  header_text "Assigning scope '$scope_name' to client '$client_id' in realm '$realm_name'"
  
  # Check if container is running
  if ! $CONTAINER_ENGINE ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi
  
  # Get the client's internal ID
  local internal_client_id
  internal_client_id=$($CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
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
  scope_id=$($CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
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
  $CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh update clients/"$internal_client_id"/optional-client-scopes/"$scope_id" \
    -r "$realm_name" \
    --server https://localhost:8443 \
    --realm master \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}" \
    --truststore /opt/keycloak/conf/certs/truststore.jks \
    --trustpass "${TRUSTSTORE_PASS}"
    
  header_text "Scope '$scope_name' assigned successfully to client '$client_id'"
}

function disable_trusted_hosts() {
  local realm_name="$1"

  if [[ -z "$realm_name" ]]; then
    abort "Error: Realm name is required for --disable-trusted-hosts"
  fi

  header_text "Disabling trusted hosts policy for dynamic client registration in realm '$realm_name'..."

  # Check if container is running
  if ! $CONTAINER_ENGINE ps | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
    abort "Error: Keycloak container is not running. Start it with --start first."
  fi

  # Configure admin CLI credentials
  $CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh config credentials \
    --server http://localhost:8081 \
    --realm "${realm_name}" \
    --user "${KEYCLOAK_ADMIN}" \
    --password "${KEYCLOAK_ADMIN_PASSWORD}"

  # Find and delete the trusted hosts policy component
  local trusted_hosts_id
  trusted_hosts_id=$($CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
    /opt/keycloak/bin/kcadm.sh get components \
    --realm "$realm_name" \
    --query 'providerType=org.keycloak.services.clientregistration.policy.ClientRegistrationPolicy' \
    --fields id,providerId | \
     jq -r '.[] | select(.providerId=="trusted-hosts") | .id')

  if [[ -n "$trusted_hosts_id" ]]; then
    $CONTAINER_ENGINE exec "${KEYCLOAK_CONTAINER_NAME}" \
      /opt/keycloak/bin/kcadm.sh delete components/"$trusted_hosts_id" -r "$realm_name"
    header_text "Trusted hosts policy removed successfully from realm '$realm_name' (ID: $trusted_hosts_id)"
  else
    header_text "No trusted hosts policy found in realm '$realm_name' - it may already be disabled"
  fi
}
