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

	// Test Tag Filtering
	t.Run("TestGetDueCardWithTagFilter", func(t *testing.T) {
		// Create more cards with specific tags and due times for filtering test
		// Card A: Math, due 1 hour ago
		err = createTestCard(c, ctx, "Math Card A", "Answer A", []string{"math", "easy"}, -1)
		if err != nil {
			t.Fatalf("Failed to create math card A: %v", err)
		}
		// Card B: Chemistry, due 2 hours ago (should be most due overall)
		err = createTestCard(c, ctx, "Chem Card B", "Answer B", []string{"chemistry", "medium"}, -2)
		if err != nil {
			t.Fatalf("Failed to create chem card B: %v", err)
		}
		// Card C: Math, due 30 mins ago
		err = createTestCard(c, ctx, "Math Card C", "Answer C", []string{"math", "hard"}, -0.5)
		if err != nil {
			t.Fatalf("Failed to create math card C: %v", err)
		}
		// Card D: Chemistry, due 10 mins ago
		err = createTestCard(c, ctx, "Chem Card D", "Answer D", []string{"chemistry", "hard"}, -0.16)
		if err != nil {
			t.Fatalf("Failed to create chem card D: %v", err)
		}
		// Card E: Math & Easy tag, due 5 mins ago
		err = createTestCard(c, ctx, "Math Easy Card E", "Answer E", []string{"math", "easy"}, -0.08)
		if err != nil {
			t.Fatalf("Failed to create math easy card E: %v", err)
		}

		// Total cards = 3 (from outer test) + 5 (from this subtest) = 8
		totalCardsExpected := 8
		// Due cards = 2 (from outer test, excluding Card 3) + 5 (from this subtest) = 7
		dueCardsExpected := 7

		// Test Case 1: Filter by "chemistry"
		getDueCardRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"chemistry"},
		}
		result, err = c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card with chemistry filter: %v", err)
		}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse chemistry filter response JSON: %v", err)
		}
		if !strings.Contains(response.Card.Front, "Chem Card B") {
			t.Errorf("Expected most due chemistry card (Card B), got: %s", response.Card.Front)
		}
		if response.Stats.TotalCards != totalCardsExpected {
			t.Errorf("Expected %d total cards in stats, got %d", totalCardsExpected, response.Stats.TotalCards)
		}
		if response.Stats.DueCards != dueCardsExpected {
			t.Errorf("Expected %d due cards in stats, got %d", dueCardsExpected, response.Stats.DueCards)
		}
		t.Logf("Filter 'chemistry': Got '%s', Stats: %d total, %d due", response.Card.Front, response.Stats.TotalCards, response.Stats.DueCards)

		// Test Case 2: Filter by "math"
		getDueCardRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"math"},
		}
		result, err = c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card with math filter: %v", err)
		}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse math filter response JSON: %v", err)
		}
		if !strings.Contains(response.Card.Front, "Math Card A") { // Card A is due longer ago than C or E
			t.Errorf("Expected most due math card (Card A), got: %s", response.Card.Front)
		}
		if response.Stats.TotalCards != totalCardsExpected {
			t.Errorf("Expected %d total cards in stats, got %d", totalCardsExpected, response.Stats.TotalCards)
		}
		if response.Stats.DueCards != dueCardsExpected {
			t.Errorf("Expected %d due cards in stats, got %d", dueCardsExpected, response.Stats.DueCards)
		}
		t.Logf("Filter 'math': Got '%s', Stats: %d total, %d due", response.Card.Front, response.Stats.TotalCards, response.Stats.DueCards)

		// Test Case 3: Filter by "math" and "easy"
		getDueCardRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"math", "easy"},
		}
		result, err = c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card with math & easy filter: %v", err)
		}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse math & easy filter response JSON: %v", err)
		}
		if !strings.Contains(response.Card.Front, "Math Card A") { // Card A is due longer ago than E
			t.Errorf("Expected most due math & easy card (Card A), got: %s", response.Card.Front)
		}
		if response.Stats.TotalCards != totalCardsExpected {
			t.Errorf("Expected %d total cards in stats, got %d", totalCardsExpected, response.Stats.TotalCards)
		}
		if response.Stats.DueCards != dueCardsExpected {
			t.Errorf("Expected %d due cards in stats, got %d", dueCardsExpected, response.Stats.DueCards)
		}
		t.Logf("Filter 'math', 'easy': Got '%s', Stats: %d total, %d due", response.Card.Front, response.Stats.TotalCards, response.Stats.DueCards)

		// Test Case 4: Filter by non-existent tag "physics"
		getDueCardRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"physics"},
		}
		result, err = c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card with physics filter: %v", err)
		}
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent for physics filter, got %T", result.Content[0])
		}
		if !strings.Contains(textContent.Text, "No cards due for review") {
			t.Errorf("Expected 'No cards due' error for physics filter, got: %s", textContent.Text)
		}
		// Check stats are still returned even with no matching card
		var errorResponse struct {
			Error string    `json:"error"`
			Stats CardStats `json:"stats"` // Expect stats to be included in error response
		}
		err = json.Unmarshal([]byte(textContent.Text), &errorResponse)
		if err == nil { //Unmarshal error is expected if stats aren't present
			if errorResponse.Stats.TotalCards != totalCardsExpected {
				t.Errorf("Expected %d total cards in stats (physics filter error), got %d", totalCardsExpected, errorResponse.Stats.TotalCards)
			}
			if errorResponse.Stats.DueCards != dueCardsExpected {
				t.Errorf("Expected %d due cards in stats (physics filter error), got %d", dueCardsExpected, errorResponse.Stats.DueCards)
			}
			t.Logf("Filter 'physics': Got expected error, Stats: %d total, %d due", errorResponse.Stats.TotalCards, errorResponse.Stats.DueCards)
		} else {
			// If unmarshal failed, it means stats likely weren't included, which is a bug
			t.Errorf("Failed to parse error response for physics filter, stats might be missing: %v. Response text: %s", err, textContent.Text)
		}

		// Test Case 5: No filter (should return most due overall: Card B)
		getDueCardRequest.Params.Arguments = map[string]interface{}{}
		result, err = c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card with no filter: %v", err)
		}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse no filter response JSON: %v", err)
		}
		if !strings.Contains(response.Card.Front, "Chem Card B") { // Card B due 2 hours ago
			t.Errorf("Expected most due overall card (Card B), got: %s", response.Card.Front)
		}
		if response.Stats.TotalCards != totalCardsExpected {
			t.Errorf("Expected %d total cards in stats (no filter), got %d", totalCardsExpected, response.Stats.TotalCards)
		}
		if response.Stats.DueCards != dueCardsExpected {
			t.Errorf("Expected %d due cards in stats (no filter), got %d", dueCardsExpected, response.Stats.DueCards)
		}
		t.Logf("Filter 'none': Got '%s', Stats: %d total, %d due", response.Card.Front, response.Stats.TotalCards, response.Stats.DueCards)
	})
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
	t.Run("TestRatingGood", func(t *testing.T) {
		// Setup client
		c, ctx, cancel, tempFilePath := setupMCPClient(t)
		defer c.Close()
		defer cancel()
		defer os.Remove(tempFilePath)

		// First, create a test card
		front := "What is the capital of France?"
		back := "Paris"
		tags := []string{"geography", "europe", "test-review"}
		hourOffset := -1.0 // Make it due 1 hour ago

		// Create a card for testing
		err := createTestCard(c, ctx, front, back, tags, hourOffset)
		if err != nil {
			t.Fatalf("Failed to create test card: %v", err)
		}

		// We need to get the card ID using get_due_card
		getDueCardRequest := mcp.CallToolRequest{}
		getDueCardRequest.Params.Name = "get_due_card"

		// Call get_due_card to retrieve the card we just created
		dueResult, err := c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card: %v", err)
		}

		// Extract the text content
		dueTextContent, ok := dueResult.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", dueResult.Content[0])
		}

		// Parse the JSON response
		var dueResponse CardResponse
		err = json.Unmarshal([]byte(dueTextContent.Text), &dueResponse)
		if err != nil {
			t.Fatalf("Failed to parse due card response JSON: %v", err)
		}

		// Store the card ID for the review
		cardID := dueResponse.Card.ID

		// Verify we got the right card
		if cardID == "" {
			t.Fatalf("Failed to get valid card ID")
		}
		if dueResponse.Card.Front != front {
			t.Fatalf("Got wrong card, expected front '%s', got '%s'", front, dueResponse.Card.Front)
		}

		// Now submit a review for this card with rating "Good" (3)
		submitReviewRequest := mcp.CallToolRequest{}
		submitReviewRequest.Params.Name = "submit_review"
		submitReviewRequest.Params.Arguments = map[string]interface{}{
			"card_id": cardID,
			"rating":  3.0, // 3 = Good
			"answer":  "Paris is the capital of France",
		}

		result, err := c.CallTool(ctx, submitReviewRequest)
		if err != nil {
			t.Fatalf("Failed to call submit_review: %v", err)
		}

		// Parse the JSON response
		var response ReviewResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		// Verify basic response structure
		if !response.Success {
			t.Error("Review submission should succeed")
		}
		if response.Card.ID != cardID {
			t.Errorf("Card ID mismatch, expected %s, got %s", cardID, response.Card.ID)
		}

		// With rating "Good", the card should have a future due date,
		// but might still be due if we test another card
		t.Logf("Successfully submitted 'Good' review for card: %s", cardID)
		t.Logf("New due date: %v", response.Card.FSRS.Due)
		t.Logf("Card state: %d", response.Card.FSRS.State)
	})

	t.Run("TestRatingEasy", func(t *testing.T) {
		// Setup client
		c, ctx, cancel, tempFilePath := setupMCPClient(t)
		defer c.Close()
		defer cancel()
		defer os.Remove(tempFilePath)

		// First, create a test card
		front := "What is the capital of Spain?"
		back := "Madrid"
		tags := []string{"geography", "europe", "test-review-easy"}
		hourOffset := -1.0 // Make it due 1 hour ago

		// Create a card for testing
		err := createTestCard(c, ctx, front, back, tags, hourOffset)
		if err != nil {
			t.Fatalf("Failed to create test card: %v", err)
		}

		// First call get_due_card to retrieve the card we just created
		getDueCardRequest := mcp.CallToolRequest{}
		getDueCardRequest.Params.Name = "get_due_card"

		// Call get_due_card to retrieve the card
		dueResult, err := c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card: %v", err)
		}

		// Extract the card details
		var dueResponse CardResponse
		err = json.Unmarshal([]byte(dueResult.Content[0].(mcp.TextContent).Text), &dueResponse)
		if err != nil {
			t.Fatalf("Failed to parse due card response JSON: %v", err)
		}

		// Verify we got the right card
		cardID := dueResponse.Card.ID
		if cardID == "" {
			t.Fatalf("Failed to get valid card ID")
		}
		if dueResponse.Card.Front != front {
			t.Fatalf("Got wrong card, expected front '%s', got '%s'", front, dueResponse.Card.Front)
		}

		// Now submit an "Easy" rating (4) for this card
		submitReviewRequest := mcp.CallToolRequest{}
		submitReviewRequest.Params.Name = "submit_review"
		submitReviewRequest.Params.Arguments = map[string]interface{}{
			"card_id": cardID,
			"rating":  4.0, // 4 = Easy
			"answer":  "Madrid is the capital of Spain",
		}

		result, err := c.CallTool(ctx, submitReviewRequest)
		if err != nil {
			t.Fatalf("Failed to call submit_review: %v", err)
		}

		// Parse the JSON response
		var response ReviewResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse response JSON: %v", err)
		}

		// Verify the card state was updated and is no longer New
		if response.Card.FSRS.State <= 0 {
			t.Errorf("Card state should be updated after review, got %d", response.Card.FSRS.State)
		}

		// Verify the card due date was updated (should be far in the future for Easy rating)
		if !response.Card.FSRS.Due.After(time.Now().Add(12 * time.Hour)) {
			t.Error("Card due date should be well in the future after Easy rating")
		}

		t.Logf("Successfully submitted 'Easy' review for card: %s", cardID)
		t.Logf("New due date: %v", response.Card.FSRS.Due)
		t.Logf("Card state: %d", response.Card.FSRS.State)

		// Now try to get a due card again - we should get a different card or an error
		// since the card we just reviewed with "Easy" should not be due
		dueResult2, err := c.CallTool(ctx, getDueCardRequest)
		if err != nil {
			t.Fatalf("Failed to call get_due_card after review: %v", err)
		}

		// Check if we got an error (no cards due) or a different card
		textContent, ok := dueResult2.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", dueResult2.Content[0])
		}

		// If the response contains error about no due cards, that's expected
		if strings.Contains(textContent.Text, "No cards due") {
			t.Logf("No more cards due after rating as Easy - this is expected")
		} else {
			// Otherwise, we should have gotten a different card
			var dueResponse2 CardResponse
			err = json.Unmarshal([]byte(textContent.Text), &dueResponse2)
			if err != nil {
				t.Fatalf("Failed to parse second get_due_card response: %v", err)
			}

			// Make sure we got a different card
			if dueResponse2.Card.ID == cardID {
				t.Errorf("Card rated as Easy should not appear in get_due_card immediately afterward")
			} else {
				t.Logf("Got a different card: %s", dueResponse2.Card.Front)
			}
		}
	})
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

	// First create a card to update
	front := "What is the capital of France?"
	back := "Paris"
	tags := []string{"geography", "europe"}

	err := createTestCard(c, ctx, front, back, tags, 0)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Get the card ID using list_cards
	listCardsRequest := mcp.CallToolRequest{}
	listCardsRequest.Params.Name = "list_cards"

	listResult, err := c.CallTool(ctx, listCardsRequest)
	if err != nil {
		t.Fatalf("Failed to call list_cards: %v", err)
	}

	// Parse the list cards response to get the card ID
	var listResponse ListCardsResponse
	err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &listResponse)
	if err != nil {
		t.Fatalf("Failed to parse list response: %v", err)
	}

	if len(listResponse.Cards) == 0 {
		t.Fatalf("No cards found in storage")
	}

	cardID := listResponse.Cards[0].ID

	// Define updated values
	updatedFront := "What is the capital of France? (Updated)"
	updatedBack := "Paris - City of Light"
	updatedTags := []interface{}{"geography", "europe", "travel"}

	// Call the update_card tool
	updateCardRequest := mcp.CallToolRequest{}
	updateCardRequest.Params.Name = "update_card"
	updateCardRequest.Params.Arguments = map[string]interface{}{
		"card_id": cardID,
		"front":   updatedFront,
		"back":    updatedBack,
		"tags":    updatedTags,
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

	// Verify the update was persisted by retrieving the card again
	listCardsRequest = mcp.CallToolRequest{}
	listCardsRequest.Params.Name = "list_cards"

	listResult, err = c.CallTool(ctx, listCardsRequest)
	if err != nil {
		t.Fatalf("Failed to call list_cards after update: %v", err)
	}

	// Parse the updated list
	var updatedListResponse ListCardsResponse
	err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &updatedListResponse)
	if err != nil {
		t.Fatalf("Failed to parse updated list response: %v", err)
	}

	// Find our updated card
	var updatedCard Card
	cardFound := false
	for _, card := range updatedListResponse.Cards {
		if card.ID == cardID {
			updatedCard = card
			cardFound = true
			break
		}
	}

	if !cardFound {
		t.Fatalf("Updated card with ID %s not found in list response", cardID)
	}

	// Verify the card was updated with the new values
	if updatedCard.Front != updatedFront {
		t.Errorf("Card front not updated. Expected: %s, Got: %s", updatedFront, updatedCard.Front)
	}
	if updatedCard.Back != updatedBack {
		t.Errorf("Card back not updated. Expected: %s, Got: %s", updatedBack, updatedCard.Back)
	}

	// Verify tags were updated
	if len(updatedCard.Tags) != 3 {
		t.Errorf("Expected 3 tags after update, got: %d", len(updatedCard.Tags))
	}

	// Check for travel tag which was newly added
	travelTagFound := false
	for _, tag := range updatedCard.Tags {
		if tag == "travel" {
			travelTagFound = true
			break
		}
	}
	if !travelTagFound {
		t.Errorf("New 'travel' tag not found in updated card tags: %v", updatedCard.Tags)
	}

	t.Logf("Verified card was persisted with updated values - Front: %s, Back: %s, Tags: %v",
		updatedCard.Front, updatedCard.Back, updatedCard.Tags)
}

