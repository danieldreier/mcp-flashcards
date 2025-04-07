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

// TestPersistenceAcrossSessions verifies that cards are persisted between server sessions
func TestPersistenceAcrossSessions(t *testing.T) {
	// Create a specific file path for this test
	tempFilePath := os.TempDir() + "/flashcards-persistence-test.json"

	// Clean up any existing file from previous test runs
	_ = os.Remove(tempFilePath)

	// First session: Create cards
	t.Log("Starting first session")
	createCards(t, tempFilePath)

	// Second session: Verify cards exist
	t.Log("Starting second session")
	verifyCards(t, tempFilePath)
}

// createCards creates several flashcards in the first session
func createCards(t *testing.T, filePath string) {
	// Create a client that connects to our flashcards server
	c, err := client.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		".",
		"-file",
		filePath,
	)
	if err != nil {
		t.Fatalf("Failed to create client for first session: %v", err)
	}
	defer c.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-persistence-test",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize first session: %v", err)
	}

	// Create 3 test cards
	cards := []struct {
		front string
		back  string
		tags  []string
	}{
		{"What is the capital of France?", "Paris", []string{"geography", "europe"}},
		{"What is the capital of Japan?", "Tokyo", []string{"geography", "asia"}},
		{"What is the capital of Brazil?", "Brasília", []string{"geography", "south-america"}},
	}

	// Keep track of created card IDs
	var cardIDs []string

	for _, card := range cards {
		// Create the card
		createCardRequest := mcp.CallToolRequest{}
		createCardRequest.Params.Name = "create_card"

		// Convert tags to interface slice
		tagInterfaces := make([]interface{}, len(card.tags))
		for i, tag := range card.tags {
			tagInterfaces[i] = tag
		}

		createCardRequest.Params.Arguments = map[string]interface{}{
			"front": card.front,
			"back":  card.back,
			"tags":  tagInterfaces,
		}

		result, err := c.CallTool(ctx, createCardRequest)
		if err != nil {
			t.Fatalf("Failed to create card: %v", err)
		}

		// Parse the JSON response to get the card ID
		var response CreateCardResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse create card response: %v", err)
		}

		cardIDs = append(cardIDs, response.Card.ID)
		t.Logf("Created card: %s with ID: %s", card.front, response.Card.ID)
	}

	// Verify we have a non-empty storage file
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read storage file after creation: %v", err)
	}

	if len(fileContent) == 0 {
		t.Fatal("Storage file is empty after creating cards")
	}

	t.Logf("Successfully created %d cards in the first session", len(cards))
	t.Logf("Storage file size: %d bytes", len(fileContent))
}

// verifyCards connects to a new server instance and verifies the cards exist
func verifyCards(t *testing.T, filePath string) {
	// Create a new client that connects to a fresh server instance
	c, err := client.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		".",
		"-file",
		filePath,
	)
	if err != nil {
		t.Fatalf("Failed to create client for second session: %v", err)
	}
	defer c.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-persistence-test",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize second session: %v", err)
	}

	// List all cards
	listCardsRequest := mcp.CallToolRequest{}
	listCardsRequest.Params.Name = "list_cards"
	listCardsRequest.Params.Arguments = map[string]interface{}{
		"include_stats": true,
	}

	result, err := c.CallTool(ctx, listCardsRequest)
	if err != nil {
		t.Fatalf("Failed to list cards in second session: %v", err)
	}

	// Parse the JSON response
	var response ListCardsResponse
	err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse list cards response: %v", err)
	}

	// Verify we have the expected number of cards
	expectedCardCount := 3
	if len(response.Cards) != expectedCardCount {
		t.Errorf("Expected %d cards, but got %d", expectedCardCount, len(response.Cards))
	}

	// Check for expected card content
	cardsByFront := make(map[string]Card)
	for _, card := range response.Cards {
		cardsByFront[card.Front] = card
	}

	expectedCards := []struct {
		front string
		back  string
	}{
		{"What is the capital of France?", "Paris"},
		{"What is the capital of Japan?", "Tokyo"},
		{"What is the capital of Brazil?", "Brasília"},
	}

	for _, expected := range expectedCards {
		card, exists := cardsByFront[expected.front]
		if !exists {
			t.Errorf("Card with front '%s' not found in second session", expected.front)
			continue
		}

		if card.Back != expected.back {
			t.Errorf("Card back mismatch for '%s': expected '%s', got '%s'",
				expected.front, expected.back, card.Back)
		}
	}

	t.Logf("Successfully verified persistence of %d cards across sessions", len(response.Cards))
}
