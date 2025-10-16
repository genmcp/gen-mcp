package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/genmcp/gen-mcp/pkg/invocation"
	_ "github.com/genmcp/gen-mcp/pkg/invocation/cli"
	_ "github.com/genmcp/gen-mcp/pkg/invocation/http"
	"github.com/genmcp/gen-mcp/pkg/invocation/utils"
	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/oauth"
	"github.com/genmcp/gen-mcp/pkg/observability/logging"
)

func MakeServer(mcpServer *mcpfile.MCPServer) (*mcp.Server, error) {
	logger := mcpServer.Runtime.GetBaseLogger()
	logger.Debug("Creating MCP server",
		zap.String("server_name", mcpServer.Name),
		zap.String("server_version", mcpServer.Version))

	// apply the runtime overrides to the mcp server
	// if something goes wrong in the env vars, we warn but continue
	envOverrider := mcpfile.NewEnvRuntimeOverrider()
	if err := envOverrider.ApplyOverrides(mcpServer.Runtime); err != nil {
		logger.Warn("Failed to apply overrides from env vars to the mcp server",
			zap.String("server_name", mcpServer.Name),
			zap.Error(err))
	}

	// Validate the server configuration before creating the server
	if err := mcpServer.Validate(invocation.InvocationValidator); err != nil {
		logger.Error("Server configuration validation failed",
			zap.String("server_name", mcpServer.Name),
			zap.Error(err))
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	logger.Info("Server configuration validated successfully",
		zap.String("server_name", mcpServer.Name))

	server, err := makeServerWithoutValidation(mcpServer)
	if err != nil {
		logger.Error("Failed to create server",
			zap.String("server_name", mcpServer.Name),
			zap.Error(err))
		return nil, err
	}

	logger.Info("MCP server created successfully",
		zap.String("server_name", mcpServer.Name),
		zap.String("server_version", mcpServer.Version))
	return server, nil
}

// makeServerWithoutValidation creates a server without performing validation
// This is used internally when validation has already been performed
func makeServerWithoutValidation(mcpServer *mcpfile.MCPServer) (*mcp.Server, error) {
	return makeServerWithTools(mcpServer, mcpServer.Tools)
}

func RunServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	logger := mcpServerConfig.Runtime.GetBaseLogger()
	logger.Info("Starting MCP server",
		zap.String("server_name", mcpServerConfig.Name),
		zap.String("server_version", mcpServerConfig.Version),
		zap.String("transport_protocol", mcpServerConfig.Runtime.TransportProtocol))

	// Validate the server configuration before running
	if err := mcpServerConfig.Validate(invocation.InvocationValidator); err != nil {
		logger.Error("Server configuration validation failed before running",
			zap.String("server_name", mcpServerConfig.Name),
			zap.Error(err))
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	logger.Debug("Server configuration validated, selecting transport protocol",
		zap.String("transport_protocol", mcpServerConfig.Runtime.TransportProtocol))

	switch strings.ToLower(mcpServerConfig.Runtime.TransportProtocol) {
	case mcpfile.TransportProtocolStreamableHttp:
		logger.Info("Running server with streamable HTTP transport")
		return runStreamableHttpServer(ctx, mcpServerConfig)
	case mcpfile.TransportProtocolStdio:
		logger.Info("Running server with stdio transport")
		return runStdioServer(ctx, mcpServerConfig)
	default:
		logger.Error("Invalid transport protocol specified",
			zap.String("transport_protocol", mcpServerConfig.Runtime.TransportProtocol))
		return fmt.Errorf("tried running invalid transport protocol")
	}
}

// RunServers runs all servers defined in the MCP file
func RunServers(ctx context.Context, mcpFilePath string) error {
	mcpConfig, err := mcpfile.ParseMCPFile(mcpFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse mcp file: %w", err)
	}

	// Now we can get the logger from the runtime config
	logger := mcpConfig.Runtime.GetBaseLogger()
	logger.Info("Starting servers from MCP file",
		zap.String("mcp_file_path", mcpFilePath),
		zap.String("server_name", mcpConfig.Name),
		zap.String("server_version", mcpConfig.Version))

	if err := mcpConfig.Validate(invocation.InvocationValidator); err != nil {
		logger.Error("MCP file validation failed",
			zap.String("mcp_file_path", mcpFilePath),
			zap.Error(err))
		return fmt.Errorf("mcp file is invalid: %w", err)
	}

	logger.Debug("MCP file validated successfully, creating server instance")

	mcpServer := &mcpfile.MCPServer{
		Name:              mcpConfig.Name,
		Version:           mcpConfig.Version,
		Runtime:           mcpConfig.Runtime,
		Tools:             mcpConfig.Tools,
		Prompts:           mcpConfig.Prompts,
		Resources:         mcpConfig.Resources,
		ResourceTemplates: mcpConfig.ResourceTemplates,
	}

	// Apply runtime overrides from environment variables
	// if something goes wrong in the env vars, we warn but continue
	envOverrider := mcpfile.NewEnvRuntimeOverrider()
	if err := envOverrider.ApplyOverrides(mcpServer.Runtime); err != nil {
		logger.Warn("Failed to apply overrides from env vars to the mcp server",
			zap.String("server_name", mcpServer.Name),
			zap.Error(err))
	}

	return RunServer(ctx, mcpServer)
}

func runStreamableHttpServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	logger := mcpServerConfig.Runtime.GetBaseLogger()
	port := mcpServerConfig.Runtime.StreamableHTTPConfig.Port
	basePath := mcpServerConfig.Runtime.StreamableHTTPConfig.BasePath
	stateless := mcpServerConfig.Runtime.StreamableHTTPConfig.Stateless

	logger.Info("Setting up streamable HTTP server",
		zap.Int("port", port),
		zap.String("base_path", basePath),
		zap.Bool("stateless", stateless))

	sm := NewServerManager(mcpServerConfig)
	// Create a root mux to handle different endpoints
	mux := http.NewServeMux()

	logger.Debug("Creating MCP handler")
	// Set up MCP server under /mcp (or whatever is under BasePath)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		s, err := sm.ServerFromContext(r.Context())
		if err != nil {
			logger.Warn("Failed to get server from context in handler",
				zap.Error(err),
				zap.String("request_uri", r.RequestURI))
			return nil
		}

		return s
	}, &mcp.StreamableHTTPOptions{
		Stateless: stateless,
	})

	logger.Debug("Setting up OAuth middleware")
	oauthHandler := oauth.Middleware(mcpServerConfig)(handler)

	mux.Handle(basePath, oauthHandler)
	logger.Debug("Registered MCP handler", zap.String("path", basePath))

	// Set up OAuth protected resource metadata endpoint under / if needed
	if mcpServerConfig.Runtime.StreamableHTTPConfig.Auth != nil {
		logger.Debug("Setting up OAuth protected resource metadata endpoint")
		mux.HandleFunc(oauth.ProtectedResourceMetadataEndpoint, oauth.ProtectedResourceMetadataHandler(mcpServerConfig))
		logger.Debug("Registered OAuth metadata handler", zap.String("path", oauth.ProtectedResourceMetadataEndpoint))
	}

	// Use custom server with the mux
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	logger.Info("Starting HTTP server", zap.Int("port", port))

	// Channel to capture server errors
	errCh := make(chan error, 1)
	go func() {
		var err error
		if mcpServerConfig.Runtime.StreamableHTTPConfig.TLS != nil {
			logger.Info("Starting HTTPS server with TLS",
				zap.String("cert_file", mcpServerConfig.Runtime.StreamableHTTPConfig.TLS.CertFile),
				zap.String("key_file", mcpServerConfig.Runtime.StreamableHTTPConfig.TLS.KeyFile))
			err = srv.ListenAndServeTLS(
				mcpServerConfig.Runtime.StreamableHTTPConfig.TLS.CertFile,
				mcpServerConfig.Runtime.StreamableHTTPConfig.TLS.KeyFile,
			)
		} else {
			logger.Info("Starting HTTP server")
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
			errCh <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, shutting down HTTP server gracefully")
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("Error during server shutdown", zap.Error(err))
			return err
		}
		logger.Info("HTTP server shutdown completed")
		return nil
	case err := <-errCh:
		logger.Error("HTTP server failed", zap.Error(err))
		return err
	}
}

