package propertytest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestReproduceDeleteBug attempts to reproduce the delete bug specifically.
func TestReproduceDeleteBug(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClient(t)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	// 1. Create a card
	createReq := mcp.CallToolRequest{}
	createReq.Params.Name = "create_card"
	createReq.Params.Arguments = map[string]interface{}{ // Direct assignment
		"front": "Delete Bug Test Card Front",
		"back":  "Delete Bug Test Card Back",
	}
	createRes, err := mcpClient.CallTool(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create card: %v", err)
	}
	createResp, err := parseCreateCardResponse(createRes) // Use helper
	if err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}
	cardID := createResp.Card.ID
	t.Logf("Created card with ID: %s", cardID)

	// 2. Delete the card
	deleteReq := mcp.CallToolRequest{}
	deleteReq.Params.Name = "delete_card"
	deleteReq.Params.Arguments = map[string]interface{}{ // Direct assignment
		"card_id": cardID,
	}
	_, err = mcpClient.CallTool(ctx, deleteReq)
	if err != nil {
		// Don't fail immediately, just log. The list check later will confirm.
		t.Logf("Delete card returned error (may be expected if race occurred): %v", err)
	}
	t.Logf("Attempted to delete card ID: %s", cardID)

	// 3. Immediately try to list cards
	listReq := mcp.CallToolRequest{}
	listReq.Params.Name = "list_cards"
	listRes, err := mcpClient.CallTool(ctx, listReq)
	if err != nil {
		t.Fatalf("Failed to list cards after delete: %v", err)
	}
	listResp, err := parseListCardsResponse(listRes) // Use helper
	if err != nil {
		t.Fatalf("Failed to parse list response after delete: %v", err)
	}

	// 4. Verify the card is NOT in the list
	found := false
	for _, card := range listResp.Cards {
		if card.ID == cardID {
			found = true
			break
		}
	}
	if found {
		t.Errorf("Deleted card %s was found in the list response!", cardID)
	} else {
		t.Logf("Verified deleted card %s is not in the list.", cardID)
	}

	// 5. Try deleting the same card again (should likely fail with "not found")
	t.Logf("Attempting to delete card %s again...", cardID)
	deleteRes2, err := mcpClient.CallTool(ctx, deleteReq) // Reuse deleteReq
	if err != nil {
		t.Logf("Second delete attempt failed as expected: %v", err)
	} else {
		// Check if response indicates success or failure
		deleteResp2, err := parseDeleteCardResponse(deleteRes2) // Requires parser
		if err != nil {
			t.Logf("Could not parse second delete response: %v", err)
		} else if deleteResp2.Success {
			t.Errorf("Second delete attempt unexpectedly succeeded for card %s!", cardID)
		} else {
			t.Logf("Second delete attempt failed, response: %s", deleteResp2.Message)
		}
	}
}

// Placeholder for parseDeleteCardResponse - needs actual implementation
func parseDeleteCardResponse(result *mcp.CallToolResult) (DeleteCardResponse, error) {
	if len(result.Content) == 0 {
		return DeleteCardResponse{}, fmt.Errorf("no content in delete response")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return DeleteCardResponse{}, fmt.Errorf("expected TextContent, got %T", result.Content[0])
	}
	var resp DeleteCardResponse
	err := json.Unmarshal([]byte(textContent.Text), &resp)
	if err != nil {
		// Attempt to parse as error response {"error": "..."}
		var errResp map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(textContent.Text), &errResp); jsonErr == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return DeleteCardResponse{Success: false, Message: errMsg}, fmt.Errorf("tool error: %s", errMsg)
			}
		}
		return DeleteCardResponse{}, fmt.Errorf("failed to parse delete response JSON: %w. Text: %s", err, textContent.Text)
	}
	// Return the parsed response, success might be false
	return resp, nil
}
