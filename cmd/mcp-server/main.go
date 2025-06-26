package main
 
import (
    "context"
    "fmt"
    "log"
    "time"
 
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)
 
func main() {
    s := server.NewMCPServer("test", "1.0.0",
        server.WithToolCapabilities(true),
        server.WithResourceCapabilities(true, true),
    )
 
    // Add real-time tools
    s.AddTool(
        mcp.NewTool("stream_data",
            mcp.WithDescription("Stream data with real-time updates"),
            mcp.WithString("source", mcp.Required()),
            mcp.WithNumber("count", mcp.DefaultNumber(10)),
        ),
        handleStreamData,
    )
 
    s.AddTool(
        mcp.NewTool("monitor_system",
            mcp.WithDescription("Monitor system metrics in real-time"),
            mcp.WithNumber("duration", mcp.DefaultNumber(60)),
        ),
        handleSystemMonitor,
    )
 
    // Add dynamic resources
    s.AddResource(
        mcp.NewResource(
            "metrics://current",
            "Current System Metrics",
            mcp.WithResourceDescription("Real-time system metrics"),
            mcp.WithMIMEType("application/json"),
        ),
        handleCurrentMetrics,
    )
 
    // Start SSE server
    log.Println("Starting SSE server on :8080")
    sseServer := server.NewSSEServer(s)
    if err := sseServer.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}
 
func handleStreamData(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    source := req.GetString("source", "")
    count := req.GetInt("count", 10)
 
    // Get server from context for notifications
    mcpServer := server.ServerFromContext(ctx)
 
    // Stream data with progress updates
    var results []map[string]interface{}
    for i := 0; i < count; i++ {
        // Check for cancellation
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
 
        // Simulate data processing
        data := generateData(source, i)
        results = append(results, data)
 
        // Send progress notification
        if mcpServer != nil {
            err := mcpServer.SendNotificationToClient(ctx, "notifications/progress", map[string]interface{}{
                "progress": i + 1,
                "total":    count,
                "message":  fmt.Sprintf("Processed %d/%d items from %s", i+1, count, source),
            })
            if err != nil {
                log.Printf("Failed to send notification: %v", err)
            }
        }
 
        time.Sleep(100 * time.Millisecond)
    }
 
    return mcp.NewToolResultText(fmt.Sprintf(`{"source":"%s","results":%v,"count":%d}`, 
        source, results, len(results))), nil
}
 
// Helper functions for the examples
func generateData(source string, index int) map[string]interface{} {
    return map[string]interface{}{
        "source": source,
        "index":  index,
        "value":  fmt.Sprintf("data_%d", index),
    }
}
 
func handleSystemMonitor(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    duration := req.GetInt("duration", 60)
    
    mcpServer := server.ServerFromContext(ctx)
 
    // Monitor system for specified duration
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
 
    timeout := time.After(time.Duration(duration) * time.Second)
    var metrics []map[string]interface{}
 
    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-timeout:
            return mcp.NewToolResultText(fmt.Sprintf(`{"duration":%d,"metrics":%v,"samples":%d}`,
                duration, metrics, len(metrics))), nil
        case <-ticker.C:
            // Collect current metrics
            currentMetrics := collectSystemMetrics()
            metrics = append(metrics, currentMetrics)
 
            // Send real-time update
            if mcpServer != nil {
                err := mcpServer.SendNotificationToClient(ctx, "system_metrics", currentMetrics)
                if err != nil {
                    log.Printf("Failed to send system metrics notification: %v", err)
                }
            }
        }
    }
}
 
func collectSystemMetrics() map[string]interface{} {
    // Placeholder implementation
    return map[string]interface{}{
        "cpu":    50.5,
        "memory": 75.2,
        "disk":   30.1,
    }
}
 
func handleCurrentMetrics(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    metrics := collectSystemMetrics()
    return []mcp.ResourceContents{
        mcp.TextResourceContents{
            URI:      req.Params.URI,
            MIMEType: "application/json",
            Text:     fmt.Sprintf(`{"cpu":%.1f,"memory":%.1f,"disk":%.1f}`, metrics["cpu"], metrics["memory"], metrics["disk"]),
        },
    }, nil
}
