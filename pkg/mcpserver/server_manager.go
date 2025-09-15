package mcpserver

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/genmcp/gen-mcp/pkg/mcpfile"
	"github.com/genmcp/gen-mcp/pkg/oauth"
)

type ServerManager struct {
	mcpServer           *mcpfile.MCPServer
	mu                  sync.RWMutex
	scopedServers       map[string]*mcp.Server // set of MCP Servers by oauth scopes
	filteredToolServers map[string]*mcp.Server // as a fallback, the set of MCP Servers that have the same set of filtered tools
}

func NewServerManager(server *mcpfile.MCPServer) *ServerManager {
	return &ServerManager{
		mcpServer:           server,
		scopedServers:       make(map[string]*mcp.Server),
		filteredToolServers: make(map[string]*mcp.Server),
	}
}

// ServerFromContext returns a server based on the auth scopes in the context
// It first checks if there is an existing server for the same set of scopes
// It then checks if after filtering the tools for the received scopes there is an existing server with the same tool set
// Finally, it creates a new server with the correct set of tools and chaches the server for future connections
func (sm *ServerManager) ServerFromContext(ctx context.Context) (*mcp.Server, error) {
	claims := oauth.GetClaimsFromContext(ctx)
	if claims == nil {
		claims = &oauth.TokenClaims{}
	}

	sm.mu.RLock()
	if s, ok := sm.scopedServers[claims.Scope]; ok {
		sm.mu.RUnlock()
		return s, nil
	}

	filteredTools := sm.filterToolsForScope(claims.Scope)
	filteredToolNames := make([]string, len(filteredTools))
	for i, t := range filteredTools {
		filteredToolNames[i] = t.Name
	}

	slices.Sort(filteredToolNames)

	filteredToolNamesKey := strings.Join(filteredToolNames, ",")

	if s, ok := sm.filteredToolServers[filteredToolNamesKey]; ok {
		sm.mu.RUnlock()
		return s, nil
	}

	// no server in either map - need to build the server here
	sm.mu.RUnlock()
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, err := makeServerWithTools(sm.mcpServer, filteredTools)
	if err != nil {
		return nil, err
	}

	sm.scopedServers[claims.Scope] = s
	sm.filteredToolServers[filteredToolNamesKey] = s

	return s, nil
}

func (sm *ServerManager) filterToolsForScope(scope string) []*mcpfile.Tool {
	var allowedTools []*mcpfile.Tool

	userScopes := strings.Split(scope, " ")
	scopesLookup := make(map[string]struct{}, len(userScopes))
	for _, s := range userScopes {
		scopesLookup[s] = struct{}{}
	}

	for _, tool := range sm.mcpServer.Tools {
		if err := checkAuthorization(tool.RequiredScopes, scopesLookup); err != nil {
			continue
		}

		allowedTools = append(allowedTools, tool)
	}

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