func TestDeleteCard(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

	// First create a card to delete
	front := "Card to be deleted"
	back := "This card will be deleted in the test"
	tags := []string{"test", "delete"}

	err := createTestCard(c, ctx, front, back, tags, 0)
	if err != nil {
		t.Fatalf("Failed to create test card: %v", err)
	}

	// Create a second card that won't be deleted, to verify targeted deletion
	err = createTestCard(c, ctx, "Card that should remain", "This card should not be deleted", []string{"test", "remain"}, 0)
	if err != nil {
		t.Fatalf("Failed to create second test card: %v", err)
	}

	// Get all cards and count them
	listCardsRequest := mcp.CallToolRequest{}
	listCardsRequest.Params.Name = "list_cards"

	listResult, err := c.CallTool(ctx, listCardsRequest)
	if err != nil {
		t.Fatalf("Failed to call list_cards: %v", err)
	}

	// Parse the list cards response
	var listResponse ListCardsResponse
	err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &listResponse)
	if err != nil {
		t.Fatalf("Failed to parse list response: %v", err)
	}

	initialCardCount := len(listResponse.Cards)
	if initialCardCount < 2 {
		t.Fatalf("Expected at least 2 cards before deletion, got %d", initialCardCount)
	}

	// Find the card to delete (the one with "delete" tag)
	var cardToDeleteID string
	for _, card := range listResponse.Cards {
		for _, tag := range card.Tags {
			if tag == "delete" {
				cardToDeleteID = card.ID
				break
			}
		}
		if cardToDeleteID != "" {
			break
		}
	}

	if cardToDeleteID == "" {
		t.Fatalf("Could not find card with 'delete' tag")
	}

	t.Logf("Initial card count: %d, will delete card ID: %s", initialCardCount, cardToDeleteID)

	// Call the delete_card tool
	deleteCardRequest := mcp.CallToolRequest{}
	deleteCardRequest.Params.Name = "delete_card"
	deleteCardRequest.Params.Arguments = map[string]interface{}{
		"card_id": cardToDeleteID,
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

	// Verify the card was actually deleted by listing cards again
	listResult, err = c.CallTool(ctx, listCardsRequest)
	if err != nil {
		t.Fatalf("Failed to call list_cards after deletion: %v", err)
	}

	// Parse the updated list
	var afterDeleteListResponse ListCardsResponse
	err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &afterDeleteListResponse)
	if err != nil {
		t.Fatalf("Failed to parse list response after deletion: %v", err)
	}

	// Verify card count decreased by exactly 1
	finalCardCount := len(afterDeleteListResponse.Cards)
	if finalCardCount != initialCardCount-1 {
		t.Errorf("Expected %d cards after deletion, got %d", initialCardCount-1, finalCardCount)
	}

	// Verify the specific card is no longer in the list
	for _, card := range afterDeleteListResponse.Cards {
		if card.ID == cardToDeleteID {
			t.Errorf("Card with ID %s was found after deletion", cardToDeleteID)
		}
	}

	// Try to delete the same card again - should fail
	result, err = c.CallTool(ctx, deleteCardRequest)
	if err != nil {
		t.Fatalf("Failed to call delete_card second time: %v", err)
	}

	// The error should be in the response text
	secondDeleteText := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(secondDeleteText, "Card not found") &&
		!strings.Contains(secondDeleteText, "error") {
		t.Errorf("Expected error when deleting non-existent card, but got: %s", secondDeleteText)
	}

	t.Logf("Verified card was correctly deleted and cannot be deleted again")
}

