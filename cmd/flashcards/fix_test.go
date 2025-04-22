package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestFixCardRepetition tests that our fix prevents the same card from being
// returned immediately after submitting a review with a high rating
func TestFixCardRepetition(t *testing.T) {
	// Create temporary storage file for testing
	tempFile, err := os.CreateTemp("", "flashcards-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	t.Logf("Created temporary file: %s", tempFilePath)

	// Initialize with an empty JSON object
	err = os.WriteFile(tempFilePath, []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("Failed to initialize temp file: %v", err)
	}

	// Create a client that connects to our flashcards server
	c, err := client.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		".",
		"-file",
		tempFilePath,
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()
	t.Log("Client created successfully")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-fix-test",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	t.Log("MCP client initialized successfully")

	// 1. Create two cards that are due now (one is more overdue than the other)
	// First card: Due one hour ago (higher priority)
	createCardRequest := mcp.CallToolRequest{}
	createCardRequest.Params.Name = "create_card"
	createCardRequest.Params.Arguments = map[string]interface{}{
		"front":       "First test question",
		"back":        "First test answer",
		"tags":        []interface{}{"test"},
		"hour_offset": -1.0, // Make it due 1 hour ago
	}

	createResult, err := c.CallTool(ctx, createCardRequest)
	if err != nil {
		t.Fatalf("Failed to create first test card: %v", err)
	}

	// Extract the card ID
	createTextContent, ok := createResult.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", createResult.Content[0])
	}

	var createResponse struct {
		Card struct {
			ID string `json:"id"`
		} `json:"card"`
	}
	err = json.Unmarshal([]byte(createTextContent.Text), &createResponse)
	if err != nil {
		t.Fatalf("Failed to parse create_card response: %v", err)
	}
	card1ID := createResponse.Card.ID
	t.Logf("Created first card with ID: %s", card1ID)

	// Second card: Due 30 minutes ago (lower priority)
	createCard2Request := mcp.CallToolRequest{}
	createCard2Request.Params.Name = "create_card"
	createCard2Request.Params.Arguments = map[string]interface{}{
		"front":       "Second test question",
		"back":        "Second test answer",
		"tags":        []interface{}{"test"},
		"hour_offset": -0.5, // Due 30 minutes ago
	}

	createResult2, err := c.CallTool(ctx, createCard2Request)
	if err != nil {
		t.Fatalf("Failed to create second test card: %v", err)
	}

	// Extract the second card ID
	createTextContent2, ok := createResult2.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", createResult2.Content[0])
	}

	var createResponse2 struct {
		Card struct {
			ID string `json:"id"`
		} `json:"card"`
	}
	err = json.Unmarshal([]byte(createTextContent2.Text), &createResponse2)
	if err != nil {
		t.Fatalf("Failed to parse create_card response: %v", err)
	}
	card2ID := createResponse2.Card.ID
	t.Logf("Created second card with ID: %s", card2ID)

	// 2. Get the first due card - should be the first card (due 1 hour ago)
	getDueCardRequest := mcp.CallToolRequest{}
	getDueCardRequest.Params.Name = "get_due_card"

	dueResult, err := c.CallTool(ctx, getDueCardRequest)
	if err != nil {
		t.Fatalf("Failed to call get_due_card: %v", err)
	}

	// Extract the text content
	dueTextContent, ok := dueResult.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", dueResult.Content[0])
	}

	t.Logf("First get_due_card response: %s", dueTextContent.Text)

	// Parse the JSON response
	var firstCardResponse CardResponse
	err = json.Unmarshal([]byte(dueTextContent.Text), &firstCardResponse)
	if err != nil {
		t.Fatalf("Failed to parse due card response: %v", err)
	}

	firstDueCardID := firstCardResponse.Card.ID
	t.Logf("First due card ID: %s", firstDueCardID)

	// Verify we got the first card (most overdue)
	if firstDueCardID != card1ID {
		t.Errorf("First due card should be card1 (ID: %s), got card with ID: %s", card1ID, firstDueCardID)
	}

	// 3. Submit a review for the first card with a high rating
	submitReviewRequest := mcp.CallToolRequest{}
	submitReviewRequest.Params.Name = "submit_review"
	submitReviewRequest.Params.Arguments = map[string]interface{}{
		"card_id": firstDueCardID,
		"rating":  3.0, // Good rating
		"answer":  "Test answer",
	}

	_, err = c.CallTool(ctx, submitReviewRequest)
	if err != nil {
		t.Fatalf("Failed to call submit_review: %v", err)
	}
	t.Log("Submitted review for first card")

	// 4. Get the next due card - should be the second card, not the first one again
	secondDueResult, err := c.CallTool(ctx, getDueCardRequest)
	if err != nil {
		t.Fatalf("Failed to call get_due_card second time: %v", err)
	}

	// Extract the text content
	secondDueTextContent, ok := secondDueResult.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", secondDueResult.Content[0])
	}

	t.Logf("Second get_due_card response: %s", secondDueTextContent.Text)

	// Parse the JSON response
	var secondCardResponse CardResponse
	err = json.Unmarshal([]byte(secondDueTextContent.Text), &secondCardResponse)
	if err != nil {
		t.Fatalf("Failed to parse second due card response: %v", err)
	}

	secondDueCardID := secondCardResponse.Card.ID
	t.Logf("Second due card ID: %s", secondDueCardID)

	// Verify that we got the second card, not the first one again
	if secondDueCardID == firstDueCardID {
		t.Errorf("Fix failed: Same card returned after review")
	} else if secondDueCardID != card2ID {
		t.Errorf("Expected second card (ID: %s), but got different card (ID: %s)", card2ID, secondDueCardID)
	} else {
		t.Log("Fix successful: Different card returned after review")
	}
}
