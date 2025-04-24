package propertytest

import (
	"testing"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSNewCardTransitions verifies the state transitions for a new card.
func TestFSRSNewCardTransitions(t *testing.T) {
	// Each test gets its own client and context
	mcpClient, ctx, _, clientCleanup, err := SetupTestClient(t)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup() // Handles client, context, and temp file cleanup

	// Create a SUT instance (cleanup handled by clientCleanup)
	sut := FlashcardSUTFactory(mcpClient, ctx, nil, nil, t)

	// Create a new card
	createCard := &CreateCardCmd{
		Front: "New Card Front",
		Back:  "New Card Back",
		Tags:  []string{"new-card-test"},
	}
	createResult := createCard.Run(sut)
	createResp, ok := createResult.(CreateCardResponse)
	if !ok {
		t.Fatalf("Failed to create card: %v", createResult)
	}
	cardID := createResp.Card.ID

	t.Logf("Initial card state: %v", createResp.Card.FSRS.State)
	if createResp.Card.FSRS.State != gofsrs.New {
		t.Errorf("Expected initial state New, got %v", createResp.Card.FSRS.State)
	}

	// Test transitions for each rating
	testCases := []struct {
		rating   gofsrs.Rating
		expected gofsrs.State
	}{
		{gofsrs.Again, gofsrs.Learning},
		{gofsrs.Hard, gofsrs.Learning},
		{gofsrs.Good, gofsrs.Review},
		{gofsrs.Easy, gofsrs.Review},
	}

	for _, tc := range testCases {
		// Need to recreate the card for each transition test to start from New state
		createResult := createCard.Run(sut)
		createResp, ok := createResult.(CreateCardResponse)
		if !ok {
			t.Fatalf("Failed to re-create card for rating %d: %v", tc.rating, createResult)
		}
		cardID = createResp.Card.ID

		// Submit review
		submitReview := &SubmitReviewCmd{
			CardID: cardID,
			Rating: tc.rating,
			Answer: "Test Answer",
		}

		result := submitReview.Run(sut)
		reviewResp, ok := result.(ReviewResponse)
		if !ok {
			// Handle potential errors from submit_review run
			err, isErr := result.(error)
			if isErr {
				t.Fatalf("SubmitReview failed for rating %d: %v", tc.rating, err)
			} else {
				t.Fatalf("SubmitReview returned unexpected type for rating %d: %T", tc.rating, result)
			}
		}

		if !reviewResp.Success {
			t.Fatalf("SubmitReview was not successful for rating %d: %s", tc.rating, reviewResp.Message)
		}

		// Check the resulting state
		t.Logf("Rating: %d -> State: %v", tc.rating, reviewResp.Card.FSRS.State)
		if reviewResp.Card.FSRS.State != tc.expected {
			t.Errorf("Rating %d: Expected state %v, got %v", tc.rating, tc.expected, reviewResp.Card.FSRS.State)
		}
	}
}
