package propertytest

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"

	// MCP related imports
	"github.com/mark3labs/mcp-go/mcp"
)

// TestFSRSModelComparison directly compares the FSRS model predictions from the property test
// with the actual implementation in the service.
func TestFSRSModelComparison(t *testing.T) {
	// --- Setup: Client for this test (now handles state file creation) ---
	mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 60)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup() // Ensures client, context, AND temp state are cleaned up

	// --- Test Data ---
	testCases := []struct {
		name        string
		front       string
		back        string
		reviews     []gofsrs.Rating // Sequence of ratings
		initialFSRS *gofsrs.Card    // Optional initial FSRS state
	}{
		{
			name:    "New Card Good",
			front:   "Test Front 1",
			back:    "Test Back 1",
			reviews: []gofsrs.Rating{gofsrs.Good},
		},
		{
			name:    "New Card Again -> Good",
			front:   "Test Front 2",
			back:    "Test Back 2",
			reviews: []gofsrs.Rating{gofsrs.Again, gofsrs.Good},
		},
		{
			name:    "New Card Sequence (Good, Hard, Good, Easy, Again, Good)",
			front:   "Test Front 3",
			back:    "Test Back 3",
			reviews: []gofsrs.Rating{gofsrs.Good, gofsrs.Hard, gofsrs.Good, gofsrs.Easy, gofsrs.Again, gofsrs.Good},
		},
		// Add more complex cases, maybe including initial non-new state
	}

	fsrsParams := gofsrs.DefaultParam() // Use default FSRS parameters for both models

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// --- Create Card in MCP/SUT ---
			createReq := mcp.CallToolRequest{}
			createReq.Params.Name = "create_card"
			createReq.Params.Arguments = map[string]interface{}{
				"front": tc.front,
				"back":  tc.back,
			}

			createRes, err := mcpClient.CallTool(ctx, createReq)
			if err != nil {
				t.Fatalf("Failed to create card via MCP: %v", err)
			}
			createResp, err := parseCreateCardResponse(createRes)
			if err != nil {
				t.Fatalf("Failed to parse create card response: %v", err)
			}
			cardID := createResp.Card.ID

			// Declare sutCard as propertytest.Card and copy initial fields
			var sutCard Card // Use the propertytest.Card type
			sutCard.ID = createResp.Card.ID
			sutCard.Front = createResp.Card.Front
			sutCard.Back = createResp.Card.Back
			sutCard.Tags = createResp.Card.Tags
			sutCard.CreatedAt = createResp.Card.CreatedAt
			sutCard.FSRS = createResp.Card.FSRS // FSRS struct is compatible

			// --- Initialize Go-FSRS Model ---
			fsrsModelCard := gofsrs.NewCard()
			if tc.initialFSRS != nil {
				fsrsModelCard = *tc.initialFSRS
			}

			// --- Apply Reviews Sequentially ---
			now := time.Now()
			for i, rating := range tc.reviews {
				// Apply review to SUT via MCP
				submitReq := mcp.CallToolRequest{}
				submitReq.Params.Name = "submit_review"
				submitReq.Params.Arguments = map[string]interface{}{
					"card_id": cardID,
					"rating":  float64(rating),
					"answer":  "test answer",
				}

				submitRes, err := mcpClient.CallTool(ctx, submitReq)
				if err != nil {
					t.Fatalf("Review %d: Failed to submit review via MCP: %v", i+1, err)
				}
				submitResp, err := parseSubmitReviewResponse(submitRes)
				if err != nil {
					t.Fatalf("Review %d: Failed to parse submit review response: %v", i+1, err)
				}
				sutCard = submitResp.Card // Now this assignment is type-correct

				// Apply review to go-fsrs model directly
				schedule := fsrsParams.Repeat(fsrsModelCard, now)
				fsrsModelCard = schedule[rating].Card

				// --- Compare States After Each Review ---
				compareFSRSCards(t, fmt.Sprintf("After Review %d (Rating: %d)", i+1, rating), sutCard.FSRS, fsrsModelCard)

				// Advance time slightly for the next review
				if fsrsModelCard.ScheduledDays > 0 {
					now = now.AddDate(0, 0, int(fsrsModelCard.ScheduledDays)+1)
				} else {
					now = now.Add(time.Minute)
				}
			}
		})
	}
}

