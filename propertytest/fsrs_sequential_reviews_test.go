package propertytest

import (
	"fmt"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSSequentialReviews tests how FSRS handles sequential reviews of the same card,
// verifying correct state transitions between different states of the FSRS algorithm.
func TestFSRSSequentialReviews(t *testing.T) {
	// Setup a client
	mcpClient, ctx, cancel, baseCleanup := SetupPropertyTestClient(t)
	defer func() {
		cancel()
		mcpClient.Close()
		baseCleanup()
	}()

	// Create a SUT
	sut := &FlashcardSUT{
		Client:      mcpClient,
		Ctx:         ctx,
		Cancel:      cancel,
		CleanupFunc: baseCleanup,
		T:           t,
	}

	// Test different sequences of ratings
	sequences := []struct {
		name    string
		ratings []gofsrs.Rating
	}{
		{
			name:    "New_Good_Good", // Test New -> Learning -> Review transition
			ratings: []gofsrs.Rating{gofsrs.Good, gofsrs.Good},
		},
		{
			name:    "New_Again_Good", // Test New -> Learning -> Learning transition
			ratings: []gofsrs.Rating{gofsrs.Again, gofsrs.Good},
		},
		{
			name:    "New_Good_Again", // Test New -> Learning -> Relearning
			ratings: []gofsrs.Rating{gofsrs.Good, gofsrs.Again},
		},
		{
			name:    "New_Easy_Again_Good", // Test New -> Review -> Relearning -> Review
			ratings: []gofsrs.Rating{gofsrs.Easy, gofsrs.Again, gofsrs.Good},
		},
	}

	for _, seq := range sequences {
		t.Run(seq.name, func(t *testing.T) {
			// Create a new card for this sequence
			createCard := &CreateCardCmd{
				Front: fmt.Sprintf("Test Card Front - Sequence %s", seq.name),
				Back:  fmt.Sprintf("Test Card Back - Sequence %s", seq.name),
				Tags:  []string{"test", "sequence", seq.name},
			}
			createResult := createCard.Run(sut)
			createResp, ok := createResult.(CreateCardResponse)
			if !ok {
				t.Fatalf("Failed to create card: %v", createResult)
			}

			cardID := createResp.Card.ID
			t.Logf("Created test card with ID: %s, Initial FSRS State: %v",
				cardID, createResp.Card.FSRS.State)

			// Keep track of the current card state for our model
			currentCard := Card{
				ID:    cardID,
				Front: fmt.Sprintf("Test Card Front - Sequence %s", seq.name),
				Back:  fmt.Sprintf("Test Card Back - Sequence %s", seq.name),
				Tags:  []string{"test", "sequence", seq.name},
				FSRS:  createResp.Card.FSRS,
			}

			// Apply each rating in sequence
			for i, rating := range seq.ratings {
				t.Logf("Applying rating %d (%d of %d)",
					rating, i+1, len(seq.ratings))

				// Create a review command for this step
				submitReview := &SubmitReviewCmd{
					CardID: cardID,
					Rating: rating,
					Answer: fmt.Sprintf("Test answer for rating %d (step %d)",
						rating, i+1),
				}

				// Create a mock state with the current card state
				mockState := &CommandState{
					Cards: map[string]Card{
						cardID: currentCard,
					},
					KnownRealIDs: []string{cardID},
					LastRealID:   cardID,
					T:            t,
				}

				// Calculate the expected next state using our model
				submitReview.NextState(mockState)

				// Log the expected values
				t.Logf("Current state before review: %v", currentCard.FSRS.State)
				t.Logf("Expected state after review: %v", submitReview.ExpectedFSRSState)

				// Run the actual command
				result := submitReview.Run(sut)

				// Verify the result
				reviewResp, ok := result.(ReviewResponse)
				if !ok {
					t.Fatalf("Failed to submit review: %v", result)
				}

				if !reviewResp.Success {
					t.Fatalf("Review was not successful: %s", reviewResp.Message)
				}

				// Log actual values
				t.Logf("Actual FSRS state after review: %v", reviewResp.Card.FSRS.State)
				t.Logf("Actual due date: %v", reviewResp.Card.FSRS.Due)

				// Verify predictions match the implementation
				if reviewResp.Card.FSRS.State != submitReview.ExpectedFSRSState {
					t.Errorf("FSRS state mismatch for rating %d (step %d): expected %v, got %v",
						rating, i+1, submitReview.ExpectedFSRSState, reviewResp.Card.FSRS.State)
				}

				// Check due date
				timeDiff := reviewResp.Card.FSRS.Due.Sub(submitReview.ExpectedDueDate).Abs()
				t.Logf("Due date difference: %v", timeDiff)

				if timeDiff > 5*time.Hour {
					t.Errorf("Due date difference too large for rating %d (step %d): %v",
						rating, i+1, timeDiff)
				}

				// Update our model of the current card state for the next iteration
				currentCard.FSRS = reviewResp.Card.FSRS
			}

			// After all reviews, verify the final state is as expected
			expectedFinalState := getExpectedFinalState(seq.ratings)

			t.Logf("Final state after sequence %s: %v",
				seq.name, currentCard.FSRS.State)

			if currentCard.FSRS.State != expectedFinalState {
				t.Errorf("Final state mismatch for sequence %s: expected %v, got %v",
					seq.name, expectedFinalState, currentCard.FSRS.State)
			}
		})
	}
}

// getExpectedFinalState returns the expected final state for a sequence of ratings
// based on FSRS algorithm documentation
func getExpectedFinalState(ratings []gofsrs.Rating) gofsrs.State {
	if len(ratings) == 0 {
		return gofsrs.New // Default, empty sequence
	}

	// For the sequences we're testing, we know what the final states should be
	// New -> Good -> Good = Review (graduated after second good rating)
	// New -> Again -> Good = Learning (still in learning phase)
	// New -> Good -> Again = Relearning (went back to relearning after forgetting)
	// New -> Easy -> Again -> Good = Review (back to review after relearning)

	if len(ratings) >= 2 {
		// New -> Good -> Good = Review
		if ratings[0] == gofsrs.Good && ratings[1] == gofsrs.Good {
			return gofsrs.Review
		}

		// New -> Again -> Good = Learning
		if ratings[0] == gofsrs.Again && ratings[1] == gofsrs.Good {
			return gofsrs.Learning
		}

		// New -> Good -> Again = Relearning
		if ratings[0] == gofsrs.Good && ratings[1] == gofsrs.Again {
			return gofsrs.Relearning
		}
	}

	if len(ratings) >= 3 {
		// New -> Easy -> Again -> Good = Review
		if ratings[0] == gofsrs.Easy && ratings[1] == gofsrs.Again && ratings[2] == gofsrs.Good {
			return gofsrs.Review
		}
	}

	// Default fallback - would need more comprehensive logic for other sequences
	return gofsrs.Learning
}
