package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestGetDueCard(t *testing.T) {
	// Get the path to a temporary file for the binary
	// (not actually used, just a placeholder)
	tmpPath := filepath.Join(os.TempDir(), "flashcards")
	_ = os.WriteFile(tmpPath, []byte{}, 0755)

	// Create a client that connects to our flashcards server
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
		Name:    "flashcards-test-client",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Call the get_due_card tool
	getDueCardRequest := mcp.CallToolRequest{}
	getDueCardRequest.Params.Name = "get_due_card"
	// No parameters needed

	result, err := c.CallTool(ctx, getDueCardRequest)
	if err != nil {
		t.Fatalf("Failed to call get_due_card: %v", err)
	}

	// Check if we got a response
	if len(result.Content) == 0 {
		t.Fatalf("No content returned from get_due_card")
	}

	// Extract the text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response CardResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the response structure
	if response.Card.ID == "" {
		t.Error("Card ID is empty")
	}
	if response.Card.Front == "" {
		t.Error("Card front is empty")
	}
	if response.Card.Back == "" {
		t.Error("Card back is empty")
	}

	// Check stats
	if response.Stats.TotalCards <= 0 {
		t.Error("Total cards should be > 0")
	}
	if response.Stats.DueCards <= 0 {
		t.Error("Due cards should be > 0")
	}

	// Print the response for debugging
	t.Logf("Successfully got card: %s - %s", response.Card.Front, response.Card.Back)
	t.Logf("Stats: %d total cards, %d due cards", response.Stats.TotalCards, response.Stats.DueCards)
}
