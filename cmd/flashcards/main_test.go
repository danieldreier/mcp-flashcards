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

// setupMCPClient creates and initializes a new MCP client for testing
func setupMCPClient(t *testing.T) (*client.StdioMCPClient, context.Context, context.CancelFunc) {
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

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-test-client",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		cancel()
		c.Close()
		t.Fatalf("Failed to initialize: %v", err)
	}

	return c, ctx, cancel
}

func TestGetDueCard(t *testing.T) {
	// Setup client
	c, ctx, cancel := setupMCPClient(t)
	defer c.Close()
	defer cancel()

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

func TestSubmitReview(t *testing.T) {
	// Setup client
	c, ctx, cancel := setupMCPClient(t)
	defer c.Close()
	defer cancel()

	// Call the submit_review tool
	submitReviewRequest := mcp.CallToolRequest{}
	submitReviewRequest.Params.Name = "submit_review"
	submitReviewRequest.Params.Arguments = map[string]interface{}{
		"card_id": "card1",
		"rating":  3.0, // 3 = Good
		"answer":  "Paris is the capital of France",
	}

	result, err := c.CallTool(ctx, submitReviewRequest)
	if err != nil {
		t.Fatalf("Failed to call submit_review: %v", err)
	}

	// Check if we got a response
	if len(result.Content) == 0 {
		t.Fatalf("No content returned from submit_review")
	}

	// Extract the text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response ReviewResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the response structure
	if !response.Success {
		t.Error("Review submission should succeed")
	}
	if response.Message == "" {
		t.Error("Response message is empty")
	}

	// Print the response for debugging
	t.Logf("Submit review response: %s", response.Message)
}

func TestCreateCard(t *testing.T) {
	// Setup client
	c, ctx, cancel := setupMCPClient(t)
	defer c.Close()
	defer cancel()

	// Call the create_card tool
	createCardRequest := mcp.CallToolRequest{}
	createCardRequest.Params.Name = "create_card"
	createCardRequest.Params.Arguments = map[string]interface{}{
		"front": "What is the capital of Germany?",
		"back":  "Berlin",
		"tags":  []interface{}{"geography", "europe"},
	}

	result, err := c.CallTool(ctx, createCardRequest)
	if err != nil {
		t.Fatalf("Failed to call create_card: %v", err)
	}

	// Check if we got a response
	if len(result.Content) == 0 {
		t.Fatalf("No content returned from create_card")
	}

	// Extract the text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response CreateCardResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the response structure
	if response.Card.ID == "" {
		t.Error("Card ID is empty")
	}
	if response.Card.Front != "What is the capital of Germany?" {
		t.Errorf("Card front incorrect, got: %s", response.Card.Front)
	}
	if response.Card.Back != "Berlin" {
		t.Errorf("Card back incorrect, got: %s", response.Card.Back)
	}
	if len(response.Card.Tags) != 2 {
		t.Errorf("Expected 2 tags, got: %d", len(response.Card.Tags))
	}

	// Print the response for debugging
	t.Logf("Successfully created card: %s with ID: %s", response.Card.Front, response.Card.ID)
}

func TestUpdateCard(t *testing.T) {
	// Setup client
	c, ctx, cancel := setupMCPClient(t)
	defer c.Close()
	defer cancel()

	// Call the update_card tool
	updateCardRequest := mcp.CallToolRequest{}
	updateCardRequest.Params.Name = "update_card"
	updateCardRequest.Params.Arguments = map[string]interface{}{
		"card_id": "card1",
		"front":   "What is the capital of France? (Updated)",
		"back":    "Paris - City of Light",
		"tags":    []interface{}{"geography", "europe", "travel"},
	}

	result, err := c.CallTool(ctx, updateCardRequest)
	if err != nil {
		t.Fatalf("Failed to call update_card: %v", err)
	}

	// Check if we got a response
	if len(result.Content) == 0 {
		t.Fatalf("No content returned from update_card")
	}

	// Extract the text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response UpdateCardResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the response structure
	if !response.Success {
		t.Error("Card update should succeed")
	}
	if response.Message == "" {
		t.Error("Response message is empty")
	}

	// Print the response for debugging
	t.Logf("Update card response: %s", response.Message)
}

func TestDeleteCard(t *testing.T) {
	// Setup client
	c, ctx, cancel := setupMCPClient(t)
	defer c.Close()
	defer cancel()

	// Call the delete_card tool
	deleteCardRequest := mcp.CallToolRequest{}
	deleteCardRequest.Params.Name = "delete_card"
	deleteCardRequest.Params.Arguments = map[string]interface{}{
		"card_id": "card1",
	}

	result, err := c.CallTool(ctx, deleteCardRequest)
	if err != nil {
		t.Fatalf("Failed to call delete_card: %v", err)
	}

	// Check if we got a response
	if len(result.Content) == 0 {
		t.Fatalf("No content returned from delete_card")
	}

	// Extract the text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response DeleteCardResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the response structure
	if !response.Success {
		t.Error("Card deletion should succeed")
	}
	if response.Message == "" {
		t.Error("Response message is empty")
	}

	// Print the response for debugging
	t.Logf("Delete card response: %s", response.Message)
}

func TestListCards(t *testing.T) {
	// Setup client
	c, ctx, cancel := setupMCPClient(t)
	defer c.Close()
	defer cancel()

	// Test 1: List all cards without filtering
	t.Run("ListAllCards", func(t *testing.T) {
		listCardsRequest := mcp.CallToolRequest{}
		listCardsRequest.Params.Name = "list_cards"
		listCardsRequest.Params.Arguments = map[string]interface{}{
			"include_stats": true,
		}

		result, err := c.CallTool(ctx, listCardsRequest)
		if err != nil {
			t.Fatalf("Failed to call list_cards: %v", err)
		}

		// Check if we got a response
		if len(result.Content) == 0 {
			t.Fatalf("No content returned from list_cards")
		}

		// Extract the text content
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", result.Content[0])
		}

		// Parse the JSON response
		var response ListCardsResponse
		err = json.Unmarshal([]byte(textContent.Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		// Verify the response structure
		if len(response.Cards) == 0 {
			t.Error("Expected at least one card")
		}

		// Check that stats were included
		if response.Stats.TotalCards <= 0 {
			t.Error("Total cards should be > 0")
		}

		// Print the response for debugging
		t.Logf("Listed %d cards", len(response.Cards))
		t.Logf("Stats: %d total cards, %d due cards", response.Stats.TotalCards, response.Stats.DueCards)
	})

	// Test 2: List cards with tag filtering
	t.Run("ListFilteredCards", func(t *testing.T) {
		listCardsRequest := mcp.CallToolRequest{}
		listCardsRequest.Params.Name = "list_cards"
		listCardsRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"europe"},
		}

		result, err := c.CallTool(ctx, listCardsRequest)
		if err != nil {
			t.Fatalf("Failed to call list_cards with filter: %v", err)
		}

		// Check if we got a response
		if len(result.Content) == 0 {
			t.Fatalf("No content returned from list_cards with filter")
		}

		// Extract the text content
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", result.Content[0])
		}

		// Parse the JSON response
		var response ListCardsResponse
		err = json.Unmarshal([]byte(textContent.Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		// Verify the filtered response
		if len(response.Cards) == 0 {
			t.Error("Expected at least one card with 'europe' tag")
		}

		// Verify all cards have the requested tag
		for _, card := range response.Cards {
			foundTag := false
			for _, tag := range card.Tags {
				if tag == "europe" {
					foundTag = true
					break
				}
			}
			if !foundTag {
				t.Errorf("Card %s doesn't have the 'europe' tag", card.ID)
			}
		}

		// Print the response for debugging
		t.Logf("Listed %d cards with 'europe' tag", len(response.Cards))
	})
}
