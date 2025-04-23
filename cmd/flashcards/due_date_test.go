package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/google/go-cmp/cmp" // Using go-cmp for better diffs
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// setupMCPClient (Reusing from main_test.go implicitly, ensure it's available or copy if needed)
// func setupMCPClient(t *testing.T) (*client.StdioMCPClient, context.Context, context.CancelFunc, string)

// Helper function to create a temporary file path for testing
func tempFilePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()                                         // Creates a temporary directory cleaned up automatically
	return filepath.Join(dir, "test_flashcards_due_date.json") // Use unique name
}

// Helper to call manage_due_dates tool via MCP client
func callManageDueDatesClient(c *client.Client, ctx context.Context, t *testing.T, params map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()
	req := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "manage_due_dates",
			Arguments: params,
		},
	}
	result, err := c.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("client.CallTool raw error: %w", err)
	}

	textContent, extractErr := extractResultText(result)
	if extractErr != nil {
		if result != nil && result.IsError {
			return nil, fmt.Errorf("failed to extract text from error result: %w", extractErr)
		}
		if result == nil || !result.IsError {
			return nil, extractErr
		}
		textContent = "(tool error with unextractable content)"
	}

	if result.IsError {
		return nil, fmt.Errorf("manage_due_dates tool error: %s", textContent)
	}

	var data map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(textContent), &data); unmarshalErr != nil {
		var msgData struct {
			Message string `json:"message"`
		}
		if unmarshalErr2 := json.Unmarshal([]byte(textContent), &msgData); unmarshalErr2 == nil {
			return map[string]interface{}{"message": msgData.Message}, nil
		}
		return nil, fmt.Errorf("failed to unmarshal result JSON '%s': %w", textContent, unmarshalErr)
	}
	return data, nil
}

// Helper to call create_card tool via MCP client
func callCreateCardClient(c *client.Client, ctx context.Context, t *testing.T, params map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()
	req := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "create_card",
			Arguments: params,
		},
	}
	result, err := c.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("client.CallTool raw error: %w", err)
	}

	textContent, extractErr := extractResultText(result)
	if extractErr != nil {
		if result != nil && result.IsError {
			return nil, fmt.Errorf("failed to extract text from error result: %w", extractErr)
		}
		if result == nil || !result.IsError {
			return nil, extractErr
		}
		textContent = "(tool error with unextractable content)"
	}
	if result.IsError {
		return nil, fmt.Errorf("create_card tool error: %s", textContent)
	}

	var data map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(textContent), &data); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal result JSON '%s': %w", textContent, unmarshalErr)
	}
	if cardData, ok := data["card"].(map[string]interface{}); ok {
		return cardData, nil
	}
	return nil, fmt.Errorf("unexpected structure in create_card response JSON '%s': expected top-level 'card' key, got %v", textContent, data)
}

// Helper to call submit_review tool via MCP client
func callSubmitReviewClient(c *client.Client, ctx context.Context, t *testing.T, params map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()
	req := mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Name:      "submit_review",
			Arguments: params,
		},
	}
	result, err := c.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("client.CallTool raw error: %w", err)
	}

	textContent, extractErr := extractResultText(result)
	if extractErr != nil {
		if result != nil && result.IsError {
			return nil, fmt.Errorf("failed to extract text from error result: %w", extractErr)
		}
		if result == nil || !result.IsError {
			return nil, extractErr
		}
		textContent = "(tool error with unextractable content)"
	}
	if result.IsError {
		return nil, fmt.Errorf("submit_review tool error: %s", textContent)
	}

	var data map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(textContent), &data); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal result JSON '%s': %w", textContent, unmarshalErr)
	}
	return data, nil
}

// Helper to extract text content from CallToolResult
func extractResultText(result *mcp.CallToolResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("tool result is nil")
	}
	if len(result.Content) == 0 {
		if result.IsError {
			return "(error result with no content)", nil
		}
		return "", fmt.Errorf("tool result content is empty")
	}
	if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
		return textContent.Text, nil
	}
	if embedContent, ok := mcp.AsEmbeddedResource(result.Content[0]); ok {
		if textResource, ok := mcp.AsTextResourceContents(embedContent.Resource); ok {
			return textResource.Text, nil
		}
	}

	return "", fmt.Errorf("tool result content[0] is not TextContent or Embedded TextResource: %T", result.Content[0])
}

