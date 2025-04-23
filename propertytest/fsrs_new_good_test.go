package propertytest

import (
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSNewCardGoodRating specifically focuses on the New -> Good transition
// that was failing in the original test.
func TestFSRSNewCardGoodRating(t *testing.T) {
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

	// Direct library test for reference
	t.Run("DirectLibraryReference", func(t *testing.T) {
		now := time.Now()
		// Test with pristine new card
		newCard := gofsrs.NewCard()
		params := gofsrs.DefaultParam()

		schedulingInfo := params.Repeat(newCard, now)
		goodRatingResult := schedulingInfo[gofsrs.Good]

		t.Logf("Direct library - New card rated Good: State=%v, Due=%v (in %v)",
			goodRatingResult.Card.State,
			goodRatingResult.Card.Due,
			goodRatingResult.Card.Due.Sub(now))

		// Reference the documentation for clarity
		t.Logf("Expected state transitions in FSRS:")
		t.Logf("New -> Again: Learning")
		t.Logf("New -> Hard: Learning")
		t.Logf("New -> Good: Learning")
		t.Logf("New -> Easy: Review")
	})

	// Multiple service tests with debugging
	for i := 0; i < 3; i++ {
		t.Run("ServiceTest", func(t *testing.T) {
			// Create a new card
			createCard := &CreateCardCmd{
				Front: "Test New Card Front - Good Rating Test",
				Back:  "Test New Card Back - Good Rating Test",
				Tags:  []string{"test", "new-good-test"},
			}
			createResult := createCard.Run(sut)
			createResp, ok := createResult.(CreateCardResponse)
			if !ok {
				t.Fatalf("Failed to create card: %v", createResult)
			}

			cardID := createResp.Card.ID
			t.Logf("Created test card with ID: %s, Initial FSRS State: %v",
				cardID, createResp.Card.FSRS.State)

			// Verify it's in the New state
			if createResp.Card.FSRS.State != gofsrs.New {
				t.Fatalf("Created card is not in New state, got: %v", createResp.Card.FSRS.State)
			}

			// DEEP DEBUGGING: Print the exact FSRS fields of the card
			t.Logf("Card FSRS before review: State=%v, Due=%v, LastReview=%v",
				createResp.Card.FSRS.State,
				createResp.Card.FSRS.Due,
				createResp.Card.FSRS.LastReview)

			// Calculate expected state directly with the library
			now := time.Now()
			libraryCard := gofsrs.NewCard()
			params := gofsrs.DefaultParam()
			libraryResult := params.Repeat(libraryCard, now)[gofsrs.Good]

			t.Logf("Library prediction: State=%v, Due=%v",
				libraryResult.Card.State, libraryResult.Card.Due)

			// Create review command for the test
			submitReview := &SubmitReviewCmd{
				CardID: cardID,
				Rating: gofsrs.Good,
				Answer: "Test answer for New->Good transition",
			}

			// Create a model of the expected state change
			cardModel := Card{
				ID:    cardID,
				Front: "Test New Card Front - Good Rating Test",
				Back:  "Test New Card Back - Good Rating Test",
				Tags:  []string{"test", "new-good-test"},
				FSRS:  createResp.Card.FSRS,
			}

			mockState := &CommandState{
				Cards: map[string]Card{
					cardID: cardModel,
				},
				KnownRealIDs: []string{cardID},
				LastRealID:   cardID,
				T:            t,
			}

			// Calculate expected state in the model
			submitReview.NextState(mockState)

			t.Logf("Model prediction: State=%v, Due=%v",
				submitReview.ExpectedFSRSState, submitReview.ExpectedDueDate)

			// Execute the review
			result := submitReview.Run(sut)

			// Verify review result
			reviewResp, ok := result.(ReviewResponse)
			if !ok {
				t.Fatalf("Failed to submit review: %v", result)
			}

			if !reviewResp.Success {
				t.Fatalf("Review was not successful: %s", reviewResp.Message)
			}

			// Log actual values
			t.Logf("Actual result: State=%v, Due=%v",
				reviewResp.Card.FSRS.State, reviewResp.Card.FSRS.Due)

			// Explicit comparison of all results
			t.Logf("Comparison summary:")
			t.Logf("Library expected State=%v", libraryResult.Card.State)
			t.Logf("Model predicted State=%v", submitReview.ExpectedFSRSState)
			t.Logf("Service returned State=%v", reviewResp.Card.FSRS.State)

			// Verify state is correct
			if libraryResult.Card.State != gofsrs.Learning {
				t.Errorf("Library expected Learning state (1), got: %v", libraryResult.Card.State)
			}

			if submitReview.ExpectedFSRSState != gofsrs.Learning {
				t.Errorf("Model expected Learning state (1), got: %v", submitReview.ExpectedFSRSState)
			}

			if reviewResp.Card.FSRS.State != gofsrs.Learning {
				t.Errorf("Service implementation returned %v, expected Learning state (1)",
					reviewResp.Card.FSRS.State)
			}

			// Verify due dates are reasonable
			libraryDue := libraryResult.Card.Due
			modelDue := submitReview.ExpectedDueDate
			serviceDue := reviewResp.Card.FSRS.Due

			t.Logf("Due date comparison:")
			t.Logf("Library due: %v (in %v)", libraryDue, libraryDue.Sub(now))
			t.Logf("Model due: %v (in %v)", modelDue, modelDue.Sub(now))
			t.Logf("Service due: %v (in %v)", serviceDue, serviceDue.Sub(now))

			// Verify due dates are all within a reasonable range of each other
			libraryToModelDiff := libraryDue.Sub(modelDue).Abs()
			libraryToServiceDiff := libraryDue.Sub(serviceDue).Abs()

			t.Logf("Due date differences:")
			t.Logf("Library vs Model: %v", libraryToModelDiff)
			t.Logf("Library vs Service: %v", libraryToServiceDiff)

			if libraryToModelDiff > 5*time.Second {
				t.Errorf("Model due date differs from library by %v", libraryToModelDiff)
			}

			if libraryToServiceDiff > 5*time.Second {
				t.Errorf("Service due date differs from library by %v", libraryToServiceDiff)
			}
		})
	}
}
