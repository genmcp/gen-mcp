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

## 4. Gateway Service Pending (No LoadBalancer Provider)

### Why it's in-scope & interesting
- Exercises the **Gateway API → Service (LoadBalancer)** integration point.
- Surfaces a common bare-metal/OpenShift Local issue: no cloud LoadBalancer to provision an external IP.
- Highlights a troubleshooting path that combines **resource inspection** with **Prometheus evidence**.
- Reversible by deleting the LoadBalancer Service and falling back to the cluster-internal Route.

### How to stage it
1. Deploy the healthy baseline (Deployment, Service, Route) via `netedge-break-repair.sh --setup`.
2. **Break**: create an additional Service (`${APP_NAME}-gateway`) of type `LoadBalancer` targeting the same pods. Annotate the Service with the intended Gateway host.
3. Optionally create a `HTTPRoute` referencing the Service so tooling can report listener status.
4. Wait ~60s and observe that `.status.loadBalancer.ingress` never populates on clusters without a LoadBalancer implementation.
5. **Repair**: delete the `LoadBalancer` Service (and the optional `HTTPRoute`).

### How a human would diagnose
1. `oc get gateway` (or `HTTPRoute`) → listeners reference a Service that never obtains an external address.
2. `oc get svc ${APP_NAME}-gateway -o yaml` → `spec.type: LoadBalancer` with empty `.status.loadBalancer`.
3. `oc get infrastructure cluster -o jsonpath='{.status.platformStatus.type}'` → platform reports `None`, `BareMetal`, or other environment without managed LBs.
4. Suggest fixes: install MetalLB/OVN-K logical LB, or switch Gateway to use NodePort/Route instead of expecting a cloud LB.

### How the phase-0 agent would diagnose
- `get_service_endpoints` (`service=${APP_NAME}`, `namespace=${NAMESPACE}`) → confirm the backend pods/Endpoints are healthy.
- `inspect_route` → verify the original Route is still functional, isolating the failure to the new Gateway path.
- `query_prometheus` → compare `kube_service_spec_type{service="${APP_NAME}-gateway",namespace="${NAMESPACE}"}` with `kube_service_status_load_balancer_ingress{service="${APP_NAME}-gateway",namespace="${NAMESPACE}"}` to prove no ingress address ever appears.
- `probe_dns_local` or `exec_dns_in_pod` → demonstrate the advertised Gateway host cannot resolve/accept traffic due to the missing IP.
- Recommend mitigation: delete the LoadBalancer Service (fall back to Route) or deploy a LoadBalancer provider.

### Agent input to cause it diagnose this task:
- `On the currently connected cluster we tried to expose the app through a Gateway, but it never provisions an external address. Diagnose the root cause and suggest the fix.`

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
