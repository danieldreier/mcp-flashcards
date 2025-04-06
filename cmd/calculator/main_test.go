package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestCalculator(t *testing.T) {
	// Get the path to the calculator binary
	// For testing, we'll build it first
	calculatorPath := filepath.Join(os.TempDir(), "calculator")

	// Build the calculator binary
	err := os.WriteFile(calculatorPath, []byte{}, 0755) // Placeholder - we'll actually just use go run

	// Create a client that connects to our calculator server
	// Since we don't have a separate binary yet, we'll use "go run" to run the current directory
	c, err := client.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		".",
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "calculator-test-client",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test case 1: 2+2=4
	addRequest := mcp.CallToolRequest{}
	addRequest.Params.Name = "calculate"
	addRequest.Params.Arguments = map[string]interface{}{
		"operation": "add",
		"x":         float64(2),
		"y":         float64(2),
	}

	addResult, err := c.CallTool(ctx, addRequest)
	if err != nil {
		t.Fatalf("Failed to call add operation: %v", err)
	}

	// Extract result from text content
	if len(addResult.Content) == 0 {
		t.Fatalf("No content returned from add operation")
	}

	addTextContent, ok := addResult.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", addResult.Content[0])
	}

	if addTextContent.Text != "4.00" {
		t.Errorf("Expected 4.00, got %s", addTextContent.Text)
	}

	// Test case 2: 5*5=25
	multiplyRequest := mcp.CallToolRequest{}
	multiplyRequest.Params.Name = "calculate"
	multiplyRequest.Params.Arguments = map[string]interface{}{
		"operation": "multiply",
		"x":         float64(5),
		"y":         float64(5),
	}

	multiplyResult, err := c.CallTool(ctx, multiplyRequest)
	if err != nil {
		t.Fatalf("Failed to call multiply operation: %v", err)
	}

	// Extract result from text content
	if len(multiplyResult.Content) == 0 {
		t.Fatalf("No content returned from multiply operation")
	}

	multiplyTextContent, ok := multiplyResult.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", multiplyResult.Content[0])
	}

	if multiplyTextContent.Text != "25.00" {
		t.Errorf("Expected 25.00, got %s", multiplyTextContent.Text)
	}
}
