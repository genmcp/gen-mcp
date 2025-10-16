# NETEDGE — gen-mcp Notes

Purpose
-------
A single concise reference for the NETEDGE Phase‑0 gen‑mcp tooling: what’s included,
how to build and run, key assumptions, and short next‑step ideas.

What this directory contains
- `mcpfile.yaml` — curated NETEDGE MCP tools using the `stdio` transport.
- `docs/` — documentation (this file, scenario catalog, break/repair guide).
- `netedge-break-repair.sh` — script that stages documented ingress scenarios.
- `scripts/exec_dns_in_pod.sh` — helper invoked by the `exec_dns_in_pod` tool.

Quick summary of provided tools
- `inspect_route` — fetch a `Route` and, when possible, its `Service` and `Endpoints`.
- `get_service_endpoints` — return an Endpoints object for a Service.
- `query_prometheus` — run a Prometheus `query_range` and return JSON.
- `get_coredns_config` — fetch a ConfigMap (e.g., CoreDNS `Corefile`).
- `probe_dns_local` — run `dig`/`nslookup` on the gen‑mcp host (probe from the host).
- `exec_dns_in_pod` — run a short ephemeral pod that executes `dig` inside the cluster.

DEV NOTES — build & run
-----------------------
- Build (recommended):

  ```bash
  # builds server helper binaries and the CLI per the repo Makefile
  make build
  ```

- Run (foreground):

  ```bash
  ./genmcp run -f examples/netedge-tools/mcpfile.yaml
  ```

- Run (detached/background):

  ```bash
  ./genmcp run -f examples/netedge-tools/mcpfile.yaml -d
  ./genmcp stop -f examples/netedge-tools/mcpfile.yaml
  ```

Integration with Codex CLI
--------------------------
The NETEDGE tools use the `stdio` transport, so Codex CLI can launch the server
directly. Example `config.toml` snippet:

```toml
[mcp_servers.netedge]
command = "/absolute/path/to/genmcp"
args    = ["run", "-f", "examples/netedge-tools/mcpfile.yaml"]
```

Codex spawns the command in STDIO mode; no HTTP proxy is required. Adjust the path
to `genmcp` for your local checkout.

Key assumptions and caveats
--------------------------
- The Phase‑0 tools use the `cli` invoker (they shell out). The following tools must
  be available on the machine running the MCP server: `oc` or `kubectl`, `curl`, and
  DNS tools (`dig` or `nslookup`). `jq` or `python3` is helpful for JSON extraction.
- `exec_dns_in_pod` pulls `registry.redhat.io/openshift4/network-tools-rhel9:latest`;
  replace with an approved image if your cluster restricts external pulls.
- Template notes: when writing CLI `command` templates, each `{param}` must appear
  exactly once. If a parameter must be used multiple times, assign it once to a shell
  variable inside the command and reuse that variable. The repo validator counts the
  `"%s"` placeholders used during formatting.

We could...
-----------
- add a single aggregator HTTP endpoint (e.g. `/diagnose`) that implements the full
  `diagnose_route` playbook and returns structured JSON so agents call one concise tool.
- implement native `k8s` and `prometheus` invokers in `pkg/invocation` so tools use
  `client-go` and HTTP clients (robust in‑cluster auth) instead of shelling out.
- include a tiny `mcp-remote` adapter in the repo so Codex users can reproduce the
  Codex integration locally without a separate tool.
- add safe remediation tools behind approval gates (preview/dry‑run + action IDs +
  rollback tokens) and integrate with audit logs for traceability.

Phased roadmap
--------------
Phase 0 — Quick wins
- Provide read-only aggregation tools and lightweight probes so agents can collect
  immediate evidence without custom code. Example tasks:
  - `inspect_route`, `get_service_endpoints`, `get_coredns_config`, `query_prometheus`.
  - `probe_dns_local` and `exec_dns_in_pod` using ephemeral pod runs.
  - Deliver a curated `mcpfile.yaml` (already present) and clear DEV notes for running
    the server locally or via the Makefile.

Phase 1 — Probing and aggregation
- Add active probing and standardized aggregation:
  - Implement `probe_http`, `probe_dns` (multiple transports), `probe_endpoints`.
  - Build an aggregator HTTP endpoint (e.g. `/diagnose`) that executes the
    `diagnose_route` playbook, runs parallel probes, and returns structured JSON
    (checks, probes, root causes, recommended actions).
  - Improve ephemeral pod lifecycle and add better result correlation (logs, metrics).

Phase 2 — Safe remediation (human-in-the-loop)
- Expose guarded remediation actions with audit and rollback support:
  - `preview_apply_corefile`, `apply_corefile`, `patch_route`, `scale_backend` with
    dry-run previews and an `action-id` for traceability.
  - Integrate RBAC and scope-based filtering so tools are gated by required scopes.
  - Add approval workflows and automatic rollback tokens for safe operator-driven fixes.

These phases are incremental: each phase builds on the previous to increase
automation while keeping safety and auditability central.

Location
--------
This file is the canonical NETEDGE notes for the gen‑mcp examples: `examples/netedge-tools/docs/NETEDGE-GEN-MCP-NOTES.md`.