// Helper function to compare gofsrs.Card structs with tolerance for float values.
func compareFSRSCards(t *testing.T, context string, sut gofsrs.Card, model gofsrs.Card) {
	t.Helper()
	const tolerance = 1e-4 // Tolerance for floating point comparisons

	if sut.State != model.State {
		t.Errorf("%s: State mismatch: SUT=%v, Model=%v", context, sut.State, model.State)
	}
	if math.Abs(sut.Stability-model.Stability) > tolerance {
		t.Errorf("%s: Stability mismatch: SUT=%.4f, Model=%.4f", context, sut.Stability, model.Stability)
	}
	if math.Abs(sut.Difficulty-model.Difficulty) > tolerance {
		t.Errorf("%s: Difficulty mismatch: SUT=%.4f, Model=%.4f", context, sut.Difficulty, model.Difficulty)
	}
	if sut.ElapsedDays != model.ElapsedDays {
		t.Errorf("%s: ElapsedDays mismatch: SUT=%d, Model=%d", context, sut.ElapsedDays, model.ElapsedDays)
	}
	if sut.ScheduledDays != model.ScheduledDays {
		t.Errorf("%s: ScheduledDays mismatch: SUT=%d, Model=%d", context, sut.ScheduledDays, model.ScheduledDays)
	}
	if sut.Reps != model.Reps {
		t.Errorf("%s: Reps mismatch: SUT=%d, Model=%d", context, sut.Reps, model.Reps)
	}
	if sut.Lapses != model.Lapses {
		t.Errorf("%s: Lapses mismatch: SUT=%d, Model=%d", context, sut.Lapses, model.Lapses)
	}
	if !sut.Due.Equal(model.Due) {
		if sut.Due.Sub(model.Due).Abs() > time.Second {
			t.Errorf("%s: Due date mismatch: SUT=%v, Model=%v", context, sut.Due.Format(time.RFC3339), model.Due.Format(time.RFC3339))
		}
	}
}

// --- Parsing Helpers ---

func parseCreateCardResponse(result *mcp.CallToolResult) (CreateCardResponse, error) {
	if len(result.Content) == 0 {
		return CreateCardResponse{}, fmt.Errorf("no content in create response")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return CreateCardResponse{}, fmt.Errorf("expected TextContent, got %T", result.Content[0])
	}
	var resp CreateCardResponse
	err := json.Unmarshal([]byte(textContent.Text), &resp)
	if err != nil {
		return CreateCardResponse{}, fmt.Errorf("failed to parse create response JSON: %w. Text: %s", err, textContent.Text)
	}
	return resp, nil
}

func parseSubmitReviewResponse(result *mcp.CallToolResult) (ReviewResponse, error) {
	if len(result.Content) == 0 {
		return ReviewResponse{}, fmt.Errorf("no content in review response")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return ReviewResponse{}, fmt.Errorf("expected TextContent, got %T", result.Content[0])
	}
	var resp ReviewResponse
	err := json.Unmarshal([]byte(textContent.Text), &resp)
	if err != nil {
		// Attempt to parse as error response
		var errResp map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(textContent.Text), &errResp); jsonErr == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return ReviewResponse{Success: false, Message: errMsg}, fmt.Errorf("tool error: %s", errMsg)
			}
		}
		return ReviewResponse{}, fmt.Errorf("failed to parse review response JSON: %w. Text: %s", err, textContent.Text)
	}
	if !resp.Success {
		return resp, fmt.Errorf("review failed: %s", resp.Message)
	}
	return resp, nil
}
