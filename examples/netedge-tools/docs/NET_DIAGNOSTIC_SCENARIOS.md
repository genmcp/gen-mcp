__# Network Diagnostic Scenarios for Ingress & DNS

This document describes candidate **network breakage scenarios** for use in an AI agent diagnostic project on OpenShift 4.19. The scenarios are designed to be:
- Within the scope of the **Network Ingress and DNS team**
- Detectable using only **Phase 0** diagnostic tools
- Safe — the cluster remains fully accessible remotely

The [`../netedge-break-repair.sh`](../netedge-break-repair.sh) helper script can
stage each scenario end-to-end; see [`README-netedge-break-repair.md`](./README-netedge-break-repair.md)
for usage.

---

## 1. Route → Service Selector Mismatch (Empty Endpoints)

### Why it's in-scope & interesting
- Tests the **Route → Service → Endpoints chain** that the ingress team owns.
- Realistic: label drift or bad selectors happen often and produce 503s.
- 100% reversible by flipping a single label or selector.
- No impact on control-plane or kubeadmin connectivity.

### How to stage it
1. Create a test namespace, deploy a small app (e.g., `nginx`), expose it as a Service and Route.
2. **Break**: modify the Service `spec.selector` to a label that no pod has (or relabel pods).
3. **Repair**: restore the correct selector/pod labels.

### How a human would diagnose
1. `curl` the Route → 503 from the router.
2. `oc get route -o json` → identify `.spec.to.name`.
3. `oc get svc -o json` → inspect selector.
4. `oc get endpoints -o json` → find empty subsets.
5. Compare Service selector vs Pod labels, fix mismatch.

### How the phase-0 agent would diagnose
- `routes_all_namespaces` (resource) → discover candidate Route names/namespaces if the prompt did not supply them.
- `inspect_route` → surface Route, Service, Endpoints, detect empty endpoints.
- `get_service_endpoints` → verify endpoints are empty.
- Optionally `query_prometheus` → check router 503 metrics.

### Agent input to cause it diagnose this task:
- `On the currently connected cluster, we've deployed an app and exposed it through a Route, but it’s not working. Diagnose the root cause and suggest the fix.`
---

## 2. Route Host Without DNS Record (NXDOMAIN)

### Why it's in-scope & interesting
- Tests the DNS ↔ Ingress seam.
- Common misconfiguration: developer sets `spec.host` to `myapp.example.com` without DNS.
- Reversible: fix by using default domain or adding DNS record.

### How to stage it
1. Create a Route with `spec.host` set to `nonexistent.example.test`.
2. **Break**: leave DNS unconfigured.
3. **Repair**: update Route host to valid admitted domain or create DNS record.

### How a human would diagnose
1. `dig` / `nslookup` → NXDOMAIN.
2. `oc get route` → host is admitted, but unreachable.
3. Conclude DNS misconfiguration.

### How the phase-0 agent would diagnose
- `inspect_route` → check host, verify backend chain is healthy.
- `probe_dns_local` → show NXDOMAIN.
- Optionally `exec_dns_in_pod` → in-cluster resolution check.

### Agent input to cause it diagnose this task:
- `On the currently connected cluster, the route's hostname never resolves in DNS even though the service and pods look healthy. Diagnose the root cause and suggest the fix.`

---

## 3. NetworkPolicy Blocking Router → Service Traffic

### Why it's in-scope & interesting
- Tests namespace isolation affecting ingress traffic.
- Real-world: default-deny without allow for router.
- Reversible: apply/remove single NetworkPolicy.

### How to stage it
1. Deploy app in test namespace.
2. **Break**: apply default-deny NetworkPolicy.
3. **Repair**: remove it or add allow for ingress pods.

### How a human would diagnose
1. Route requests hang or 503.
2. Endpoints are healthy.
3. `oc get networkpolicy` → find default-deny.

### Caveat for phase-0 agent
- No built-in `get NetworkPolicy` in phase 0.
- Could infer by symptoms (503 + healthy endpoints) and escalate.

### Agent input to cause it diagnose this task:
- `On the currently connected cluster every request to the Route now times out even though the pods and service look healthy. Diagnose the root cause and suggest the fix.`

---

## 4. Route Reencrypt Without Backend TLS (Router Handshake Failures)

### Why it's in-scope & interesting
- Exercises the **router → backend TLS handshake** path that the ingress team owns.
- Mirrors real incidents where a Route is switched to `reencrypt` but the workload still serves plain HTTP.
- Drives the agent toward **Prometheus metrics** (`haproxy_*`) to see handshake failures accumulate.
- Fully reversible by removing the TLS stanza from the Route.

### How to stage it
1. Deploy the healthy baseline (Deployment, Service, Route) via `netedge-break-repair.sh --setup`.
2. **Break**: patch the Route to `spec.tls.termination: reencrypt` with a bogus CA while leaving the backend Service on HTTP.
3. Wait 30–60s; router pods will begin returning 503s because the backend never completes TLS.
4. **Repair**: remove the Route `spec.tls` block to restore plain HTTP edge traffic.

### How a human would diagnose
1. `curl` the Route → consistent 503 (router) with `x-router` headers still present.
2. `oc get route -o yaml` → `spec.tls.termination: reencrypt` even though the Service targets port 8080 (HTTP).
3. `oc logs -n openshift-ingress router-…` or Prometheus → observe `haproxy_server_ssl_verify_result_total` / `haproxy_server_connection_errors_total` increase.
4. Fix by reverting the Route to edge/HTTP or enabling TLS on the workload.

### How the phase-0 agent would diagnose
- `inspect_route` (`name=${APP_NAME}`, `namespace=${NAMESPACE}`) → surface the Route + Service + Endpoints chain and highlight `spec.tls.termination: reencrypt`.
- `get_service_endpoints` → prove the backend pods are healthy and serving (the issue is not missing endpoints).
- `query_prometheus` → run `haproxy_server_ssl_verify_result_total{route="${APP_NAME}",namespace="${NAMESPACE}"}` or `haproxy_server_connection_errors_total{route="${APP_NAME}",namespace="${NAMESPACE}"}` over the last 5–10 minutes to show TLS verification failures spiking (pass the `.svc` URL or the public Thanos route—the tool auto-discovers the external host and injects the bearer token).
- `probe_dns_local` and/or `exec_dns_in_pod` → confirm the Route hostname still resolves, isolating the fault to router ↔ backend TLS.
- Recommend mitigation: remove `spec.tls` or enable TLS (with a trusted cert) on the backend Service.

### Agent input to cause it diagnose this task:
- `On the currently connected cluster the Route suddenly started returning 503s right after someone enabled reencrypt termination. Diagnose the root cause and tell me how to fix it.`

---

## Recommended Scenario for v1: Selector Mismatch

**Why**:  
- Fully covered by existing Phase 0 tools.  
- Deterministic, scriptable, and reversible.  
- Teaches canonical ingress debugging (Route → Service → Endpoints).

### Human/Agent Diagnostic Flow
1. **Route → Service**  
   `inspect_route` → see Service + Endpoints. Expect empty endpoints.
2. **Endpoints inspection**  
   `get_service_endpoints` → confirm no addresses.
3. **Form hypothesis**  
   Selector mismatch or zero pods.
4. **Repair**  
   Fix label or selector. Endpoints repopulate.
5. **Quantify**  
   `query_prometheus` for router 503s before/after.
