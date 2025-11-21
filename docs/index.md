---
layout: default
title: Home
---

<section class="hero">
  <div class="container">
    <h1>gen-mcp</h1>
    <p class="subtitle">Transform any API into an MCP server in seconds, not hours</p>

    <div class="badge-container">
      <img src="https://img.shields.io/github/go-mod/go-version/genmcp/gen-mcp" alt="Go Version">
      <img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License">
      <img src="https://img.shields.io/badge/MCP-Compatible-green.svg" alt="MCP Compatible">
    </div>

    <div class="warning-banner">
      <strong>⚠️ Early Preview:</strong> This is a research project in active development. APIs and features may change.
    </div>

    <div class="cta-buttons">
      <a href="#quick-start" class="btn btn-primary">Get Started</a>
      <a href="https://github.com/genmcp/gen-mcp" class="btn btn-secondary" target="_blank" rel="noopener">View on GitHub</a>
    </div>
  </div>
</section>

<section id="introduction" class="section">
  <div class="container">
    <h2 class="section-title">What is gen-mcp?</h2>
    <p style="text-align: center; max-width: 800px; margin: 0 auto;">
      gen-mcp eliminates the complexity of building Model Context Protocol (MCP) servers. Instead of writing boilerplate code and learning protocol internals, simply describe your tools in a configuration file—gen-mcp handles the rest.
    </p>

    <div class="system-diagram">
      <img src="{{ '/assets/images/gen-mcp-system-diagram.jpg' | relative_url }}" alt="gen-mcp System Diagram">
    </div>

    <h3 style="text-align: center; margin-top: var(--spacing-lg);">Perfect for:</h3>
    <div class="features-grid">
      <div class="feature-card">
        <h3>🔌 API Developers</h3>
        <p>Expose existing REST APIs to AI assistants instantly without writing custom integration code.</p>
      </div>
      <div class="feature-card">
        <h3>🤖 AI Engineers</h3>
        <p>Connect LLMs to external tools without custom server code. Focus on AI, not infrastructure.</p>
      </div>
      <div class="feature-card">
        <h3>🛠️ DevOps Teams</h3>
        <p>Integrate legacy systems with modern AI workflows seamlessly and securely.</p>
      </div>
    </div>
  </div>
</section>

<section id="features" class="section">
  <div class="container">
    <h2 class="section-title">Key Features</h2>
    <div class="features-grid">
      <div class="feature-card">
        <h3>🚀 Zero-Code Server Generation</h3>
        <p>Create MCP servers from simple YAML configs. No programming required.</p>
      </div>
      <div class="feature-card">
        <h3>📡 OpenAPI Auto-Conversion</h3>
        <p>Transform existing OpenAPI specs into MCP servers instantly with a single command.</p>
      </div>
      <div class="feature-card">
        <h3>🔄 Real-Time Tool Exposure</h3>
        <p>HTTP endpoints become callable AI tools automatically. No manual mapping needed.</p>
      </div>
      <div class="feature-card">
        <h3>🛡️ Built-in Validation</h3>
        <p>Schema validation and type safety out of the box. Catch errors before runtime.</p>
      </div>
      <div class="feature-card">
        <h3>🔐 Security Out of the Box</h3>
        <p>TLS encryption and OAuth/OIDC authentication built-in and ready to use.</p>
      </div>
      <div class="feature-card">
        <h3>⚡ Background Processing</h3>
        <p>Detached server mode with process management for production deployments.</p>
      </div>
    </div>
  </div>
</section>

<section id="quick-start" class="section">
  <div class="container">
    <h2 class="section-title">Quick Start</h2>

    <h3>1. Install GenMCP</h3>

    <h4>Option A: Download Pre-built Binary</h4>
    <div class="code-example">
{% highlight bash %}
# Download from GitHub releases
# Visit: https://github.com/genmcp/gen-mcp/releases
# Or using curl (replace with latest version and platform):
curl -L https://github.com/genmcp/gen-mcp/releases/latest/download/genmcp-linux-amd64.zip -o genmcp-linux-amd64.zip
unzip genmcp-linux-amd64.zip
chmod +x genmcp-linux-amd64
sudo mv genmcp-linux-amd64 /usr/local/bin/genmcp
{% endhighlight %}
    </div>

    <h4>Verify the Signed Binary</h4>
    <p>You can cryptographically verify that the downloaded binaries are authentic using cosign:</p>
    <div class="code-example">
{% highlight bash %}
# Install cosign (see https://docs.sigstore.dev/cosign/installation/)
# Download the bundle file from releases page, then verify:
cosign verify-blob-attestation \
  --bundle genmcp-linux-amd64.zip.bundle \
  --certificate-identity-regexp "https://github.com/genmcp/gen-mcp/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --new-bundle-format \
  genmcp-linux-amd64.zip
{% endhighlight %}
    </div>

    <h4>Option B: Build from Source</h4>
    <div class="code-example">
{% highlight bash %}
# Clone and build
git clone https://github.com/genmcp/gen-mcp.git
cd gen-mcp

