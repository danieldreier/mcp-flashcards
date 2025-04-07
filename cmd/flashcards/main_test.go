package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// setupMCPClient creates and initializes a new MCP client for testing
func setupMCPClient(t *testing.T) (*client.StdioMCPClient, context.Context, context.CancelFunc, string) {
	// Create temporary storage file for testing
	tempFile, err := os.CreateTemp("", "flashcards-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()

	// Initialize with an empty JSON array to make it a valid JSON file
	err = os.WriteFile(tempFilePath, []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("Failed to initialize temp file: %v", err)
	}

	// Create a client that connects to our flashcards server
	// Pass the temporary file path as a command-line argument
	c, err := client.NewStdioMCPClient(
		"go",
		[]string{}, // Empty ENV
		"run",
		".",
		"-file",
		tempFilePath,
	)
	if err != nil {
		os.Remove(tempFilePath)
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
		os.Remove(tempFilePath)
		t.Fatalf("Failed to initialize: %v", err)
	}

	return c, ctx, cancel, tempFilePath
}

func TestGetDueCard(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

	// First create multiple test cards with different due dates

	// Card 1: Due one hour ago (high priority)
	err := createTestCard(c, ctx, "Card 1 Question", "Card 1 Answer", []string{"test", "priority-high"}, -1)
	if err != nil {
		t.Fatalf("Failed to create test card 1: %v", err)
	}

	// Card 2: Due 30 minutes ago (medium priority)
	err = createTestCard(c, ctx, "Card 2 Question", "Card 2 Answer", []string{"test", "priority-medium"}, -0.5)
	if err != nil {
		t.Fatalf("Failed to create test card 2: %v", err)
	}

	// Card 3: Due in the future (should not be returned)
	err = createTestCard(c, ctx, "Card 3 Question", "Card 3 Answer", []string{"test", "priority-none"}, 1)
	if err != nil {
		t.Fatalf("Failed to create test card 3: %v", err)
	}

	// Now call get_due_card and verify it returns the highest priority card
	getDueCardRequest := mcp.CallToolRequest{}
	getDueCardRequest.Params.Name = "get_due_card"

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

	// Verify we got Card 1 (the most overdue card)
	if !strings.Contains(response.Card.Front, "Card 1") {
		t.Errorf("Expected highest priority card (Card 1), got: %s", response.Card.Front)
	}

	// Verify card details
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
	if response.Stats.TotalCards != 3 {
		t.Errorf("Expected 3 total cards, got %d", response.Stats.TotalCards)
	}
	// We expect 2 due cards (Card 1 and Card 2, but not Card 3 which is due in the future)
	// We'll check the actual number against the expectation
	expectedDueCards := 2
	if response.Stats.DueCards != expectedDueCards {
		t.Errorf("Expected %d due cards, got %d", expectedDueCards, response.Stats.DueCards)
	}

	// Print the response for debugging
	t.Logf("Successfully got card: %s - %s", response.Card.Front, response.Card.Back)
	t.Logf("Stats: %d total cards, %d due cards", response.Stats.TotalCards, response.Stats.DueCards)
}

// createTestCard is a helper function to create a test card with specified due time
func createTestCard(c *client.StdioMCPClient, ctx context.Context, front, back string, tags []string, hourOffset float64) error {
	// Create the card
	createCardRequest := mcp.CallToolRequest{}
	createCardRequest.Params.Name = "create_card"

	// Convert tags to interface slice
	tagInterfaces := make([]interface{}, len(tags))
	for i, tag := range tags {
		tagInterfaces[i] = tag
	}

	// Add hourOffset as a custom parameter for testing
	createCardRequest.Params.Arguments = map[string]interface{}{
		"front":       front,
		"back":        back,
		"tags":        tagInterfaces,
		"hour_offset": hourOffset, // This will be used to set the due date
	}

	result, err := c.CallTool(ctx, createCardRequest)
	if err != nil {
		return fmt.Errorf("failed to call create_card: %w", err)
	}

	// Extract the text content
	if len(result.Content) == 0 {
		return fmt.Errorf("no content returned from create_card")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return fmt.Errorf("expected TextContent, got %T", result.Content[0])
	}

	// Parse the card ID from the response
	var response CreateCardResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		return fmt.Errorf("failed to parse create_card response: %w", err)
	}

	// We now have the card ID, but we need to update its due date
	// For now, since we don't have a direct way to set the due date,
	// we'll use this placeholder
	// In a real implementation, you would add a proper tool to update the due date

	// TODO: Implement a way to directly set due dates for testing
	// For now, this test relies on the implementation of CreateCard setting
	// the due date to now, and the prioritization logic working correctly

	return nil
}

func TestSubmitReview(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

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
	// Setup client with temp storage file
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

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

	// Verify the card was actually stored in the file
	fileContent, err := os.ReadFile(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to read storage file: %v", err)
	}

	// Check if file is not empty
	if len(fileContent) == 0 {
		t.Fatal("Storage file is empty")
	}

	// Parse the storage file contents
	var storageData struct {
		Cards map[string]json.RawMessage `json:"cards"`
	}

	err = json.Unmarshal(fileContent, &storageData)
	if err != nil {
		t.Fatalf("Failed to parse storage file: %v", err)
	}

	// Check if our card ID exists in the storage
	if _, exists := storageData.Cards[response.Card.ID]; !exists {
		t.Errorf("Card with ID %s not found in storage file", response.Card.ID)
	}

	t.Logf("Successfully created and persisted card: %s with ID: %s", response.Card.Front, response.Card.ID)
}

func TestUpdateCard(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

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
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

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
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

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