func TestListCards(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

	// First, clear any existing cards by creating a fresh test

	// Create some test cards with specific content for verification
	err := createTestCard(c, ctx, "Europe Card 1", "Paris", []string{"geography", "europe", "capital"}, 0)
	if err != nil {
		t.Fatalf("Failed to create test card 1: %v", err)
	}

	err = createTestCard(c, ctx, "Asia Card", "Tokyo", []string{"geography", "asia", "capital"}, 0)
	if err != nil {
		t.Fatalf("Failed to create test card 2: %v", err)
	}

	err = createTestCard(c, ctx, "Europe Card 2", "Berlin", []string{"geography", "europe", "capital"}, 0)
	if err != nil {
		t.Fatalf("Failed to create test card 3: %v", err)
	}

	// Create a card with different tags for more complex filtering tests
	err = createTestCard(c, ctx, "History Card", "World War II", []string{"history", "europe"}, 0)
	if err != nil {
		t.Fatalf("Failed to create test card 4: %v", err)
	}

	// Create a card that's due in the future to test stats calculation
	err = createTestCard(c, ctx, "Future Card", "This card is due in the future", []string{"test", "future"}, 24) // due in 24 hours
	if err != nil {
		t.Fatalf("Failed to create future test card: %v", err)
	}

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

		// Verify we got all 5 cards
		if len(response.Cards) != 5 {
			t.Errorf("Expected exactly 5 cards, got %d", len(response.Cards))
		}

		// Check that stats were included and match expected values
		if response.Stats.TotalCards != 5 {
			t.Errorf("Total cards should be 5, got %d", response.Stats.TotalCards)
		}

		// 4 cards should be due now, 1 in the future
		if response.Stats.DueCards != 4 {
			t.Errorf("Expected 4 due cards, got %d", response.Stats.DueCards)
		}

		// Verify we got all the expected cards by checking titles
		cardTitles := make(map[string]bool)
		for _, card := range response.Cards {
			cardTitles[card.Front] = true
		}

		expectedTitles := []string{
			"Europe Card 1",
			"Asia Card",
			"Europe Card 2",
			"History Card",
			"Future Card",
		}

		for _, title := range expectedTitles {
			if !cardTitles[title] {
				t.Errorf("Expected card with title '%s' not found", title)
			}
		}

		// Print the response for debugging
		t.Logf("Listed %d cards", len(response.Cards))
		t.Logf("Stats: %d total cards, %d due cards", response.Stats.TotalCards, response.Stats.DueCards)
	})

	// Test 2: List cards with single tag filtering
	t.Run("ListFilteredByEuropeTag", func(t *testing.T) {
		listCardsRequest := mcp.CallToolRequest{}
		listCardsRequest.Params.Name = "list_cards"
		listCardsRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"europe"},
		}

		result, err := c.CallTool(ctx, listCardsRequest)
		if err != nil {
			t.Fatalf("Failed to call list_cards with europe filter: %v", err)
		}

		// Parse the JSON response
		var response ListCardsResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse europe filter response: %v", err)
		}

		// Should be 3 cards with europe tag
		if len(response.Cards) != 3 {
			t.Errorf("Expected 3 cards with 'europe' tag, got %d", len(response.Cards))
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

		t.Logf("Listed %d cards with 'europe' tag", len(response.Cards))
	})

	// Test 3: List cards with multiple tag filtering
	t.Run("ListFilteredByMultipleTags", func(t *testing.T) {
		listCardsRequest := mcp.CallToolRequest{}
		listCardsRequest.Params.Name = "list_cards"
		listCardsRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"europe", "capital"},
		}

		result, err := c.CallTool(ctx, listCardsRequest)
		if err != nil {
			t.Fatalf("Failed to call list_cards with multiple tag filter: %v", err)
		}

		// Parse the JSON response
		var response ListCardsResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse multiple tag filter response: %v", err)
		}

		// Should be 2 cards with both europe and capital tags
		if len(response.Cards) != 2 {
			t.Errorf("Expected 2 cards with both 'europe' and 'capital' tags, got %d", len(response.Cards))
		}

		// Verify all returned cards have both required tags
		for _, card := range response.Cards {
			hasEurope := false
			hasCapital := false

			for _, tag := range card.Tags {
				if tag == "europe" {
					hasEurope = true
				}
				if tag == "capital" {
					hasCapital = true
				}
			}

			if !hasEurope || !hasCapital {
				t.Errorf("Card %s is missing one of the required tags", card.ID)
			}
		}

		t.Logf("Listed %d cards with both 'europe' and 'capital' tags", len(response.Cards))
	})

	// Test 4: List cards with no stats
	t.Run("ListCardsWithoutStats", func(t *testing.T) {
		listCardsRequest := mcp.CallToolRequest{}
		listCardsRequest.Params.Name = "list_cards"
		listCardsRequest.Params.Arguments = map[string]interface{}{
			"include_stats": false,
		}

		result, err := c.CallTool(ctx, listCardsRequest)
		if err != nil {
			t.Fatalf("Failed to call list_cards without stats: %v", err)
		}

		// Parse the JSON response
		var response ListCardsResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse no-stats response: %v", err)
		}

		// Verify stats are zero values when not requested
		if response.Stats.TotalCards != 0 || response.Stats.DueCards != 0 {
			t.Errorf("Stats should be zero when include_stats is false, got total=%d, due=%d",
				response.Stats.TotalCards, response.Stats.DueCards)
		}

		t.Logf("Correctly handled list request without stats")
	})

	// Test 5: List cards with non-existent tag
	t.Run("ListCardsWithNonExistentTag", func(t *testing.T) {
		listCardsRequest := mcp.CallToolRequest{}
		listCardsRequest.Params.Name = "list_cards"
		listCardsRequest.Params.Arguments = map[string]interface{}{
			"tags": []interface{}{"nonexistenttag"},
		}

		result, err := c.CallTool(ctx, listCardsRequest)
		if err != nil {
			t.Fatalf("Failed to call list_cards with non-existent tag: %v", err)
		}

		// Parse the JSON response
		var response ListCardsResponse
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse non-existent tag response: %v", err)
		}

		// Should return 0 cards
		if len(response.Cards) != 0 {
			t.Errorf("Expected 0 cards with non-existent tag, got %d", len(response.Cards))
		}

		t.Logf("Correctly returned empty list for non-existent tag")
	})
}