func runStdioServer(ctx context.Context, mcpServerConfig *mcpfile.MCPServer) error {
	logger := mcpServerConfig.Runtime.GetBaseLogger()
	logger.Info("Setting up stdio server",
		zap.String("server_name", mcpServerConfig.Name),
		zap.String("server_version", mcpServerConfig.Version))

	s, err := makeServerWithoutValidation(mcpServerConfig)
	if err != nil {
		logger.Error("Failed to create stdio server", zap.Error(err))
		return fmt.Errorf("failed to create server: %w", err)
	}

	logger.Info("Starting stdio server")
	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.Error("Stdio server failed", zap.Error(err))
		return err
	}

	logger.Info("Stdio server completed")
	return nil
}

// checkPrimitiveAuthorization verifies if user has required scopes for a primitive (tool or prompt)
func checkPrimitiveAuthorization(ctx context.Context, requiredScopes []string, primitiveName, primitiveType string) error {
	if len(requiredScopes) == 0 {
		return nil // No scopes required
	}

	baseLogger := logging.BaseFromContext(ctx)
	userClaims := oauth.GetClaimsFromContext(ctx)
	if userClaims == nil {
		// Server-side security logging - NOT sent to client
		baseLogger.Warn("Authorization check failed: no authentication context found",
			zap.String("primitive_name", primitiveName),
			zap.String("primitive_type", primitiveType))
		return fmt.Errorf("no authentication context found")
	}

	// Split the scope string into individual scopes
	userScopes := strings.Split(userClaims.Scope, " ")

	// Check if user has all required scopes
	for _, requiredScope := range requiredScopes {
		if !slices.Contains(userScopes, requiredScope) {
			// Server-side security logging - NOT sent to client
			baseLogger.Warn("Authorization check failed: missing required scope",
				zap.String("primitive_name", primitiveName),
				zap.String("primitive_type", primitiveType),
				zap.String("user_subject", userClaims.Subject))
			return fmt.Errorf("missing required scope")
		}
	}

	// Log successful authorization at debug level (server-side only)
	baseLogger.Debug("Authorization check passed",
		zap.String("primitive_name", primitiveName),
		zap.String("primitive_type", primitiveType),
		zap.String("user_subject", userClaims.Subject))

	return nil
}