// Helper to read due_date_progress resource via MCP client
func readDueDateProgressClient(c *client.Client, ctx context.Context, t *testing.T) ([]DueDateProgressInfo, error) {
	t.Helper()
	req := mcp.ReadResourceRequest{
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI: "due-date-progress",
		},
	}

	// Log the request for debugging
	t.Logf("Sending ReadResource request with URI: %s", req.Params.URI)

	result, err := c.ReadResource(ctx, req)
	if err != nil {
		t.Logf("ReadResource error: %v", err)
		return nil, fmt.Errorf("client.ReadResource raw error: %w", err)
	}

	t.Logf("ReadResource response received. Contents count: %d", len(result.Contents))

	if len(result.Contents) != 1 {
		return nil, fmt.Errorf("expected 1 resource content, got %d", len(result.Contents))
	}

	t.Logf("Content type: %T", result.Contents[0])

	textContent, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		return nil, fmt.Errorf("expected TextResourceContents, got %T", result.Contents[0])
	}

	t.Logf("Resource content: URI=%s, MIMEType=%s, Text=%s", textContent.URI, textContent.MIMEType, textContent.Text)

	var progressInfos []DueDateProgressInfo
	if unmarshalErr := json.Unmarshal([]byte(textContent.Text), &progressInfos); unmarshalErr != nil {
		t.Logf("JSON unmarshal error: %v", unmarshalErr)
		return nil, fmt.Errorf("failed to unmarshal progress JSON '%s': %w", textContent.Text, unmarshalErr)
	}

	t.Logf("Unmarshaled progress info count: %d", len(progressInfos))
	for i, info := range progressInfos {
		t.Logf("Progress info %d: %+v", i, info)
	}

	return progressInfos, nil
}

// Debug function to check the server resource handling directly
func debugCheckResourceDirectly(t *testing.T, s *FlashcardService, tag string) {
	// Create a context with the service
	ctx := context.WithValue(context.Background(), "service", s)

	// Create a resource request similar to what the client would send
	req := mcp.ReadResourceRequest{
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI: "due-date-progress",
		},
	}

	// Call the handler directly
	t.Logf("Calling handleDueDateProgressResource directly")
	contents, err := handleDueDateProgressResource(ctx, req)
	if err != nil {
		t.Logf("Direct handler error: %v", err)
		return
	}

	t.Logf("Direct handler response - contents count: %d", len(contents))
	if len(contents) == 0 {
		t.Logf("No contents returned from direct handler call")
		return
	}

	// Check the contents
	textContent, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Logf("Content is not TextResourceContents: %T", contents[0])
		return
	}

	t.Logf("Direct resource content: URI=%s, MIMEType=%s, Text=%s",
		textContent.URI, textContent.MIMEType, textContent.Text)

	// Try to parse the content
	var progressInfos []DueDateProgressInfo
	if unmarshalErr := json.Unmarshal([]byte(textContent.Text), &progressInfos); unmarshalErr != nil {
		t.Logf("Direct JSON unmarshal error: %v", unmarshalErr)
		return
	}

	t.Logf("Direct handler - progress info count: %d", len(progressInfos))
	for i, info := range progressInfos {
		t.Logf("Direct progress info %d: %+v", i, info)

		// Check if the tag we're looking for is in this progress info
		if info.Tag == tag {
			t.Logf("Found direct match for tag %s: %+v", tag, info)
		}
	}
}