# Build CLI
make build-cli

# Add to PATH (recommended)
sudo mv genmcp /usr/local/bin
{% endhighlight %}
    </div>

    <h3>2. Choose Your Adventure</h3>
    <p><strong>Option A: Convert Existing API</strong></p>
    <div class="code-example">
{% highlight bash %}
genmcp convert https://api.example.com/openapi.json
genmcp run
{% endhighlight %}
    </div>

    <p><strong>Option B: Create Custom Tools</strong></p>
    <div class="code-example">
{% highlight bash %}
# Create mcpfile.yaml and mcpserver.yaml with your tools (see documentation)
genmcp run
{% endhighlight %}
    </div>

    <h3>3. See It In Action</h3>
    <div class="cta-buttons">
      <a href="https://youtu.be/boMyFzpgJoA" class="btn btn-secondary" target="_blank" rel="noopener">📹 HTTP Conversion Demo</a>
      <a href="https://youtu.be/yqJV9rNwfg8" class="btn btn-secondary" target="_blank" rel="noopener">📹 Ollama Integration Demo</a>
    </div>
    <p style="text-align: center; font-size: 0.9rem; color: var(--text-secondary); margin-top: var(--spacing-sm);">
      <em>Note: These videos were recorded before the project was renamed from automcp to gen-mcp. The functionality remains the same.</em>
    </p>
  </div>
</section>

<section id="documentation" class="section">
  <div class="container">
    <h2 class="section-title">Documentation</h2>
    <div class="features-grid">
      <div class="feature-card">
        <h3>📖 MCP File Format</h3>
        <p>Learn to write custom tool configurations with our comprehensive guide.</p>
        <a href="{{ '/mcpfile.html' | relative_url }}" class="btn btn-primary" style="margin-top: var(--spacing-sm);">Read Guide</a>
        <a href="{{ '/mcpserver.html' | relative_url }}" class="btn btn-primary" style="margin-top: var(--spacing-sm);">Server Config</a>
      </div>
      <div class="feature-card">
        <h3>📚 Examples</h3>
        <p>Real-world integration examples to get you started quickly.</p>
        <a href="{{ '/examples.html' | relative_url }}" class="btn btn-primary" style="margin-top: var(--spacing-sm);">View Examples</a>
      </div>
      <div class="feature-card">
        <h3>🔧 Core Commands</h3>
        <p>Master the essential gen-mcp CLI commands.</p>
        <a href="{{ '/commands.html' | relative_url }}" class="btn btn-primary" style="margin-top: var(--spacing-sm);">Learn Commands</a>
      </div>
    </div>
  </div>
</section>


<section id="contributing" class="section">
  <div class="container">
    <h2 class="section-title">Contributing</h2>
    <p style="text-align: center; max-width: 800px; margin: 0 auto var(--spacing-md);">
      We welcome contributions! This is an early-stage research project with lots of room for improvement.
    </p>

    <div class="features-grid">
      <div class="feature-card">
        <h3>💬 Join Discord</h3>
        <p>Connect with other users, share ideas, and get help from the community.</p>
        <a href="https://discord.gg/AwP6GAUEQR" class="btn btn-primary" style="margin-top: var(--spacing-sm);" target="_blank" rel="noopener">Join Discord</a>
      </div>
      <div class="feature-card">
        <h3>🐛 Report Issues</h3>
        <p>Found a bug or have a feature request? Let us know on GitHub.</p>
        <a href="https://github.com/genmcp/gen-mcp/issues" class="btn btn-primary" style="margin-top: var(--spacing-sm);" target="_blank" rel="noopener">Report Issue</a>
      </div>
      <div class="feature-card">
        <h3>🛠️ Development Setup</h3>
        <p>Get started with local development in just a few commands.</p>
        <a href="https://github.com/genmcp/gen-mcp?tab=readme-ov-file#development-setup" class="btn btn-primary" style="margin-top: var(--spacing-sm);" target="_blank" rel="noopener">View Setup Guide</a>
      </div>
    </div>
  </div>
</section>