// createAuthorizedToolHandler wraps a tool handler with authorization checks
func createAuthorizedToolHandler(tool *mcpfile.Tool) (mcp.ToolHandler, error) {
	invoker, err := invocation.CreateInvoker(tool)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoker for tool %s: %w", tool.Name, err)
	}

	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientLogger := logging.FromContext(ctx) // Sent to MCP client

		// Check if user has required scopes for this tool
		if err := checkPrimitiveAuthorization(ctx, tool.RequiredScopes, tool.Name, "tool"); err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Tool authorization failed",
				zap.String("tool_name", tool.Name),
				zap.Error(err))
			// Return generic error to client - don't reveal tool name or specific authorization failure
			return utils.McpTextError("forbidden: insufficient permissions"), nil
		}

		// Client can see their own successful tool invocations
		clientLogger.Info("Tool invocation started", zap.String("tool_name", tool.Name))

		result, err := invoker.Invoke(ctx, req)
		if err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Tool invocation failed",
				zap.String("tool_name", tool.Name),
				zap.Error(err))
			// Log generic error for client
			clientLogger.Error("Tool invocation failed",
				zap.String("tool_name", tool.Name),
				zap.String("error", "invocation error"))
			// Return result (may contain partial output) but with generic error to prevent info leakage
			if result != nil {
				return result, nil
			}
			return utils.McpTextError("tool invocation failed"), nil
		}

		clientLogger.Info("Tool invocation completed successfully", zap.String("tool_name", tool.Name))
		return result, nil
	}, nil
}

func createAuthorizedPromptHandler(prompt *mcpfile.Prompt) (mcp.PromptHandler, error) {
	invoker, err := invocation.CreatePromptInvoker(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoker for prompt %s: %w", prompt.Name, err)
	}

	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		clientLogger := logging.FromContext(ctx) // Sent to MCP client

		// Check if user has required scopes for this prompt
		if err := checkPrimitiveAuthorization(ctx, prompt.RequiredScopes, prompt.Name, "prompt"); err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Prompt authorization failed",
				zap.String("prompt_name", prompt.Name),
				zap.Error(err))
			// Return generic error to client - don't reveal prompt name or specific authorization failure
			return utils.McpPromptTextError("forbidden: insufficient permissions"), nil
		}

		// Client can see their own successful prompt invocations
		clientLogger.Info("Prompt invocation started", zap.String("prompt_name", prompt.Name))

		result, err := invoker.InvokePrompt(ctx, req)
		if err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Prompt invocation failed",
				zap.String("prompt_name", prompt.Name),
				zap.Error(err))
			// Log generic error for client
			clientLogger.Error("Prompt invocation failed",
				zap.String("prompt_name", prompt.Name),
				zap.String("error", "invocation error"))
			// Return result (may contain partial output) but with generic error to prevent info leakage
			if result != nil {
				return result, nil
			}
			return utils.McpPromptTextError("prompt invocation failed"), nil
		}

		clientLogger.Info("Prompt invocation completed successfully", zap.String("prompt_name", prompt.Name))
		return result, nil
	}, nil
}

