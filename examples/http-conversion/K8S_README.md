# HTTP Endpoint Conversion on Kubernetes (minikube/kind) Example

This example demonstrates gen-mcp's ability to automatically convert HTTP REST API endpoints into MCP tools. gen-mcp can expose any REST API as MCP tools that can be called by AI assistants, eliminating the need to write custom MCP server code.

## Prerequisites

- Helm (v3.10+)
- kubectl
- ko
- Docker or Podman
- minikube or kind
- Basic Kubernetes knowledge

## Getting Started

### 1. Set Up Your Kubernetes Cluster

#### Option A: Using minikube
```bash
minikube start
```

#### Option B: Using kind
```bash
kind create cluster --name genmcp-demo
```

### 2. Install ToolHive Operator

Add the ToolHive Helm repository and install the operator:

```bash
helm repo add toolhive https://stacklok.github.io/toolhive
helm repo update
helm install toolhive-operator toolhive/toolhive-operator
```

Wait for the operator to be ready:

```bash
kubectl wait --for=condition=available --timeout=300s deployment/toolhive-operator-controller-manager
```

### 3. Deploy the Feature Request API Server

First, deploy the feature request server to your cluster:

```bash
cd feature-requests
ko apply -f config/deployment.yaml -f config/service.yaml
```

The API will be available internally in the cluster with endpoints:
- `GET /features` - List all features (summaries only, sorted by upvotes)
- `GET /features/top` - Get the highest-voted feature (summary only)
- `GET /features/{id}` - Get detailed information about a specific feature
- `POST /features` - Add a new feature request
- `POST /features/vote` - Vote for a feature (increases upvotes)
- `POST /features/complete` - Mark a feature request as completed
- `DELETE /features/{id}` - Delete a feature request
- `GET /openapi.json` - Get OpenAPI specification

### 4. Access the API Server

Since we're not using ingress, use kubectl port-forward to access the API:

```bash
kubectl port-forward service/feature-request-demo 9090:80
```

Now the API will be available at `http://localhost:9090`. Keep this port-forward running in a separate terminal.

### 5. Generate Initial MCP Configuration

Use gen-mcp to automatically generate a starter configuration from the API:

```bash
genmcp convert http://localhost:9090/openapi.json -H http://feature-request-demo.default.svc.cluster.local
```

Note: we are using the `-H` flag here to set the base host URL for the API spec, as the openapi.json file says that the endpoints are available at `localhost:9090`.
By setting it to `http://feature-request-demo.default.svc.cluster.local`, the generated configuration will point to the internal Kubernetes service URL that the MCP server can access from within the cluster.

This creates an initial `mcpfile.yaml` based on the OpenAPI specification, with the endpoints all pointing to the internal Kubernetes service.

### 6. Customize the Configuration

Edit the generated `mcpfile.yaml` to:
- Select which endpoints should be exposed as MCP tools
- Improve tool descriptions to help AI models understand when to use each tool
- Add usage instructions or constraints in descriptions
- Configure input validation schemas

Example customizations in this demo:
- Clear, specific descriptions for each tool
- Guidance on when to call related tools (e.g., "Always call get_features-id after this tool...")
- Proper input schemas with required parameters
- Only exposing read endpoints initially (GET operations) for safety

### 7. Deploy the MCP Server with ToolHive

First, create a configmap to contain the mcpfile.yaml:

```bash
kubectl create configmap genmcp-config --from-file=mcpfile.yaml
```

Next, deploy the gen-mcp server using the ToolHive operator:

```bash
ko apply -f toolhive/mcp-server.yaml
```

### 8. Access the MCP Server

Use kubectl port-forward to access the MCP server:

```bash
kubectl port-forward services/mcp-genmcp-proxy 8080:8080
```

The MCP service will now be accessible at `http://localhost:8080/mcp`. To connect to the server, you will need to use the `streamablehttp` protocol and the url `http://localhost:8080/mcp`. You can also explore the MCP server using the [MCP Inspector](https://modelcontextprotocol.io/legacy/tools/inspector), an interactive developer tool for testing and debugging MCP servers.

### 9. Test the MCP Server

You can test the MCP server by connecting with an MCP client or by using tools like curl to verify the endpoints are working:

```bash
# Test the feature requests API
curl http://localhost:9090/features

# Test the MCP server health (if available)
curl http://localhost:8080/health
```

## Key gen-mcp HTTP Conversion Features

- **Automatic Tool Generation**: HTTP endpoints become MCP tools automatically from OpenAPI specs
- **Path Parameter Substitution**: URL templates like `{id}` are handled automatically  
- **Schema Validation**: Input parameters are validated before API calls
- **Streamable HTTP Protocol**: Real-time communication via `streamablehttp`
- **Flexible Configuration**: Full control over which endpoints to expose and how
- **POST/PUT/DELETE Support**: Can expose write operations like adding features, voting, completing, and deleting

## ToolHive Benefits

- **Automatic MCP Server Management**: ToolHive operator handles the lifecycle of MCP servers
- **Resource Configuration**: Easy to configure CPU/memory limits and requests
- **Security by Default**: Servers run in isolated containers with minimal permissions
- **Network Policy Support**: Built-in network security with configurable permission profiles

## Cleanup

To clean up the resources:

```bash
# Delete the MCP server
kubectl delete -f toolhive/mcp-server.yaml

# Delete the feature requests API
kubectl delete -f feature-requests/config/

# Delete the configmap
kubectl delete configmap genmcp-config

# Uninstall ToolHive operator (optional)
helm uninstall toolhive-operator

# Delete the cluster (if using kind)
kind delete cluster --name genmcp-demo

# Stop minikube (if using minikube)
minikube stop
```

## Troubleshooting

- **Port-forward not working**: Make sure the pods are running with `kubectl get pods`
- **API not accessible**: Verify the service is created with `kubectl get services`
- **MCP server not starting**: Check logs with `kubectl logs deployment/genmcp`
- **ToolHive operator issues**: Check operator logs with `kubectl logs -n toolhive-system deployment/toolhive-operator-controller-manager`
