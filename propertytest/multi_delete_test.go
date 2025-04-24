package propertytest

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestMultipleDeletes checks for race conditions or errors when deleting multiple cards concurrently.
func TestMultipleDeletes(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 60)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	// Create multiple cards
	numCards := 10
	cardIDs := make([]string, 0, numCards)
	t.Logf("Creating %d cards...", numCards)
	for i := 0; i < numCards; i++ {
		createReq := mcp.CallToolRequest{}
		createReq.Params.Name = "create_card"
		createReq.Params.Arguments = map[string]interface{}{ // Direct assignment
			"front": fmt.Sprintf("Multi-Delete Card %d", i),
			"back":  "Delete Me",
			"tags":  []string{"multi-delete"},
		}
		createRes, err := mcpClient.CallTool(ctx, createReq)
		if err != nil {
			t.Fatalf("Failed to create card %d: %v", i, err)
		}
		createResp, err := parseCreateCardResponse(createRes) // Use helper
		if err != nil {
			t.Fatalf("Failed to parse create response for card %d: %v", i, err)
		}
		cardIDs = append(cardIDs, createResp.Card.ID)
	}
	t.Logf("Created %d cards successfully.", numCards)

	// Delete cards concurrently
	var wg sync.WaitGroup
	deleteErrors := make(chan error, numCards)
	t.Logf("Deleting %d cards concurrently...", numCards)
	for _, cardID := range cardIDs {
		wg.Add(1)
		go func(idToDelete string) {
			defer wg.Done()
			deleteReq := mcp.CallToolRequest{}
			deleteReq.Params.Name = "delete_card"
			deleteReq.Params.Arguments = map[string]interface{}{ // Direct assignment
				"card_id": idToDelete,
			}
			_, err := mcpClient.CallTool(ctx, deleteReq)
			if err != nil {
				// Log the error and send to channel, but don't fail test immediately
				t.Logf("Error deleting card %s: %v", idToDelete, err)
				deleteErrors <- fmt.Errorf("failed to delete %s: %w", idToDelete, err)
			} else {
				t.Logf("Successfully deleted card %s", idToDelete)
			}
		}(cardID)
	}

	wg.Wait()
	close(deleteErrors)
	t.Log("Concurrent deletion process finished.")

	// Check for any errors during deletion
	for err := range deleteErrors {
		if err != nil {
			// Fail the test if any deletion resulted in an unexpected error
			// We might expect "not found" if deletes race, but other errors are suspect.
			if !strings.Contains(err.Error(), "not found") {
				t.Errorf("Unexpected error during concurrent delete: %v", err)
			}
		}
	}

	// Verify all cards are actually gone
	t.Log("Verifying cards are deleted...")
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

	if len(listResp.Cards) > 0 {
		var remainingIDs []string
		for _, card := range listResp.Cards {
			remainingIDs = append(remainingIDs, card.ID)
		}
		t.Errorf("Expected 0 cards after concurrent delete, but found %d: %v", len(listResp.Cards), remainingIDs)
	}
	t.Log("Verified 0 cards remain.")
}

// TestMultipleDeletes_Sequential is a simpler sequential version for baseline comparison.
func TestMultipleDeletes_Sequential(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 60)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	// Create multiple cards
	numCards := 5
	cardIDs := make([]string, 0, numCards)
	t.Logf("Creating %d cards sequentially...", numCards)
	for i := 0; i < numCards; i++ {
		createReq := mcp.CallToolRequest{}
		createReq.Params.Name = "create_card"
		createReq.Params.Arguments = map[string]interface{}{ // Direct assignment
			"front": fmt.Sprintf("Seq Delete Card %d", i),
			"back":  "Delete Me Seq",
		}
		createRes, err := mcpClient.CallTool(ctx, createReq)
		if err != nil {
			t.Fatalf("Failed to create seq card %d: %v", i, err)
		}
		createResp, err := parseCreateCardResponse(createRes) // Use helper
		if err != nil {
			t.Fatalf("Failed to parse create response for seq card %d: %v", i, err)
		}
		cardIDs = append(cardIDs, createResp.Card.ID)
	}
	t.Logf("Created %d cards successfully.", numCards)

	// Delete cards sequentially
	t.Logf("Deleting %d cards sequentially...", numCards)
	for i, cardID := range cardIDs {
		deleteReq := mcp.CallToolRequest{}
		deleteReq.Params.Name = "delete_card"
		deleteReq.Params.Arguments = map[string]interface{}{ // Direct assignment
			"card_id": cardID,
		}
		_, err := mcpClient.CallTool(ctx, deleteReq)
		if err != nil {
			// Sequential delete should not error unless something is wrong
			t.Errorf("Error deleting card %s (index %d) sequentially: %v", cardID, i, err)
		} else {
			t.Logf("Successfully deleted card %s sequentially", cardID)
		}
	}
	t.Log("Sequential deletion process finished.")

	// Verify all cards are actually gone
	t.Log("Verifying cards are deleted...")
	listReq := mcp.CallToolRequest{}
	listReq.Params.Name = "list_cards"
	listRes, err := mcpClient.CallTool(ctx, listReq)
	if err != nil {
		t.Fatalf("Failed to list cards after seq delete: %v", err)
	}
	listResp, err := parseListCardsResponse(listRes) // Use helper
	if err != nil {
		t.Fatalf("Failed to parse list response after seq delete: %v", err)
	}

	if len(listResp.Cards) > 0 {
		var remainingIDs []string
		for _, card := range listResp.Cards {
			remainingIDs = append(remainingIDs, card.ID)
		}
		t.Errorf("Expected 0 cards after sequential delete, but found %d: %v", len(listResp.Cards), remainingIDs)
	}
	t.Log("Verified 0 cards remain.")
}
