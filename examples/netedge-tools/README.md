NETEDGE gen-mcp examples
========================

This directory contains the NETEDGE gen-mcp example toolset. Documentation now
lives under `docs/`; start with the canonical notes for build/run details and
integration guidance.

Docs
- [`docs/NETEDGE-GEN-MCP-NOTES.md`](docs/NETEDGE-GEN-MCP-NOTES.md) — canonical notes covering setup, runtime tips, and roadmap.
- [`docs/NET_DIAGNOSTIC_SCENARIOS.md`](docs/NET_DIAGNOSTIC_SCENARIOS.md) — ingress and DNS failure scenarios for agents.
- [`docs/README-netedge-break-repair.md`](docs/README-netedge-break-repair.md) — break/repair script usage for staging scenarios.

Key files
- `mcpfile.yaml` — curated MCP tool definitions used by these examples.
- `netedge-break-repair.sh` — helper script that stages the documented scenarios.
- `scripts/exec_dns_in_pod.sh` — helper invoked by the `exec_dns_in_pod` MCP tool.

Update the canonical notes if you need to expand or refine the documentation.
