package propertytest

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestIsolatedReviewTransition tests the problematic Review state transition in a completely isolated process
// This is a last resort for debugging hanging tests
func TestIsolatedReviewTransition(t *testing.T) {
	t.Log("Running isolated review transition test...")

	// We'll run 'go test' directly for the specific test case but with a timeout
	// This ensures we don't hang the overall test process
	testName := "TestDeadlockDiagnosis"
	cmd := exec.Command("go", "test", "-timeout", "15s", "-run", testName, "-v", "./...")

	// Create a context with timeout to prevent the command itself from hanging
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// Capture output for analysis
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Log the output regardless of success/failure
	t.Logf("Test output: \n%s", outputStr)

	if err != nil {
		if strings.Contains(outputStr, "context deadline exceeded") ||
			strings.Contains(outputStr, "signal: killed") {
			t.Errorf("Test process timed out - likely deadlock detected: %v", err)
		} else {
			t.Errorf("Test failed with error: %v", err)
		}
	} else {
		t.Log("Test completed successfully")
	}
}

// TestStateIsolation verifies that separate test clients operate on isolated state.
func TestStateIsolation(t *testing.T) {
	numClients := 3
	var wg sync.WaitGroup
	results := make(chan map[string]int, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// Each goroutine gets its own isolated client and state file
			mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 60)
			if err != nil {
				t.Errorf("Client %d: Failed to setup client: %v", clientID, err)
				results <- nil // Signal error
				return
			}
			defer clientCleanup()

			// Create a unique card for this client
			uniqueFront := fmt.Sprintf("Client %d Card Front", clientID)
			uniqueBack := fmt.Sprintf("Client %d Card Back", clientID)
			createReq := mcp.CallToolRequest{}
			createReq.Params.Name = "create_card"
			createReq.Params.Arguments = map[string]interface{}{ // Direct assignment
				"front": uniqueFront,
				"back":  uniqueBack,
			}
			_, err = mcpClient.CallTool(ctx, createReq)
			if err != nil {
				t.Errorf("Client %d: Failed to create card: %v", clientID, err)
				results <- nil
				return
			}

			// Add a small delay to increase chance of interleaving
			time.Sleep(time.Duration(clientID*50) * time.Millisecond)

			// List cards and check counts
			listReq := mcp.CallToolRequest{}
			listReq.Params.Name = "list_cards"
			listRes, err := mcpClient.CallTool(ctx, listReq)
			if err != nil {
				t.Errorf("Client %d: Failed to list cards: %v", clientID, err)
				results <- nil
				return
			}

			listResp, err := parseListCardsResponse(listRes) // Assuming this parser exists
			if err != nil {
				t.Errorf("Client %d: Failed to parse list response: %v", clientID, err)
				results <- nil
				return
			}

			clientCardCount := 0
			otherCardCount := 0
			for _, card := range listResp.Cards {
				if card.Front == uniqueFront {
					clientCardCount++
				} else {
					otherCardCount++
				}
			}

			results <- map[string]int{
				"clientCards": clientCardCount,
				"otherCards":  otherCardCount,
				"totalCards":  len(listResp.Cards),
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify results
	for i := 0; i < numClients; i++ {
		counts := <-results
		if counts == nil {
			// Error already logged by goroutine
			continue
		}
		t.Logf("Client %d Results: Total=%d, ClientSpecific=%d, Others=%d",
			i, counts["totalCards"], counts["clientCards"], counts["otherCards"])

		if counts["clientCards"] != 1 {
			t.Errorf("Client %d: Expected exactly 1 client-specific card, found %d", i, counts["clientCards"])
		}
		if counts["otherCards"] != 0 {
			t.Errorf("Client %d: Expected 0 cards from other clients, found %d", i, counts["otherCards"])
		}
		if counts["totalCards"] != 1 {
			t.Errorf("Client %d: Expected total of 1 card, found %d", i, counts["totalCards"])
		}
	}
}

// Placeholder for parseListCardsResponse - needs actual implementation
func parseListCardsResponse(result *mcp.CallToolResult) (ListCardsResponse, error) {
	if len(result.Content) == 0 {
		// Return empty list if no content, assuming that's valid
		return ListCardsResponse{Cards: []Card{}}, nil
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return ListCardsResponse{}, fmt.Errorf("expected TextContent, got %T", result.Content[0])
	}
	var resp ListCardsResponse
	err := json.Unmarshal([]byte(textContent.Text), &resp)
	if err != nil {
		return ListCardsResponse{}, fmt.Errorf("failed to parse list cards response JSON: %w. Text: %s", err, textContent.Text)
	}
	return resp, nil
}