func TestHelpAnalyzeLearning(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

	// Create some test cards with different ratings

	// Card 1: Create with easy material, will give good rating
	err := createTestCard(c, ctx, "What is 2+2?", "4", []string{"math", "easy"}, -1)
	if err != nil {
		t.Fatalf("Failed to create test card 1: %v", err)
	}

	// Card 2: Create with difficult material, will give low rating
	err = createTestCard(c, ctx, "Explain quantum entanglement", "A quantum phenomenon where pairs of particles remain connected", []string{"physics", "difficult"}, -1)
	if err != nil {
		t.Fatalf("Failed to create test card 2: %v", err)
	}

	// Card 3: Another difficult card
	err = createTestCard(c, ctx, "What is the capital of Kazakhstan?", "Astana (now Nur-Sultan)", []string{"geography", "difficult"}, -1)
	if err != nil {
		t.Fatalf("Failed to create test card 3: %v", err)
	}

	// Get the cards to obtain their IDs
	listCardsRequest := mcp.CallToolRequest{}
	listCardsRequest.Params.Name = "list_cards"

	listResult, err := c.CallTool(ctx, listCardsRequest)
	if err != nil {
		t.Fatalf("Failed to call list_cards: %v", err)
	}

	// Parse the list cards response
	var listResponse ListCardsResponse
	err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &listResponse)
	if err != nil {
		t.Fatalf("Failed to parse list_cards response: %v", err)
	}

	// Add reviews for each card
	for _, card := range listResponse.Cards {
		var rating float64
		if strings.Contains(card.Front, "2+2") {
			rating = 4.0 // Easy card gets high rating
		} else {
			rating = 1.0 // Difficult cards get low rating
		}

		// Submit a review for this card
		submitRequest := mcp.CallToolRequest{}
		submitRequest.Params.Name = "submit_review"
		submitRequest.Params.Arguments = map[string]interface{}{
			"card_id": card.ID,
			"rating":  rating,
			"answer":  "Test answer for " + card.Front,
		}

		_, err := c.CallTool(ctx, submitRequest)
		if err != nil {
			t.Fatalf("Failed to submit review for card %s: %v", card.ID, err)
		}
	}

	// Now call the help_analyze_learning tool
	helpAnalyzeRequest := mcp.CallToolRequest{}
	helpAnalyzeRequest.Params.Name = "help_analyze_learning"

	result, err := c.CallTool(ctx, helpAnalyzeRequest)
	if err != nil {
		t.Fatalf("Failed to call help_analyze_learning: %v", err)
	}

	// Check if we got a response
	if len(result.Content) == 0 {
		t.Fatalf("No content returned from help_analyze_learning")
	}

	// Extract the text content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", result.Content[0])
	}

	// Parse the JSON response as an AnalyzeLearningResponse
	var response AnalyzeLearningResponse
	err = json.Unmarshal([]byte(textContent.Text), &response)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify the response structure and content
	// Should have at least 2 low-scoring cards (the difficult ones)
	if len(response.LowScoringCards) < 2 {
		t.Errorf("Expected at least 2 low-scoring cards, got %d", len(response.LowScoringCards))
	}

	// Should have identified "difficult" as a common tag
	foundDifficultTag := false
	for _, tag := range response.CommonTags {
		if tag == "difficult" {
			foundDifficultTag = true
			break
		}
	}
	if !foundDifficultTag && len(response.LowScoringCards) > 0 {
		t.Errorf("Expected 'difficult' to be identified as a common tag, got: %v", response.CommonTags)
	}

	// Total reviews should match our submitted reviews (3 cards)
	if response.TotalReviews != 3 {
		t.Errorf("Expected 3 total reviews, got %d", response.TotalReviews)
	}

	// Stats should have correct total cards
	if response.Stats.TotalCards != 3 {
		t.Errorf("Expected 3 total cards in stats, got %d", response.Stats.TotalCards)
	}

	// Verify at least one low-scoring card has expected data
	for _, cardData := range response.LowScoringCards {
		// Each card should have review data
		if len(cardData.Reviews) == 0 {
			t.Errorf("Expected card %s to have review data", cardData.Card.ID)
		}

		// Card with rating 1.0 should have avgRating of 1.0
		if cardData.AvgRating == 1.0 {
			// Found a low-scoring card, check if it has the difficult tag
			hasTag := false
			for _, tag := range cardData.Card.Tags {
				if tag == "difficult" {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("Expected low-scoring card to have 'difficult' tag, got: %v", cardData.Card.Tags)
			}
		}
	}

	t.Logf("Successfully called help_analyze_learning tool and verified response")
}
