package propertytest

import (
	"fmt"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSModelComparison directly compares the FSRS model predictions from the property test
// with the actual implementation in the service.
func TestFSRSModelComparison(t *testing.T) {
	// Compare raw library behavior vs service behavior to identify any deviations

	// Setup a client for the service
	mcpClient, ctx, cancel, baseCleanup := SetupPropertyTestClient(t)
	defer func() {
		cancel()
		mcpClient.Close()
		baseCleanup()
	}()

	// Create a SUT for the service
	sut := &FlashcardSUT{
		Client:      mcpClient,
		Ctx:         ctx,
		Cancel:      cancel,
		CleanupFunc: baseCleanup,
		T:           t,
	}

	// Create a direct library instance for comparison
	directParams := gofsrs.DefaultParam()

	// Test cases combining different initial states and ratings
	testCases := []struct {
		name         string
		initialState gofsrs.State
		lastReview   time.Duration // Time since last review (relative to now)
		rating       gofsrs.Rating
		description  string
	}{
		{
			name:         "New_Again",
			initialState: gofsrs.New,
			lastReview:   0, // New card has no last review
			rating:       gofsrs.Again,
			description:  "New card rated Again",
		},
		{
			name:         "New_Hard",
			initialState: gofsrs.New,
			lastReview:   0,
			rating:       gofsrs.Hard,
			description:  "New card rated Hard",
		},
		{
			name:         "New_Good",
			initialState: gofsrs.New,
			lastReview:   0,
			rating:       gofsrs.Good,
			description:  "New card rated Good",
		},
		{
			name:         "New_Easy",
			initialState: gofsrs.New,
			lastReview:   0,
			rating:       gofsrs.Easy,
			description:  "New card rated Easy",
		},
		{
			name:         "Learning_1h_Again",
			initialState: gofsrs.Learning,
			lastReview:   -1 * time.Hour,
			rating:       gofsrs.Again,
			description:  "Learning card (1h old) rated Again",
		},
		{
			name:         "Learning_1h_Good",
			initialState: gofsrs.Learning,
			lastReview:   -1 * time.Hour,
			rating:       gofsrs.Good,
			description:  "Learning card (1h old) rated Good",
		},
		{
			name:         "Review_1d_Again",
			initialState: gofsrs.Review,
			lastReview:   -24 * time.Hour,
			rating:       gofsrs.Again,
			description:  "Review card (1 day old) rated Again",
		},
		{
			name:         "Review_1d_Good",
			initialState: gofsrs.Review,
			lastReview:   -24 * time.Hour,
			rating:       gofsrs.Good,
			description:  "Review card (1 day old) rated Good",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			lastReviewTime := now.Add(tc.lastReview)

			// 1. Calculate expected result directly using the library
			directCard := gofsrs.NewCard()
			directCard.State = tc.initialState
			if tc.lastReview != 0 {
				directCard.LastReview = lastReviewTime
			}

			// Print initial card state for direct library test
			t.Logf("Library test setup: State=%v, LastReview=%v",
				directCard.State, directCard.LastReview)

			// Get direct library prediction
			directSchedulingInfo := directParams.Repeat(directCard, now)
			directNextState := directSchedulingInfo[tc.rating].Card.State
			directDueDate := directSchedulingInfo[tc.rating].Card.Due

			// Print direct library results
			t.Logf("Direct library result: NextState=%v, DueDate=%v, DueInterval=%v",
				directNextState, directDueDate, directDueDate.Sub(now))

			// 2. Create a card in the service with the same initial state
			// First create a new card (always starts in New state)
			createCard := &CreateCardCmd{
				Front: fmt.Sprintf("Test Card Front - %s", tc.name),
				Back:  fmt.Sprintf("Test Card Back - %s", tc.name),
				Tags:  []string{"test", "model-comparison", tc.name},
			}
			createResult := createCard.Run(sut)
			createResp, ok := createResult.(CreateCardResponse)
			if !ok {
				t.Fatalf("Failed to create card: %v", createResult)
			}

			cardID := createResp.Card.ID
			t.Logf("Created card with ID: %s, Initial State: %v",
				cardID, createResp.Card.FSRS.State)

			// If we need a state other than New, we need to perform preliminary reviews
			// to get the card into the desired state
			if tc.initialState != gofsrs.New {
				t.Logf("Need to transition card from New to %v state", tc.initialState)

				// Transition strategy depends on target state
				switch tc.initialState {
				case gofsrs.Learning:
					// New -> Again = Learning
					initialReview := &SubmitReviewCmd{
						CardID: cardID,
						Rating: gofsrs.Again,
						Answer: "Initial review to reach Learning state",
					}
					result := initialReview.Run(sut)
					reviewResp, ok := result.(ReviewResponse)
					if !ok || !reviewResp.Success {
						t.Fatalf("Failed initial review: %v", result)
					}
					t.Logf("Transitioned to Learning state: %v", reviewResp.Card.FSRS.State)

				case gofsrs.Review:
					// New -> Easy = Review
					initialReview := &SubmitReviewCmd{
						CardID: cardID,
						Rating: gofsrs.Easy,
						Answer: "Initial review to reach Review state",
					}
					result := initialReview.Run(sut)
					reviewResp, ok := result.(ReviewResponse)
					if !ok || !reviewResp.Success {
						t.Fatalf("Failed initial review: %v", result)
					}
					t.Logf("Transitioned to Review state: %v", reviewResp.Card.FSRS.State)

				case gofsrs.Relearning:
					// New -> Easy -> Again = Relearning
					firstReview := &SubmitReviewCmd{
						CardID: cardID,
						Rating: gofsrs.Easy,
						Answer: "First review to reach Review state",
					}
					result1 := firstReview.Run(sut)
					reviewResp1, ok := result1.(ReviewResponse)
					if !ok || !reviewResp1.Success {
						t.Fatalf("Failed first review: %v", result1)
					}

					secondReview := &SubmitReviewCmd{
						CardID: cardID,
						Rating: gofsrs.Again,
						Answer: "Second review to reach Relearning state",
					}
					result2 := secondReview.Run(sut)
					reviewResp2, ok := result2.(ReviewResponse)
					if !ok || !reviewResp2.Success {
						t.Fatalf("Failed second review: %v", result2)
					}
					t.Logf("Transitioned to Relearning state: %v", reviewResp2.Card.FSRS.State)
				}
			}

			// Get the current card state
			getCardCmd := &GetCardCmd{CardID: cardID}
			getResult := getCardCmd.Run(sut)
			cards, ok := getResult.(ListCardsResponse)
			if !ok {
				t.Fatalf("Failed to get card: %v", getResult)
			}

			var currentCard Card
			for _, card := range cards.Cards {
				if card.ID == cardID {
					currentCard = card
					break
				}
			}

			if currentCard.ID == "" {
				t.Fatalf("Card %s not found in list response", cardID)
			}

			// Verify the card is in the expected initial state
			t.Logf("Card state before test review: %v", currentCard.FSRS.State)
			if currentCard.FSRS.State != tc.initialState {
				t.Fatalf("Card is not in the expected initial state: got %v, want %v",
					currentCard.FSRS.State, tc.initialState)
			}

			// 3. Run the review in the test model to get the expected result
			modelCard := currentCard // Copy current state for the model

			// Create mock state for NextState to populate expectations
			mockState := &CommandState{
				Cards: map[string]Card{
					cardID: modelCard,
				},
				KnownRealIDs: []string{cardID},
				LastRealID:   cardID,
				T:            t,
			}

			// Create the review command
			submitReview := &SubmitReviewCmd{
				CardID: cardID,
				Rating: tc.rating,
				Answer: fmt.Sprintf("Test review with rating %d", tc.rating),
			}

			// Call NextState to populate the expected values
			submitReview.NextState(mockState)

			// Print model predictions
			t.Logf("Model predicts: NextState=%v, DueDate=%v",
				submitReview.ExpectedFSRSState, submitReview.ExpectedDueDate)

			// 4. Run the review in the service
			result := submitReview.Run(sut)

			// Verify the result
			reviewResp, ok := result.(ReviewResponse)
			if !ok {
				t.Fatalf("Failed to submit review: %v", result)
			}

			if !reviewResp.Success {
				t.Fatalf("Review was not successful: %s", reviewResp.Message)
			}

			// 5. Compare both results
			t.Logf("Service result: NextState=%v, DueDate=%v",
				reviewResp.Card.FSRS.State, reviewResp.Card.FSRS.Due)

			// Check if direct library, model, and service all agree
			modelMatchesLibrary := submitReview.ExpectedFSRSState == directNextState
			serviceMatchesLibrary := reviewResp.Card.FSRS.State == directNextState
			serviceMatchesModel := reviewResp.Card.FSRS.State == submitReview.ExpectedFSRSState

			t.Logf("Comparison - Model matches Library: %v, Service matches Library: %v, Service matches Model: %v",
				modelMatchesLibrary, serviceMatchesLibrary, serviceMatchesModel)

			if !modelMatchesLibrary {
				t.Errorf("Model prediction (%v) doesn't match direct library call (%v)",
					submitReview.ExpectedFSRSState, directNextState)
			}

			if !serviceMatchesLibrary {
				t.Errorf("Service implementation (%v) doesn't match direct library call (%v)",
					reviewResp.Card.FSRS.State, directNextState)
			}

			if !serviceMatchesModel {
				t.Errorf("Service implementation (%v) doesn't match model prediction (%v)",
					reviewResp.Card.FSRS.State, submitReview.ExpectedFSRSState)
			}

			// Check due dates
			modelDueDiff := submitReview.ExpectedDueDate.Sub(directDueDate).Abs()
			serviceDueDiff := reviewResp.Card.FSRS.Due.Sub(directDueDate).Abs()

			t.Logf("Due date differences - Model vs Library: %v, Service vs Library: %v",
				modelDueDiff, serviceDueDiff)

			if modelDueDiff > 5*time.Second {
				t.Errorf("Model due date differs from library by %v", modelDueDiff)
			}

			if serviceDueDiff > 5*time.Second {
				t.Errorf("Service due date differs from library by %v", serviceDueDiff)
			}
		})
	}
}
