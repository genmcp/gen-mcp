# NetEdge Break/Repair Script

`netedge-break-repair.sh` stages, breaks, and repairs the deterministic ingress and DNS scenarios captured in `NET_DIAGNOSTIC_SCENARIOS.md`. It deploys a minimal application stack (Deployment, Service, Route), introduces the chosen fault, and restores the healthy baseline when asked.

## Prerequisites

- `oc` CLI available in `$PATH`
- Credentials that allow creating resources in the target namespace
- Ability to pull `registry.redhat.io/openshift4/network-tools-rhel9:latest` (needed if scenario 3 traffic blocks need cleanup)

## Basic Usage

```bash
examples/netedge-tools/netedge-break-repair.sh [--scenario=<1|2|3>] <action>
```

Actions:

- `--setup` – Deploy the healthy baseline (Deployment, Service, Route)
- `--break` – Apply the scenario-specific failure
- `--repair` – Restore the healthy state for the scenario
- `--status` – Show Route, Service, Endpoints (and NetworkPolicy) details
- `--curl` – Curl the current Route host (best effort)
- `--cleanup` – Remove the created resources and any managed NetworkPolicy

If `--scenario` is omitted the script defaults to scenario **1**. Always reuse the same `--scenario=N` flag on follow-up commands; the script prints a reminder after each action.

## Scenarios

1. **Route → Service selector mismatch** – Patches the Service selector to a non-matching value so no endpoints remain (router returns 503). Repair restores the correct selector.
   - Agent prompt: “On the current cluster we exposed an app through a Route but it keeps failing. Diagnose the root cause and tell me how to fix it.”
2. **Route host without DNS record** – Stores the original host, then patches `spec.host` to an NXDOMAIN value. Repair restores the saved host from annotation.
   - Agent prompt: “On the current cluster the Route’s hostname never resolves in DNS even though the Service and Pods look healthy. Diagnose the root cause and tell me how to fix it.”
3. **NetworkPolicy blocking router traffic** – Applies a default-deny ingress NetworkPolicy in the namespace. Repair deletes the policy.
   - Agent prompt: “On the current cluster every request to the Route now times out even though the pods and service look healthy. Diagnose the root cause and tell me how to fix it.”

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
```

## Notes

- The script refreshes the Route host from the API after each action so `--curl` always uses the current value.
- Scenario 2 stores the original host in the `netedge-tools-original-host` annotation on the Route; avoid deleting this annotation if you plan to run `--repair`.
- Scenario 3 leaves the namespace intact but removes the managed NetworkPolicy during cleanup.
- If `oc` cannot reach the cluster, commands fail early with diagnostics.