func TestDueDateWorkflow(t *testing.T) {
	// --- Setup using client helper with pre-initialized storage ---
	// Create temporary file with initial data
	tempFile, err := os.CreateTemp("", "flashcards-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Use a test date that is always in the future for testing
	testDate := "2024-08-20"
	testTopic := "Biology Test"
	testTag := "test-biology-test-" + testDate

	// Initialize with empty structure
	initialJSON := `{
		"cards": {},
		"reviews": [],
		"due_dates": [],
		"last_updated": "2025-04-22T00:00:00Z"
	}`

	err = os.WriteFile(tempFilePath, []byte(initialJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial JSON: %v", err)
	}

	// Create client that connects to our flashcards server with the file
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

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer c.Close()
	defer cancel()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "workflow-test-client",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Create a direct service instance for debugging
	fileStorage := storage.NewFileStorage(tempFilePath)
	if err := fileStorage.Load(); err != nil {
		t.Logf("Failed to initialize debug storage: %v", err)
	}
	debugService := NewFlashcardService(fileStorage)

	var createdDueDateID string
	var createdDueDateTag string

	t.Run("Create Due Date", func(t *testing.T) {
		params := map[string]interface{}{
			"action": "create",
			"topic":  testTopic,
			"date":   testDate,
		}
		result, err := callManageDueDatesClient(c, ctx, t, params)
		if err != nil {
			t.Fatalf("Create due date failed: %v", err)
		}
		if result["topic"] != testTopic {
			t.Errorf("Expected topic '%s', got '%v'", testTopic, result["topic"])
		}
		if id, ok := result["id"].(string); !ok || id == "" {
			t.Fatalf("Created due date missing or invalid 'id': %v", result["id"])
		} else {
			createdDueDateID = id
		}
		if tag, ok := result["tag"].(string); !ok || tag == "" {
			t.Fatalf("Created due date missing or invalid 'tag': %v", result["tag"])
		} else {
			createdDueDateTag = tag
			if tag != testTag {
				t.Errorf("Expected tag '%s', got '%s'", testTag, tag)
			}
		}
		t.Logf("Created Due Date ID: %s, Tag: %s", createdDueDateID, createdDueDateTag)
	})

	// Check file content
	fileContent, err := os.ReadFile(tempFilePath)
	if err != nil {
		t.Logf("Error reading test file %s after Create: %v", tempFilePath, err)
	} else {
		t.Logf("Content of %s after Create Due Date:\n%s", tempFilePath, string(fileContent))
	}

	// Reload debug service to see updated state
	if err := fileStorage.Load(); err != nil {
		t.Logf("Failed to reload debug storage: %v", err)
	}

	t.Run("Check Initial Progress", func(t *testing.T) {
		// Direct debug check
		debugCheckResourceDirectly(t, debugService, createdDueDateTag)

		// Use ReadResource to get the due date progress
		req := mcp.ReadResourceRequest{
			Params: struct {
				URI       string                 `json:"uri"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
			}{
				URI: "due-date-progress",
			},
		}

		result, err := c.ReadResource(ctx, req)
		if err != nil {
			t.Fatalf("Failed to read resource: %v", err)
		}

		t.Logf("Resource response received. Contents count: %d", len(result.Contents))

		if len(result.Contents) == 0 {
			t.Fatalf("No contents returned from ReadResource")
		}

		content, ok := result.Contents[0].(mcp.TextResourceContents)
		if !ok {
			t.Fatalf("Expected TextResourceContents, got %T", result.Contents[0])
		}

		t.Logf("Resource content: %s", content.Text)

		// Should be able to parse this as a list of DueDateProgressInfo
		var progressInfos []DueDateProgressInfo
		err = json.Unmarshal([]byte(content.Text), &progressInfos)
		if err != nil {
			t.Fatalf("Failed to unmarshal progress info: %v", err)
		}

		// We should have exactly one progress info
		if len(progressInfos) != 1 {
			t.Fatalf("Expected 1 progress info, got %d", len(progressInfos))
		}

		info := progressInfos[0]
		t.Logf("Initial Progress: %+v", info)

		// Verify details
		if info.ID != createdDueDateID {
			t.Errorf("Expected progress for ID %s, got %s", createdDueDateID, info.ID)
		}
		if info.Tag != createdDueDateTag {
			t.Errorf("Expected progress for tag %s, got %s", createdDueDateTag, info.Tag)
		}
		if info.TotalCards != 0 {
			t.Errorf("Expected 0 total cards initially, got %d", info.TotalCards)
		}
		if info.MasteredCards != 0 {
			t.Errorf("Expected 0 mastered cards initially, got %d", info.MasteredCards)
		}
		if info.ProgressPercent != 0.0 {
			t.Errorf("Expected 0%% progress initially, got %.2f%%", info.ProgressPercent)
		}
	})

	// Create cards
	cardIDs := make([]string, 3)
	t.Run("Create Cards", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			params := map[string]interface{}{
				"front": fmt.Sprintf("Bio Q %d", i+1),
				"back":  fmt.Sprintf("Bio A %d", i+1),
				"tags":  []interface{}{createdDueDateTag, "biology"},
			}
			result, err := callCreateCardClient(c, ctx, t, params)
			if err != nil {
				t.Fatalf("Create card %d failed: %v", i+1, err)
			}
			if id, ok := result["id"].(string); !ok || id == "" {
				t.Fatalf("Created card %d missing or invalid 'id': %v", i+1, result["id"])
			} else {
				cardIDs[i] = id
			}
		}
		if len(cardIDs) != 3 {
			t.Fatalf("Expected 3 card IDs, got %d", len(cardIDs))
		}
		t.Logf("Created Card IDs: %v", cardIDs)
	})

	// Reload debug service to see updated state
	if err := fileStorage.Load(); err != nil {
		t.Logf("Failed to reload debug storage: %v", err)
	}

	t.Run("Check Progress After Card Creation", func(t *testing.T) {
		// Direct debug check
		debugCheckResourceDirectly(t, debugService, createdDueDateTag)

		// Use ReadResource to get the due date progress
		req := mcp.ReadResourceRequest{
			Params: struct {
				URI       string                 `json:"uri"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
			}{
				URI: "due-date-progress",
			},
		}

		result, err := c.ReadResource(ctx, req)
		if err != nil {
			t.Fatalf("Failed to read resource: %v", err)
		}

		content, ok := result.Contents[0].(mcp.TextResourceContents)
		if !ok {
			t.Fatalf("Expected TextResourceContents, got %T", result.Contents[0])
		}

		t.Logf("Resource content after card creation: %s", content.Text)

		var progressInfos []DueDateProgressInfo
		err = json.Unmarshal([]byte(content.Text), &progressInfos)
		if err != nil {
			t.Fatalf("Failed to unmarshal progress info: %v", err)
		}

		if len(progressInfos) != 1 {
			t.Fatalf("Expected 1 progress info, got %d", len(progressInfos))
		}

		info := progressInfos[0]
		t.Logf("Progress After Creation: %+v", info)

		// Verify details after card creation
		if info.ID != createdDueDateID {
			t.Errorf("Expected progress for ID %s, got %s", createdDueDateID, info.ID)
		}
		if info.TotalCards != 3 {
			t.Errorf("Expected 3 total cards, got %d", info.TotalCards)
		}
		if info.MasteredCards != 0 {
			t.Errorf("Expected 0 mastered cards, got %d", info.MasteredCards)
		}
		if info.ProgressPercent != 0.0 {
			t.Errorf("Expected 0%% progress, got %.2f%%", info.ProgressPercent)
		}
		if info.CardsLeft != 3 {
			t.Errorf("Expected 3 cards left, got %d", info.CardsLeft)
		}
	})

	t.Run("Review Cards", func(t *testing.T) {
		params1 := map[string]interface{}{
			"card_id": cardIDs[0],
			"rating":  float64(gofsrs.Easy),
			"answer":  "Correct Answer A1",
		}
		_, err := callSubmitReviewClient(c, ctx, t, params1)
		if err != nil {
			t.Fatalf("Submit review 1 failed: %v", err)
		}

		params2 := map[string]interface{}{
			"card_id": cardIDs[1],
			"rating":  float64(gofsrs.Again),
			"answer":  "Wrong Answer",
		}
		_, err = callSubmitReviewClient(c, ctx, t, params2)
		if err != nil {
			t.Fatalf("Submit review 2 failed: %v", err)
		}

		params3 := map[string]interface{}{
			"card_id": cardIDs[2],
			"rating":  float64(gofsrs.Good),
			"answer":  "Correct Answer A3",
		}
		_, err = callSubmitReviewClient(c, ctx, t, params3)
		if err != nil {
			t.Fatalf("Submit review 3 failed: %v", err)
		}

		t.Logf("Successfully submitted reviews for all cards")
	})

	// Reload debug service to see updated state
	if err := fileStorage.Load(); err != nil {
		t.Logf("Failed to reload debug storage: %v", err)
	}

	t.Run("Check Progress After Reviews", func(t *testing.T) {
		// Direct debug check
		debugCheckResourceDirectly(t, debugService, createdDueDateTag)

		// Use ReadResource to get the due date progress
		req := mcp.ReadResourceRequest{
			Params: struct {
				URI       string                 `json:"uri"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
			}{
				URI: "due-date-progress",
			},
		}

		result, err := c.ReadResource(ctx, req)
		if err != nil {
			t.Fatalf("Failed to read resource: %v", err)
		}

		content, ok := result.Contents[0].(mcp.TextResourceContents)
		if !ok {
			t.Fatalf("Expected TextResourceContents, got %T", result.Contents[0])
		}

		t.Logf("Resource content after reviews: %s", content.Text)

		var progressInfos []DueDateProgressInfo
		err = json.Unmarshal([]byte(content.Text), &progressInfos)
		if err != nil {
			t.Fatalf("Failed to unmarshal progress info: %v", err)
		}

		if len(progressInfos) != 1 {
			t.Fatalf("Expected 1 progress info, got %d", len(progressInfos))
		}

		info := progressInfos[0]
		t.Logf("Progress After Reviews: %+v", info)

		// Verify details after reviews
		if info.ID != createdDueDateID {
			t.Errorf("Expected progress for ID %s, got %s", createdDueDateID, info.ID)
		}
		if info.TotalCards != 3 {
			t.Errorf("Expected 3 total cards, got %d", info.TotalCards)
		}
		if info.MasteredCards != 1 {
			t.Errorf("Expected 1 mastered card, got %d", info.MasteredCards)
		}

		expectedPercent := (1.0 / 3.0) * 100.0
		if diff := cmp.Diff(expectedPercent, info.ProgressPercent, cmp.Comparer(func(x, y float64) bool { return math.Abs(x-y) < 0.01 })); diff != "" {
			t.Errorf("ProgressPercent mismatch (-want +got):\n%s", diff)
		}

		if info.CardsLeft != 2 {
			t.Errorf("Expected 2 cards left, got %d", info.CardsLeft)
		}

		if info.DaysRemaining <= 0 && info.RequiredPace != 0 {
			t.Errorf("Expected 0 required pace when days remaining <= 0, got %f", info.RequiredPace)
		} else if info.DaysRemaining > 0 && info.RequiredPace <= 0 && info.CardsLeft > 0 {
			t.Errorf("Expected positive required pace, got %f", info.RequiredPace)
		}
	})
}

// TestDueDateProgressClientServer tests the client-server communication specifically
// for due date progress resources. This is a simpler test to identify issues.
func TestDueDateProgressClientServer(t *testing.T) {
	// Set up client with a pre-initialized storage file
	tempFile, err := os.CreateTemp("", "flashcards-client-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Initialize with a JSON structure that includes a due date
	initialJSON := `{
		"cards": {},
		"reviews": [],
		"due_dates": [
			{
				"id": "client-test-id",
				"topic": "Client Server Test",
				"due_date": "2025-05-01T00:00:00Z",
				"tag": "test-client-tag"
			}
		],
		"last_updated": "2025-04-22T00:00:00Z"
	}`

	err = os.WriteFile(tempFilePath, []byte(initialJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial JSON: %v", err)
	}

	// Create a new MCP client that points to our server with the prepared file
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

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "due-date-test-client",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// Create a few cards with the test tag
	testTag := "test-client-tag"

	// Record card IDs for later reviews
	cardIDs := make([]string, 0, 3)

	for i := 1; i <= 3; i++ {
		// Create a card
		createCardRequest := mcp.CallToolRequest{}
		createCardRequest.Params.Name = "create_card"
		createCardRequest.Params.Arguments = map[string]interface{}{
			"front": fmt.Sprintf("Client Q%d", i),
			"back":  fmt.Sprintf("Client A%d", i),
			"tags":  []interface{}{testTag},
		}

		result, err := c.CallTool(ctx, createCardRequest)
		if err != nil {
			t.Fatalf("Failed to call create_card: %v", err)
		}

		// Extract card ID from response
		if len(result.Content) == 0 {
			t.Fatalf("No content returned from create_card")
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", result.Content[0])
		}

		var response CreateCardResponse
		err = json.Unmarshal([]byte(textContent.Text), &response)
		if err != nil {
			t.Fatalf("Failed to parse card response: %v", err)
		}

		cardID := response.Card.ID
		cardIDs = append(cardIDs, cardID)
		t.Logf("Created card %d with ID: %s", i, cardID)
	}

	// Submit an Easy review for one card (should count as mastered)
	if len(cardIDs) > 0 {
		submitReviewRequest := mcp.CallToolRequest{}
		submitReviewRequest.Params.Name = "submit_review"
		submitReviewRequest.Params.Arguments = map[string]interface{}{
			"card_id": cardIDs[0],
			"rating":  float64(gofsrs.Easy),
			"answer":  "Test answer for client",
		}

		_, err := c.CallTool(ctx, submitReviewRequest)
		if err != nil {
			t.Fatalf("Failed to call submit_review: %v", err)
		}
		t.Logf("Successfully submitted Easy review for card: %s", cardIDs[0])
	}

	// Now read the due-date-progress resource and see what we get
	req := mcp.ReadResourceRequest{
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI: "due-date-progress",
		},
	}

	t.Logf("Sending ReadResource request from client")
	result, err := c.ReadResource(ctx, req)
	if err != nil {
		t.Fatalf("Failed to read resource: %v", err)
	}

	t.Logf("Resource response received. Contents count: %d", len(result.Contents))

	if len(result.Contents) == 0 {
		t.Fatalf("No contents returned from ReadResource")
	}

	// Examine and validate the response
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("Expected TextResourceContents, got %T", result.Contents[0])
	}

	t.Logf("Resource URI: %s", content.URI)
	t.Logf("Resource MIME type: %s", content.MIMEType)
	t.Logf("Resource content: %s", content.Text)

	// Should be able to parse this as a list of DueDateProgressInfo
	var progressInfos []DueDateProgressInfo
	err = json.Unmarshal([]byte(content.Text), &progressInfos)
	if err != nil {
		t.Fatalf("Failed to unmarshal progress info: %v", err)
	}

	// We should have exactly one progress info
	if len(progressInfos) != 1 {
		t.Fatalf("Expected 1 progress info, got %d", len(progressInfos))
	}

	info := progressInfos[0]
	t.Logf("Progress info: %+v", info)

	// Verify contents
	expectedTag := "test-client-tag"
	if info.Tag != expectedTag {
		t.Errorf("Expected tag %s, got %s", expectedTag, info.Tag)
	}

	expectedTotalCards := 3
	if info.TotalCards != expectedTotalCards {
		t.Errorf("Expected %d total cards, got %d", expectedTotalCards, info.TotalCards)
	}

	expectedMasteredCards := 1
	if info.MasteredCards != expectedMasteredCards {
		t.Errorf("Expected %d mastered cards, got %d", expectedMasteredCards, info.MasteredCards)
	}
}

// --- Test Helpers ---

// createInitialCards is a helper used within setupClientForDueDateTest
func createInitialCards(c *client.Client, ctx context.Context) ([]string, error) {
	ids := make([]string, 0, 3)
	initialCards := []struct {
		front string
		back  string
		tags  []string
		hours float64 // Due offset
	}{
		{"Q1", "A1", []string{"tag1", "common"}, -2},
		{"Q2", "A2", []string{"tag2", "common"}, -1},
		{"Q3", "A3", []string{"tag1"}, 1},
	}

	for _, data := range initialCards {
		err := createTestCard(c, ctx, data.front, data.back, data.tags, data.hours)
		if err != nil {
			return nil, fmt.Errorf("failed to create card '%s': %w", data.front, err)
		}

		listReq := mcp.CallToolRequest{}
		listReq.Params.Name = "list_cards"

		listRes, listErr := c.CallTool(ctx, listReq) // Use different var name for CallTool error
		if listErr != nil {
			return nil, fmt.Errorf("failed list after create '%s': %w", data.front, listErr)
		}

		var listResponse ListCardsResponse
		if len(listRes.Content) > 0 {
			if textContent, ok := listRes.Content[0].(mcp.TextContent); ok {
				if err = json.Unmarshal([]byte(textContent.Text), &listResponse); err == nil { // Assign to outer err
					foundID := false
					for _, card := range listResponse.Cards {
						if card.Front == data.front {
							ids = append(ids, card.ID)
							foundID = true
							break
						}
					}
					if !foundID {
						// Log or handle case where card wasn't found immediately?
					}
				} else {
					// Log or handle JSON unmarshal error for list response
					return nil, fmt.Errorf("failed to unmarshal list response for '%s': %w", data.front, err)
				}
			}
		}
	}
	if len(ids) != len(initialCards) {
		return nil, fmt.Errorf("could not retrieve all created card IDs, expected %d, got %d", len(initialCards), len(ids))
	}
	return ids, nil
}

// setupClientForDueDateTest sets up the MCP client and initial card state for due date tests.
func setupClientForDueDateTest(t *testing.T) (c *client.Client, ctx context.Context, cancel context.CancelFunc, cleanup func(), cardIDs []string) {
	c, ctx, cancel, tempPath := setupMCPClient(t) // Assumes setupMCPClient is defined (e.g., in main_test.go)
	cleanup = func() {
		cancel()
		if c != nil {
			c.Close()
		}
		os.Remove(tempPath)
	}

	ids, err := createInitialCards(c, ctx)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create initial cards: %v", err)
	}
	cardIDs = ids
	return c, ctx, cancel, cleanup, cardIDs
}

// getDueCardCall calls the get_due_card tool via the MCP client.
func getDueCardCall(t *testing.T, c *client.Client, ctx context.Context) (Card, CardStats, error) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = "get_due_card"
	res, err := c.CallTool(ctx, req) // Declare and assign err here
	if err != nil {
		return Card{}, CardStats{}, fmt.Errorf("CallTool failed: %w", err)
	}
	if len(res.Content) == 0 {
		return Card{}, CardStats{}, fmt.Errorf("no content returned")
	}
	txt, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		return Card{}, CardStats{}, fmt.Errorf("expected TextContent, got %T", res.Content[0])
	}
	var cardResp CardResponse
	// Use = to assign to existing err
	if err = json.Unmarshal([]byte(txt.Text), &cardResp); err != nil {
		var errResp struct {
			Error string
			Stats CardStats
		}
		if jsonErr := json.Unmarshal([]byte(txt.Text), &errResp); jsonErr == nil && errResp.Error != "" {
			t.Logf("Original JSON parse error (ignored): %v", err)
			return Card{}, errResp.Stats, fmt.Errorf(errResp.Error)
		}
		return Card{}, CardStats{}, fmt.Errorf("failed unmarshal CardResponse: %w. Text: %s", err, txt.Text)
	}
	return cardResp.Card, cardResp.Stats, nil
}

// submitReviewCall calls the submit_review tool via the MCP client.
func submitReviewCall(t *testing.T, c *client.Client, ctx context.Context, cardID string, rating int) (Card, error) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = "submit_review"
	req.Params.Arguments = map[string]interface{}{"card_id": cardID, "rating": float64(rating)}
	res, err := c.CallTool(ctx, req) // Declare and assign err here
	if err != nil {
		return Card{}, fmt.Errorf("CallTool failed: %w", err)
	}
	if len(res.Content) == 0 {
		return Card{}, fmt.Errorf("no content returned")
	}
	txt, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		return Card{}, fmt.Errorf("expected TextContent, got %T", res.Content[0])
	}
	var reviewResp ReviewResponse
	// Use = to assign to existing err
	if err = json.Unmarshal([]byte(txt.Text), &reviewResp); err != nil {
		return Card{}, fmt.Errorf("failed unmarshal ReviewResponse: %w. Text: %s", err, txt.Text)
	}
	if !reviewResp.Success {
		return Card{}, fmt.Errorf("review submission failed: %s", reviewResp.Message)
	}
	return reviewResp.Card, nil
}

// listCardsCall calls the list_cards tool via the MCP client.
func listCardsCall(t *testing.T, c *client.Client, ctx context.Context) ([]Card, CardStats, error) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = "list_cards"
	req.Params.Arguments = map[string]interface{}{"include_stats": true}
	res, err := c.CallTool(ctx, req) // Declare and assign err here
	if err != nil {
		return nil, CardStats{}, fmt.Errorf("CallTool failed: %w", err)
	}
	if len(res.Content) == 0 {
		return nil, CardStats{}, fmt.Errorf("no content returned")
	}
	txt, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		return nil, CardStats{}, fmt.Errorf("expected TextContent, got %T", res.Content[0])
	}
	var listResp ListCardsResponse
	// Use = to assign to existing err
	if err = json.Unmarshal([]byte(txt.Text), &listResp); err != nil {
		return nil, CardStats{}, fmt.Errorf("failed unmarshal ListCardsResponse: %w. Text: %s", err, txt.Text)
	}
	return listResp.Cards, listResp.Stats, nil
}

// createCardDirectly interacts with storage to create a card, bypassing MCP.
func createCardDirectly(t *testing.T, s *FlashcardService, front, back string, tags []string) storage.Card {
	t.Helper()
	card, err := s.Storage.CreateCard(front, back, tags)
	if err != nil {
		t.Fatalf("Failed to create card directly: %v", err)
	}
	// Manually save storage as service layer might not auto-save on direct access
	if err := s.Storage.Save(); err != nil {
		t.Fatalf("Failed to save storage after direct create: %v", err)
	}
	return card
}

// updateCardDirectly interacts with storage to update a card.
func updateCardDirectly(t *testing.T, s *FlashcardService, card storage.Card) {
	t.Helper()
	if err := s.Storage.UpdateCard(card); err != nil {
		t.Fatalf("Failed to update card directly: %v", err)
	}
	if err := s.Storage.Save(); err != nil {
		t.Fatalf("Failed to save storage after direct update: %v", err)
	}
}

// setDueDateDirectly updates a card's due date directly in storage (for testing setup).
func setDueDateDirectly(t *testing.T, s *FlashcardService, cardID string, due time.Time) {
	t.Helper()
	card, err := s.Storage.GetCard(cardID)
	if err != nil {
		t.Fatalf("Failed to get card %s for direct due date update: %v", cardID, err)
	}
	card.FSRS.Due = due
	updateCardDirectly(t, s, card) // Use helper to update and save
}

// setupClientAndService creates a service with temp storage and a client connected to it.
func setupClientAndService(t *testing.T) (c *client.Client, s *FlashcardService, ctx context.Context, cancel context.CancelFunc, cleanup func()) {
	t.Helper()
	tempPath := tempFilePath(t)
	fileStorage := storage.NewFileStorage(tempPath)
	var err error
	if err = fileStorage.Load(); err != nil { // Initialize empty store
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	s = NewFlashcardService(fileStorage)

	// Use stdio client pointing to the binary with the temp file
	// Assumes binary is built in parent dir or accessible path
	wd, _ := os.Getwd()
	binPath := filepath.Join(filepath.Dir(wd), "flashcards") // Adjust if needed

	c, err = client.NewStdioMCPClient(
		binPath,
		[]string{},
		"-file",
		tempPath,
	)
	if err != nil {
		os.Remove(tempPath)
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "test-service-client", Version: "0.1"}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		cancel()
		c.Close()
		os.Remove(tempPath)
		t.Fatalf("Failed to initialize client: %v", err)
	}

	cleanup = func() {
		cancel()
		if c != nil {
			c.Close()
		}
		os.Remove(tempPath)
	}
	return c, s, ctx, cancel, cleanup
}