func createAuthorizedResourceHandler(resource *mcpfile.Resource) (mcp.ResourceHandler, error) {
	invoker, err := invocation.CreateResourceInvoker(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoker for resource %s: %w", resource.Name, err)
	}

	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		clientLogger := logging.FromContext(ctx) // Sent to MCP client

		// Check if user has required scopes for this resource
		if err := checkPrimitiveAuthorization(ctx, resource.RequiredScopes, resource.Name, "resource"); err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Resource authorization failed",
				zap.String("resource_name", resource.Name),
				zap.Error(err))
			// Return generic error to client - don't reveal resource name or specific authorization failure
			return utils.McpResourceTextError("forbidden: insufficient permissions"), nil
		}

		// Client can see their own successful resource access
		clientLogger.Info("Resource access started", zap.String("resource_name", resource.Name))

		result, err := invoker.InvokeResource(ctx, req)
		if err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Resource access failed",
				zap.String("resource_name", resource.Name),
				zap.Error(err))
			// Log generic error for client
			clientLogger.Error("Resource access failed",
				zap.String("resource_name", resource.Name),
				zap.String("error", "invocation error"))
			// Return result (may contain partial output) but with generic error to prevent info leakage
			if result != nil {
				return result, nil
			}
			return utils.McpResourceTextError("resource access failed"), nil
		}

		clientLogger.Info("Resource access completed successfully", zap.String("resource_name", resource.Name))
		return result, nil
	}, nil
}

func createAuthorizedResourceTemplateHandler(resourceTemplate *mcpfile.ResourceTemplate) (mcp.ResourceHandler, error) {
	invoker, err := invocation.CreateResourceTemplateInvoker(resourceTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoker for resource template %s: %w", resourceTemplate.Name, err)
	}
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		clientLogger := logging.FromContext(ctx) // Sent to MCP client

		// Check if user has required scopes for this resource template
		if err := checkPrimitiveAuthorization(ctx, resourceTemplate.RequiredScopes, resourceTemplate.Name, "resource_template"); err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Resource template authorization failed",
				zap.String("resource_template_name", resourceTemplate.Name),
				zap.Error(err))
			// Return generic error to client - don't reveal resource template name or specific authorization failure
			return utils.McpResourceTextError("forbidden: insufficient permissions"), nil
		}

		// Client can see their own successful resource template access
		clientLogger.Info("Resource template access started", zap.String("resource_template_name", resourceTemplate.Name))

		result, err := invoker.InvokeResourceTemplate(ctx, req)
		if err != nil {
			// Log detailed error server-side only
			baseLogger := logging.BaseFromContext(ctx)
			baseLogger.Error("Resource template access failed",
				zap.String("resource_template_name", resourceTemplate.Name),
				zap.Error(err))
			// Log generic error for client
			clientLogger.Error("Resource template access failed",
				zap.String("resource_template_name", resourceTemplate.Name),
				zap.String("error", "invocation error"))
			// Return result (may contain partial output) but with generic error to prevent info leakage
			if result != nil {
				return result, nil
			}
			return utils.McpResourceTextError("resource template access failed"), nil
		}

		clientLogger.Info("Resource template access completed successfully", zap.String("resource_template_name", resourceTemplate.Name))
		return result, nil
	}, nil
}

