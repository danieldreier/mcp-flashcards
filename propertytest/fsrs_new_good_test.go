package propertytest

import (
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSNewCardGood tests the specific transition New -> Good -> Review state
func TestFSRSNewCardGood(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClient(t)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	sut := FlashcardSUTFactory(mcpClient, ctx, nil, nil, t)

	// Create a new card
	createCard := &CreateCardCmd{
		Front: "New Card Good Front",
		Back:  "New Card Good Back",
		Tags:  []string{"new-good-test"},
	}
	createResult := createCard.Run(sut)
	createResp, ok := createResult.(CreateCardResponse)
	if !ok {
		t.Fatalf("Failed to create card: %v", createResult)
	}
	cardID := createResp.Card.ID

	if createResp.Card.FSRS.State != gofsrs.New {
		t.Fatalf("Expected initial state New, got %v", createResp.Card.FSRS.State)
	}

	// Submit 'Good' review
	submitReview := &SubmitReviewCmd{
		CardID: cardID,
		Rating: gofsrs.Good,
		Answer: "Test Answer Good",
	}

	result := submitReview.Run(sut)
	reviewResp, ok := result.(ReviewResponse)
	if !ok {
		err, isErr := result.(error)
		if isErr {
			t.Fatalf("SubmitReview failed: %v", err)
		} else {
			t.Fatalf("SubmitReview returned unexpected type: %T", result)
		}
	}

	if !reviewResp.Success {
		t.Fatalf("SubmitReview was not successful: %s", reviewResp.Message)
	}

	// Check the resulting state
	expectedState := gofsrs.Review
	actualState := reviewResp.Card.FSRS.State
	t.Logf("Rating: Good -> State: %v", actualState)
	if actualState != expectedState {
		t.Errorf("Rating Good: Expected state %v, got %v", expectedState, actualState)
	}

	// Check due date is reasonable (should be >= 1 day for Good on New)
	now := time.Now()
	if !reviewResp.Card.FSRS.Due.After(now.Add(23 * time.Hour)) { // Allow some buffer
		t.Errorf("Due date for New->Good rating should be at least ~1 day, got %v (now=%v)",
			reviewResp.Card.FSRS.Due, now)
	}
}
