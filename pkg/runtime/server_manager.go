package runtime

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	definitions "github.com/genmcp/gen-mcp/pkg/config/definitions"
	"github.com/genmcp/gen-mcp/pkg/mcpserver"
	"github.com/genmcp/gen-mcp/pkg/oauth"
)

type ServerManager struct {
	mcpServer           *mcpserver.MCPServer
	mu                  sync.RWMutex
	scopedServers       map[string]*mcp.Server // set of MCP Servers by oauth scopes
	filteredToolServers map[string]*mcp.Server // as a fallback, the set of MCP Servers that have the same set of filtered tools
}

func NewServerManager(server *mcpserver.MCPServer) *ServerManager {
	logger := server.MCPServerConfig.Runtime.GetBaseLogger()
	logger.Debug("Creating new server manager",
		zap.String("server_name", server.Name()),
		zap.String("server_version", server.Version()))

	return &ServerManager{
		mcpServer:           server,
		scopedServers:       make(map[string]*mcp.Server),
		filteredToolServers: make(map[string]*mcp.Server),
	}
}

// ServerFromContext returns a server based on the auth scopes in the context
// It first checks if there is an existing server for the same set of scopes
// It then checks if after filtering the tools for the received scopes there is an existing server with the same tool set
// Finally, it creates a new server with the correct set of tools and caches the server for future connections
func (sm *ServerManager) ServerFromContext(ctx context.Context) (*mcp.Server, error) {
	logger := sm.mcpServer.MCPServerConfig.Runtime.GetBaseLogger()

	claims := oauth.GetClaimsFromContext(ctx)
	if claims == nil {
		claims = &oauth.TokenClaims{}
	}

	logger.Debug("Looking up server for context",
		zap.String("user_subject", claims.Subject),
		zap.String("scopes", claims.Scope))

	sm.mu.RLock()
	if s, ok := sm.scopedServers[claims.Scope]; ok {
		sm.mu.RUnlock()
		logger.Debug("Server cache hit by scopes",
			zap.String("user_subject", claims.Subject),
			zap.String("scopes", claims.Scope))
		return s, nil
	}

	filteredTools := sm.filterToolsForScope(claims.Scope)
	filteredToolNames := make([]string, len(filteredTools))
	for i, t := range filteredTools {
		filteredToolNames[i] = t.Name
	}

	slices.Sort(filteredToolNames)

	filteredToolNamesKey := strings.Join(filteredToolNames, ",")

	logger.Debug("Filtered tools for user scopes",
		zap.String("user_subject", claims.Subject),
		zap.Int("total_tools", len(sm.mcpServer.MCPToolDefinitions.Tools)),
		zap.Int("filtered_tools", len(filteredTools)),
		zap.Strings("tool_names", filteredToolNames))

	if s, ok := sm.filteredToolServers[filteredToolNamesKey]; ok {
		sm.mu.RUnlock()
		logger.Debug("Server cache hit by filtered tools",
			zap.String("user_subject", claims.Subject),
			zap.String("tool_names_key", filteredToolNamesKey))
		return s, nil
	}

	// no server in either map - need to build the server here
	sm.mu.RUnlock()
	sm.mu.Lock()
	defer sm.mu.Unlock()

	logger.Info("Creating new server instance for user scopes",
		zap.String("user_subject", claims.Subject),
		zap.String("scopes", claims.Scope),
		zap.Int("filtered_tools", len(filteredTools)))

	s, err := makeServerWithTools(sm.mcpServer, filteredTools)
	if err != nil {
		logger.Error("Failed to create server for user scopes",
			zap.String("user_subject", claims.Subject),
			zap.Error(err))
		return nil, err
	}

	sm.scopedServers[claims.Scope] = s
	sm.filteredToolServers[filteredToolNamesKey] = s

	logger.Info("Server created and cached successfully",
		zap.String("user_subject", claims.Subject),
		zap.Int("total_scoped_servers", len(sm.scopedServers)),
		zap.Int("total_filtered_servers", len(sm.filteredToolServers)))

	return s, nil
}

func (sm *ServerManager) filterToolsForScope(scope string) []*definitions.Tool {
	logger := sm.mcpServer.MCPServerConfig.Runtime.GetBaseLogger()
	var allowedTools []*definitions.Tool

	userScopes := strings.Split(scope, " ")
	scopesLookup := make(map[string]struct{}, len(userScopes))
	for _, s := range userScopes {
		scopesLookup[s] = struct{}{}
	}

	logger.Debug("Filtering tools for scope",
		zap.String("scope", scope),
		zap.Int("total_tools", len(sm.mcpServer.MCPToolDefinitions.Tools)))

	for _, tool := range sm.mcpServer.MCPToolDefinitions.Tools {
		if err := checkAuthorization(tool.RequiredScopes, scopesLookup); err != nil {
			logger.Debug("Tool filtered out due to insufficient scopes",
				zap.String("tool_name", tool.Name))
			continue
		}

		logger.Debug("Tool included for user",
			zap.String("tool_name", tool.Name))
		allowedTools = append(allowedTools, tool)
	}

	logger.Debug("Tool filtering completed",
		zap.Int("total_tools", len(sm.mcpServer.MCPToolDefinitions.Tools)),
		zap.Int("allowed_tools", len(allowedTools)))

	return allowedTools
}

func checkAuthorization(requiredScopes []string, userScopes map[string]struct{}) error {
	if len(requiredScopes) == 0 {
		return nil
	}

	for _, requiredScope := range requiredScopes {
		if _, ok := userScopes[requiredScope]; !ok {
			return fmt.Errorf("missing required scope '%s'", requiredScope)
		}
	}

	return nil
}
