# NetEdge Break/Repair Script

`netedge-break-repair.sh` stages, breaks, and repairs the deterministic ingress and DNS scenarios captured in [`NET_DIAGNOSTIC_SCENARIOS.md`](./NET_DIAGNOSTIC_SCENARIOS.md). It deploys a minimal application stack (Deployment, Service, Route), introduces the chosen fault, and restores the healthy baseline when asked. Scenario 4 flips the Route into `reencrypt` mode without upgrading the backend so router ↔ pod TLS handshakes fail.

## Prerequisites

- `oc` CLI available in `$PATH`
- `envsubst` (from GNU `gettext`) available for templating manifests
- Credentials that allow creating resources in the target namespace
- Ability to pull `quay.io/openshift/origin-hello-openshift:latest` (default demo image; override with `IMAGE` if restricted)

## Basic Usage

```bash
examples/netedge-tools/netedge-break-repair.sh [--scenario=<1|2|3|4>] <action>
```

Actions:

- `--setup` – Deploy the healthy baseline (Deployment, Service, Route)
- `--break` – Apply the scenario-specific failure
- `--repair` – Restore the healthy state for the scenario
- `--status` – Show Route, Service, Endpoints (and NetworkPolicy) details
- `--curl` – Curl the admitted Route hostname (best effort)
- `--cleanup` – Remove the created resources and any managed NetworkPolicy

If `--scenario` is omitted the script defaults to scenario **1**. Always reuse the same `--scenario=N` flag on follow-up commands; the script prints a reminder after each action. Scenario **4** assumes the workload is still serving plain HTTP so the router’s reencrypt attempt will fail.

## Scenarios

1. **Route → Service selector mismatch** – Patches the Service selector to a non-matching value so no endpoints remain (router returns 503). Repair restores the correct selector.
   - Agent prompt: “On the current cluster we exposed an app through a Route but it keeps failing. Diagnose the root cause and tell me how to fix it.”
2. **Route host without DNS record** – Stores the original host, then patches `spec.host` to an NXDOMAIN value. Repair restores the saved host from annotation.
   - Agent prompt: “On the current cluster the Route’s hostname never resolves in DNS even though the Service and Pods look healthy. Diagnose the root cause and tell me how to fix it.”
3. **NetworkPolicy blocking router traffic** – Applies a default-deny ingress NetworkPolicy in the namespace. Repair deletes the policy.
   - Agent prompt: “On the current cluster every request to the Route now times out even though the pods and service look healthy. Diagnose the root cause and tell me how to fix it.”
4. **Route reencrypt without backend TLS** – Patches the Route to `termination: reencrypt` while the workload continues to serve plain HTTP. Router ↔ backend TLS handshakes fail, producing 503s until the TLS block is removed. Repair deletes the TLS stanza.
   - Agent prompt: “On the current cluster the Route started returning 503s right after someone enabled reencrypt termination. Diagnose the root cause and tell me how to fix it.”
   - After `--break`, use `--curl` (HTTPS with `-k`) to see the router return 503 while handshakes fail.

## Environment Overrides

Export any of these before running the script to change defaults:

- `NAMESPACE` – Target namespace (default: `test-ingress`)
- `APP_NAME` – Base name for Deployment/Service/Route (default: `hello`)
- `APP_LABEL` – Label shared by Deployment and Service (default: `hello`)
- `IMAGE` – Demo image (default: `quay.io/openshift/origin-hello-openshift:latest`)
- `PORT` – Container and Service port (default: `8080`)

## Example Workflow

```bash
# Scenario 1: selector mismatch
examples/netedge-tools/netedge-break-repair.sh --scenario=1 --setup
examples/netedge-tools/netedge-break-repair.sh --scenario=1 --break
examples/netedge-tools/netedge-break-repair.sh --scenario=1 --status
examples/netedge-tools/netedge-break-repair.sh --scenario=1 --repair

# Scenario 2: NXDOMAIN host
examples/netedge-tools/netedge-break-repair.sh --scenario=2 --setup
examples/netedge-tools/netedge-break-repair.sh --scenario=2 --break
examples/netedge-tools/netedge-break-repair.sh --scenario=2 --curl
examples/netedge-tools/netedge-break-repair.sh --scenario=2 --repair

# Scenario 3: NetworkPolicy block
examples/netedge-tools/netedge-break-repair.sh --scenario=3 --setup
examples/netedge-tools/netedge-break-repair.sh --scenario=3 --break
examples/netedge-tools/netedge-break-repair.sh --scenario=3 --repair
examples/netedge-tools/netedge-break-repair.sh --scenario=3 --cleanup

# Scenario 4: Route reencrypt without backend TLS
examples/netedge-tools/netedge-break-repair.sh --scenario=4 --setup
examples/netedge-tools/netedge-break-repair.sh --scenario=4 --break
examples/netedge-tools/netedge-break-repair.sh --scenario=4 --status
examples/netedge-tools/netedge-break-repair.sh --scenario=4 --curl
examples/netedge-tools/netedge-break-repair.sh --scenario=4 --repair
```

## Notes

- The script refreshes the Route host from the API after each action so `--curl` always hits the currently admitted hostname.
- Scenario 2 stores the original host in the `netedge-tools-original-host` annotation on the Route; avoid deleting this annotation if you plan to run `--repair`.
- Scenario 3 leaves the namespace intact but removes the managed NetworkPolicy during cleanup.
- Scenario 4 stores a marker annotation on the Route before switching to `reencrypt`. The repair step removes the TLS stanza so the router returns to plain HTTP and handshake errors stop. Use `query_prometheus` (e.g. `haproxy_server_ssl_verify_result_total`) to observe the handshake failures while the break is active.
- If `oc` cannot reach the cluster, commands fail early with diagnostics.
