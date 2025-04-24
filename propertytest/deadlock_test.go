package propertytest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestDeadlock attempts to reproduce potential deadlocks by sending concurrent requests.
func TestDeadlock(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 300) // 5 minutes timeout
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	// Create a few initial cards
	var cardIDs []string
	for i := 0; i < 5; i++ {
		createReq := mcp.CallToolRequest{}
		createReq.Params.Name = "create_card"
		createReq.Params.Arguments = map[string]interface{}{
			"front": fmt.Sprintf("Deadlock Card %d Front", i),
			"back":  fmt.Sprintf("Deadlock Card %d Back", i),
		}
		createRes, err := mcpClient.CallTool(ctx, createReq)
		if err != nil {
			t.Fatalf("Failed to create initial card %d: %v", i, err)
		}
		createResp, err := parseCreateCardResponse(createRes) // Use parsing helper
		if err != nil {
			t.Fatalf("Failed to parse create response for card %d: %v", i, err)
		}
		cardIDs = append(cardIDs, createResp.Card.ID)
	}
	t.Logf("Created %d initial cards", len(cardIDs))

	// Number of concurrent requests
	numRequests := 50
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(reqNum int) {
			defer wg.Done()
			t.Logf("Starting request %d", reqNum)

			// Choose a random operation
			opType := reqNum % 3 // Simple distribution: 0=List, 1=GetDue, 2=SubmitReview
			var toolName string
			args := make(map[string]interface{})

			switch opType {
			case 0:
				toolName = "list_cards"
			case 1:
				toolName = "get_due_card"
			case 2:
				toolName = "submit_review"
				if len(cardIDs) > 0 {
					args["card_id"] = cardIDs[reqNum%len(cardIDs)] // Cycle through card IDs
					args["rating"] = float64((reqNum % 4) + 1)     // Cycle through ratings 1-4
					args["answer"] = "Concurrent answer"
				} else {
					// Fallback if no cards exist (shouldn't happen with initial creation)
					t.Logf("Request %d: No card IDs available for submit_review, falling back to list_cards", reqNum)
					toolName = "list_cards"
				}
			}

			req := mcp.CallToolRequest{}
			req.Params.Name = toolName
			req.Params.Arguments = args

			// Use a shorter timeout for individual requests within the test
			reqCtx, reqCancel := context.WithTimeout(ctx, 10*time.Second)
			defer reqCancel()

			_, err := mcpClient.CallTool(reqCtx, req)
			if err != nil {
				// Log errors but don't fail the test immediately, as some concurrency issues might manifest as errors
				t.Logf("Request %d (%s) failed: %v", reqNum, toolName, err)
			} else {
				t.Logf("Request %d (%s) completed successfully", reqNum, toolName)
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()
	t.Log("All concurrent requests completed.")
}
