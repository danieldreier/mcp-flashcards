package propertytest

import (
	"fmt"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSNewCardTransitions tests FSRS state transitions specifically for new cards
// with different ratings to verify correct state changes and due date calculations.
func TestFSRSNewCardTransitions(t *testing.T) {
	// Setup a client using the common test function
	mcpClient, ctx, cancel, baseCleanup := SetupTestClient(t)
	defer func() {
		cancel()
		mcpClient.Close()
		baseCleanup()
	}()

	// Create a SUT using the factory function
	sut := FlashcardSUTFactory(mcpClient, ctx, cancel, baseCleanup, t)

	// Direct test of the library behavior for reference values
	t.Run("DirectLibraryBehavior", func(t *testing.T) {
		// Create a reference set of raw values from the go-fsrs library
		tempNow := time.Now()
		tempNewCard := gofsrs.NewCard() // A pristine new card
		tempParams := gofsrs.DefaultParam()

		for _, rating := range []gofsrs.Rating{gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy} {
			tempSchedulingInfo := tempParams.Repeat(tempNewCard, tempNow)
			t.Logf("Raw go-fsrs for new card, rating %d: State=%v, Due=%v",
				rating,
				tempSchedulingInfo[rating].Card.State,
				tempSchedulingInfo[rating].Card.Due.Sub(tempNow))
		}
	})

	// Test each rating with a new card each time
	ratings := []gofsrs.Rating{gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy}

	for _, rating := range ratings {
		t.Run(fmt.Sprintf("Rating_%d", rating), func(t *testing.T) {
			// Create a new card for this rating
			createCard := &CreateCardCmd{
				Front: fmt.Sprintf("Test New Card Front - Rating %d", rating),
				Back:  fmt.Sprintf("Test New Card Back - Rating %d", rating),
				Tags:  []string{"test", fmt.Sprintf("rating-%d", rating)},
			}
			createResult := createCard.Run(sut)
			createResp, ok := createResult.(CreateCardResponse)
			if !ok {
				t.Fatalf("Failed to create card: %v", createResult)
			}

			cardID := createResp.Card.ID
			t.Logf("Created test card with ID: %s, Initial FSRS State: %v",
				cardID, createResp.Card.FSRS.State)

			// Verify the card is in the New state
			if createResp.Card.FSRS.State != gofsrs.New {
				t.Fatalf("Expected new card to have state New (0), got %v",
					createResp.Card.FSRS.State)
			}

			// Create a review command
			submitReview := &SubmitReviewCmd{
				CardID: cardID,
				Rating: rating,
				Answer: fmt.Sprintf("Test answer for rating %d", rating),
			}

			// Create a mock state to track the expected FSRS state changes
			mockState := &CommandState{
				Cards: map[string]Card{
					cardID: {
						ID:    cardID,
						Front: fmt.Sprintf("Test New Card Front - Rating %d", rating),
						Back:  fmt.Sprintf("Test New Card Back - Rating %d", rating),
						Tags:  []string{"test", fmt.Sprintf("rating-%d", rating)},
						FSRS:  createResp.Card.FSRS,
					},
				},
				KnownRealIDs: []string{cardID},
				LastRealID:   cardID,
				T:            t,
			}

			// Call NextState to populate the expected FSRS state and due date
			submitReview.NextState(mockState)

			// Print the expected values for comparison
			t.Logf("Expected FSRS state for rating %d: %v",
				rating, submitReview.ExpectedFSRSState)
			t.Logf("Expected due date for rating %d: %v",
				rating, submitReview.ExpectedDueDate)

			// Run the command
			result := submitReview.Run(sut)

			// Verify the result
			reviewResp, ok := result.(ReviewResponse)
			if !ok {
				t.Fatalf("Failed to submit review: %v", result)
			}

			if !reviewResp.Success {
				t.Fatalf("Review was not successful: %s", reviewResp.Message)
			}

			// Check the FSRS state and due date
			t.Logf("Actual FSRS state: %v", reviewResp.Card.FSRS.State)
			t.Logf("Actual due date: %v", reviewResp.Card.FSRS.Due)

			// Verify state matches expectation
			if reviewResp.Card.FSRS.State != submitReview.ExpectedFSRSState {
				t.Errorf("FSRS state mismatch for rating %d: expected %v, got %v",
					rating, submitReview.ExpectedFSRSState, reviewResp.Card.FSRS.State)
			}

			// Verify due date is reasonable
			timeDiff := reviewResp.Card.FSRS.Due.Sub(submitReview.ExpectedDueDate).Abs()
			t.Logf("Due date difference: %v", timeDiff)

			// Define expected state transitions for each rating based on FSRS algorithm
			// According to documentation/common implementations:

			// Check for correct state transition based on rating
			var expectedState gofsrs.State
			switch rating {
			case gofsrs.Again, gofsrs.Hard, gofsrs.Good:
				expectedState = gofsrs.Learning
			case gofsrs.Easy:
				expectedState = gofsrs.Review
			}

			if reviewResp.Card.FSRS.State != expectedState {
				t.Errorf("Expected new card with rating %d to transition to state %v, got %v",
					rating, expectedState, reviewResp.Card.FSRS.State)
			}

			// Verify due date is in the future
			now := time.Now()
			if !reviewResp.Card.FSRS.Due.After(now) {
				t.Errorf("Due date for rating %d should be in the future", rating)
			}

			// Verify due date intervals are reasonable based on rating
			dueInterval := reviewResp.Card.FSRS.Due.Sub(now)

			switch rating {
			case gofsrs.Again:
				// Should be very soon (1-5 minutes)
				if dueInterval < 30*time.Second || dueInterval > 10*time.Minute {
					t.Errorf("Due interval for 'Again' rating is unexpected: %v", dueInterval)
				}
			case gofsrs.Hard:
				// Should be soon (5-10 minutes)
				if dueInterval < 1*time.Minute || dueInterval > 30*time.Minute {
					t.Errorf("Due interval for 'Hard' rating is unexpected: %v", dueInterval)
				}
			case gofsrs.Good:
				// Should be moderate (10-15 minutes for Learning state)
				if dueInterval < 5*time.Minute || dueInterval > 60*time.Minute {
					t.Errorf("Due interval for 'Good' rating is unexpected: %v", dueInterval)
				}
			case gofsrs.Easy:
				// Should be several days for Review state
				if dueInterval < 24*time.Hour || dueInterval > 30*24*time.Hour {
					t.Errorf("Due interval for 'Easy' rating is unexpected: %v", dueInterval)
				}
			}
		})
	}
}
