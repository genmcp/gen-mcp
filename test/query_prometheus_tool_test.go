package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	mcpli "github.com/genmcp/gen-mcp/pkg/invocation/cli"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
)

func TestQueryPrometheusFallbacksToRouteForSvcURL(t *testing.T) {
	t.Parallel()

	mcpFilePath := filepath.Join("..", "examples", "netedge-tools", "mcpfile.yaml")
	mcpCfg, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		t.Fatalf("failed to parse MCP file: %v", err)
	}

	var tool *mcpfile.Tool
	for _, ttool := range mcpCfg.Tools {
		if ttool.Name == "query_prometheus" {
			tool = ttool
			break
		}
	}
	if tool == nil {
		t.Fatalf("query_prometheus tool not found in MCP file")
	}

	rawConfig, err := invocation.ParseInvocation(tool.GetInvocationType(), tool.GetInvocationData(), tool)
	if err != nil {
		t.Fatalf("failed to parse CLI config: %v", err)
	}

	config, ok := rawConfig.(*mcpli.CliInvocationConfig)
	if !ok {
		t.Fatalf("unexpected config type %T", rawConfig)
	}

	tmpDir := t.TempDir()
	fakeRouteHost := "thanos-querier-openshift-monitoring.apps.test.example.com"
	fakeToken := "FAKE_TOKEN"

	ocScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
if [[ "$#" -ge 7 && "$1" == "-n" && "$2" == "openshift-monitoring" && "$3" == "get" && "$4" == "route" && "$5" == "thanos-querier" ]]; then
  echo "%s"
  exit 0
fi
if [[ "$#" -eq 2 && "$1" == "whoami" && "$2" == "-t" ]]; then
  echo "%s"
  exit 0
fi
echo "unexpected oc invocation: $*" >&2
exit 1
`, fakeRouteHost, fakeToken)

	if err := os.WriteFile(filepath.Join(tmpDir, "oc"), []byte(ocScript), 0o755); err != nil {
		t.Fatalf("failed to write fake oc script: %v", err)
	}

	curlLog := filepath.Join(tmpDir, "curl_args.log")
	curlResponse := `{"status":"success","data":{"result":[]}}`
	curlScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
{
  for arg in "$@"; do
    printf '%%s\n' "$arg"
  done
} > "%s"
printf '%s'
`, curlLog, curlResponse)

	if err := os.WriteFile(filepath.Join(tmpDir, "curl"), []byte(curlScript), 0o755); err != nil {
		t.Fatalf("failed to write fake curl script: %v", err)
	}

	args := make([]any, len(config.ParameterIndices))
	setArg := func(name, value string) {
		idx, ok := config.ParameterIndices[name]
		if !ok {
			t.Fatalf("missing parameter index for %s", name)
		}
		args[idx] = value
	}

	setArg("prometheus_url", "https://thanos-querier.openshift-monitoring.svc:9091")
	setArg("query", `sum by (namespace,route,host) (rate(haproxy_server_ssl_verify_result_total{namespace="test-ingress",route="hello"}[5m]))`)
	setArg("start", "2025-10-23T14:22:53Z")
	setArg("end", "2025-10-23T14:32:53Z")
	setArg("step", "60s")

	command := fmt.Sprintf(config.Command, args...)
	if testing.Verbose() {
		t.Logf("rendered command:\n%s", command)
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PATH=%s%c%s", tmpDir, os.PathListSeparator, os.Getenv("PATH")),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command execution failed: %v\noutput: %s", err, string(output))
	}

	if strings.TrimSpace(string(output)) != curlResponse {
		t.Fatalf("unexpected command output: %s", string(output))
	}

	data, err := os.ReadFile(curlLog)
	if err != nil {
		t.Fatalf("failed to read curl log: %v", err)
	}
	curlArgs := string(data)
	if !strings.Contains(curlArgs, "-k") {
		t.Fatalf("expected curl to be invoked with -k, got: %s", curlArgs)
	}
	if !strings.Contains(curlArgs, "Authorization: Bearer "+fakeToken) {
		t.Fatalf("expected Authorization header with bearer token, got: %s", curlArgs)
	}
	expectedURL := "https://" + fakeRouteHost + "/api/v1/query_range"
	if !strings.Contains(curlArgs, expectedURL) {
		t.Fatalf("expected curl to target %s, got: %s", expectedURL, curlArgs)
	}
	if !strings.Contains(curlArgs, "start=2025-10-23T14:22:53Z") {
		t.Fatalf("expected start parameter in curl args, got: %s", curlArgs)
	}
	if !strings.Contains(curlArgs, "end=2025-10-23T14:32:53Z") {
		t.Fatalf("expected end parameter in curl args, got: %s", curlArgs)
	}
	if !strings.Contains(curlArgs, `query=sum by (namespace,route,host) (rate(haproxy_server_ssl_verify_result_total{namespace=test-ingress,route=hello}[5m]))`) {
		t.Fatalf("expected query parameter in curl args, got: %s", curlArgs)
	}
}
