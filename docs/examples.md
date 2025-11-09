---
layout: page
title: Examples & Tutorials
description: Learn gen-mcp through real-world integration examples
---

# Examples & Tutorials

Welcome to the gen-mcp examples gallery! These comprehensive tutorials demonstrate how to integrate different types of services and APIs as MCP tools. Each example includes step-by-step instructions, configuration breakdowns, and best practices.

Whether you're wrapping a local CLI tool, converting a REST API, or integrating a gRPC service, these examples will guide you through the process.

---

<div class="features-grid" style="margin-top: var(--spacing-lg); margin-bottom: var(--spacing-lg);">
  <div class="feature-card">
    <h3>🤖 Ollama Integration</h3>
    <p>Connect local language models to MCP Clients with gen-mcp in two ways: by wrapping the Ollama CLI, and by wrapping the Ollama HTTP endpoints.</p>
    <p><strong>What you'll learn:</strong></p>
    <ul>
      <li>HTTP-based and CLI-based tool integration</li>
      <li>Input schema validation</li>
      <li>Tool configuration best practices</li>
    </ul>
    <div class="cta-buttons" style="justify-content: flex-start;">
      <a href="{{ '/example-ollama.html' | relative_url }}" class="btn btn-primary">Read Tutorial</a>
      <a href="https://youtu.be/yqJV9rNwfg8" class="btn btn-secondary" target="_blank" rel="noopener">Watch Demo</a>
    </div>
  </div>

  <div class="feature-card">
    <h3>🔗 HTTP API Conversion</h3>
    <p>Transform any REST API into MCP tools automatically. Learn how to use OpenAPI specs to generate complete MCP servers in seconds.</p>
    <p><strong>What you'll learn:</strong></p>
    <ul>
      <li>Automatic OpenAPI → MCP conversion</li>
      <li>Path parameter substitution</li>
      <li>Customizing generated configurations</li>
    </ul>
    <div class="cta-buttons" style="justify-content: flex-start;">
      <a href="{{ '/example-http-conversion.html' | relative_url }}" class="btn btn-primary">Read Tutorial</a>
      <a href="https://youtu.be/boMyFzpgJoA" class="btn btn-secondary" target="_blank" rel="noopener">Watch Demo</a>
    </div>
  </div>

  <div class="feature-card">
    <h3>🔌 gRPC Service Integration</h3>
    <p>Expose existing gRPC services to LLMs without modifying service code using gRPC-Gateway to create an HTTP/JSON proxy.</p>
    <p><strong>Perfect for:</strong></p>
    <ul>
      <li>🏢 Integrating legacy gRPC services with AI workflows</li>
      <li>🔄 Bridging binary gRPC with JSON-based LLM interfaces</li>
      <li>🚀 Getting started before native gRPC support arrives</li>
    </ul>
    <div class="cta-buttons" style="justify-content: flex-start;">
      <a href="https://github.com/genmcp/grpc-demo/" class="btn btn-secondary" target="_blank" rel="noopener">View Demo Repository</a>
    </div>
  </div>
</div>

---

## More Examples Coming Soon

We're continuously adding new examples and tutorials. Have an interesting use case? Share it on our [Discord community](https://discord.gg/AwP6GAUEQR) or contribute it on [GitHub](https://github.com/genmcp/gen-mcp)!