// makeServerWithTools makes a server using the server metadata in mcpServer but with the tools specified in tools
// this is useful for creating servers with filtered tool lists
func makeServerWithTools(mcpServer *mcpfile.MCPServer, tools []*mcpfile.Tool) (*mcp.Server, error) {
	logger := mcpServer.Runtime.GetBaseLogger()
	logger.Debug("Building MCP server with tools",
		zap.String("server_name", mcpServer.Name),
		zap.String("server_version", mcpServer.Version),
		zap.Int("num_tools", len(tools)),
		zap.Int("num_prompts", len(mcpServer.Prompts)),
		zap.Int("num_resources", len(mcpServer.Resources)),
		zap.Int("num_resource_templates", len(mcpServer.ResourceTemplates)))

	opts := &mcp.ServerOptions{
		HasTools:     len(mcpServer.Tools) > 0,
		HasPrompts:   len(mcpServer.Prompts) > 0,
		HasResources: len(mcpServer.Resources)+len(mcpServer.ResourceTemplates) > 0,
	}
	if mcpServer.Instructions != "" {
		logger.Debug("Adding server instructions")
		opts.Instructions = mcpServer.Instructions
	}

	s := mcp.NewServer(&mcp.Implementation{
		Name:    mcpServer.Name,
		Version: mcpServer.Version,
	}, opts)

	logger.Debug("Adding logging middleware")
	s.AddReceivingMiddleware(logging.WithLoggingMiddleware(logger))

	var serverErr error
	logger.Debug("Registering tools", zap.Int("count", len(tools)))
	for _, t := range tools {
		handler, err := createAuthorizedToolHandler(t)
		if err != nil {
			logger.Error("Failed to create tool handler",
				zap.String("tool_name", t.Name),
				zap.Error(err))
			serverErr = errors.Join(serverErr, err)
			continue
		}

		tool := &mcp.Tool{
			Name:        t.Name,
			Description: t.Description,
			Title:       t.Title,
			InputSchema: t.InputSchema,
			Annotations: &mcp.ToolAnnotations{
				Title: t.Title, // some clients use the annotation instead of the title field from the tool
			},
		}

		// Only set OutputSchema if it's not nil to avoid typed nil issues
		if t.OutputSchema != nil {
			tool.OutputSchema = t.OutputSchema
		}

		// only override annotation defaults if they are set by the user
		if t.Annotations != nil {
			if t.Annotations.DestructiveHint != nil {
				tool.Annotations.DestructiveHint = t.Annotations.DestructiveHint
			}
			if t.Annotations.IdempotentHint != nil {
				tool.Annotations.IdempotentHint = *t.Annotations.IdempotentHint
			}
			if t.Annotations.OpenWorldHint != nil {
				tool.Annotations.OpenWorldHint = t.Annotations.OpenWorldHint
			}
			if t.Annotations.ReadOnlyHint != nil {
				tool.Annotations.ReadOnlyHint = *t.Annotations.ReadOnlyHint
			}
		}

		s.AddTool(tool, handler)
		logger.Debug("Registered tool", zap.String("tool_name", t.Name))
	}

	logger.Debug("Registering prompts", zap.Int("count", len(mcpServer.Prompts)))
	for _, p := range mcpServer.Prompts {
		handler, err := createAuthorizedPromptHandler(p)
		if err != nil {
			logger.Error("Failed to create prompt handler",
				zap.String("prompt_name", p.Name),
				zap.Error(err))
			serverErr = errors.Join(serverErr, err)
			continue
		}

		s.AddPrompt(
			&mcp.Prompt{
				Name:        p.Name,
				Description: p.Description,
			},
			handler,
		)
		logger.Debug("Registered prompt", zap.String("prompt_name", p.Name))
	}

	logger.Debug("Registering resources", zap.Int("count", len(mcpServer.Resources)))
	for _, r := range mcpServer.Resources {
		handler, err := createAuthorizedResourceHandler(r)
		if err != nil {
			logger.Error("Failed to create resource handler",
				zap.String("resource_name", r.Name),
				zap.Error(err))
			serverErr = errors.Join(serverErr, err)
			continue
		}

		s.AddResource(
			&mcp.Resource{
				Name:        r.Name,
				Description: r.Description,
				URI:         r.URI,
				MIMEType:    r.MIMEType,
				Size:        r.Size,
			},
			handler,
		)
		logger.Debug("Registered resource", zap.String("resource_name", r.Name))
	}

	logger.Debug("Registering resource templates", zap.Int("count", len(mcpServer.ResourceTemplates)))
	for _, rt := range mcpServer.ResourceTemplates {
		handler, err := createAuthorizedResourceTemplateHandler(rt)
		if err != nil {
			logger.Error("Failed to create resource template handler",
				zap.String("resource_template_name", rt.Name),
				zap.Error(err))
			serverErr = errors.Join(serverErr, err)
			continue
		}

		s.AddResourceTemplate(
			&mcp.ResourceTemplate{
				Name:        rt.Name,
				Description: rt.Description,
				URITemplate: rt.URITemplate,
				MIMEType:    rt.MIMEType,
			},
			handler,
		)
		logger.Debug("Registered resource template", zap.String("resource_template_name", rt.Name))
	}

	if serverErr != nil {
		logger.Warn("Server created with some errors", zap.Error(serverErr))
	} else {
		logger.Info("Server created successfully with all components")
	}

	return s, serverErr
}
