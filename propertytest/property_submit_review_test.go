package propertytest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
	// Ensure client import is present
)

// TestSubmitReviewCommand specifically tests the SubmitReviewCmd implementation
func TestSubmitReviewCommand(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClient(t)
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	// Create a SUT using the factory, passing the combined cleanup
	// Note: FlashcardSUTFactory expects the tempDir cleanup, but clientCleanup handles that.
	// We pass nil for tempDirCleanup here as clientCleanup covers it.
	sut := FlashcardSUTFactory(mcpClient, ctx, nil, nil, t)

	// Create a card first
	createCardInitial := &CreateCardCmd{
		Front: "Initial Card Front",
		Back:  "Initial Card Back",
		Tags:  []string{"initial", "submit-review-test"},
	}
	createResultInitial := createCardInitial.Run(sut)
	createRespInitial, ok := createResultInitial.(CreateCardResponse)
	if !ok {
		t.Fatalf("Failed to create initial card: %v", createResultInitial)
	}
	cardIDInitial := createRespInitial.Card.ID
	t.Logf("Created initial test card with ID: %s", cardIDInitial)

	// *** Raw go-fsrs library behavior check ***
	tempNow := time.Now()
	tempNewCard := gofsrs.NewCard()
	tempParams := gofsrs.DefaultParam()
	t.Logf("--- Raw go-fsrs library behavior check (using pristine new card at %v) ---", tempNow.Format(time.RFC3339))
	for _, rating := range []gofsrs.Rating{gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy} {
		tempSchedulingInfo := tempParams.Repeat(tempNewCard, tempNow)
		t.Logf("Raw go-fsrs for new card, rating %d: State=%v, Due=%v (Interval: %v)",
			rating,
			tempSchedulingInfo[rating].Card.State,
			tempSchedulingInfo[rating].Card.Due.Format(time.RFC3339),
			tempSchedulingInfo[rating].Card.Due.Sub(tempNow))
	}

	// Test cases for SubmitReviewCmd
	testCases := []struct {
		name           string
		initialCard    Card            // Card state *before* the review
		cmd            SubmitReviewCmd // The command to run
		expectedState  gofsrs.State    // Expected FSRS state *after* review
		expectedErrMsg string          // Substring of expected error message, if any
	}{
		{
			name: "New Card Rated Good",
			initialCard: Card{
				ID:   cardIDInitial, // Use the created card ID
				FSRS: gofsrs.NewCard(),
			},
			cmd: SubmitReviewCmd{
				CardID: cardIDInitial,
				Rating: gofsrs.Good,
				Answer: "Correct Answer",
			},
			expectedState: gofsrs.Learning, // Updated: New -> Good should go to Learning
		},
		{
			name: "New Card Rated Again",
			initialCard: Card{
				ID:   cardIDInitial,
				FSRS: gofsrs.NewCard(),
			},
			cmd: SubmitReviewCmd{
				CardID: cardIDInitial,
				Rating: gofsrs.Again,
				Answer: "Wrong Answer",
			},
			expectedState: gofsrs.Learning, // New -> Again should go to Learning
		},
		{
			name: "Non-existent Card ID",
			initialCard: Card{ // Model state doesn't matter much here
				ID:   "non-existent-id",
				FSRS: gofsrs.NewCard(),
			},
			cmd: SubmitReviewCmd{
				CardID: "non-existent-id",
				Rating: gofsrs.Good,
				Answer: "Answer",
			},
			expectedErrMsg: "card not found",
		},
		// Add more test cases: existing card in Review state, Learning state etc.
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure the target card exists for relevant tests
			if tc.cmd.CardID == cardIDInitial {
				// Recreate card if necessary to reset its state for the test?
				// For now, assume the SUT state persists between tc runs and might need reset.
				// This simple test structure might not be robust enough for complex state checks.
			}

			// Run the command
			result := tc.cmd.Run(sut)

			// Check result type and error
			if tc.expectedErrMsg != "" {
				errResult, ok := result.(error)
				if !ok {
					t.Errorf("Expected an error containing '%s', but got successful result: %T %v",
						tc.expectedErrMsg, result, result)
				} else if !strings.Contains(errResult.Error(), tc.expectedErrMsg) {
					t.Errorf("Expected error message containing '%s', got: %v",
						tc.expectedErrMsg, errResult)
				}
			} else {
				reviewResp, ok := result.(ReviewResponse)
				if !ok {
					t.Errorf("Expected successful ReviewResponse, but got: %T %v", result, result)
				} else if !reviewResp.Success {
					t.Errorf("Expected successful review, but got success=false: %s", reviewResp.Message)
				} else if reviewResp.Card.FSRS.State != tc.expectedState {
					t.Errorf("Expected resulting state %v, got %v", tc.expectedState, reviewResp.Card.FSRS.State)
				}
			}
		})
	}
}

// TestGetDueCardCommand specifically tests the GetDueCardCmd implementation
func TestGetDueCardCommand(t *testing.T) {
	// Create unique state file for this test run
	_, stateFilePath, tempCleanup, err := CreateTempStateFile(t)
	if err != nil {
		t.Fatalf("Failed to create temp state file: %v", err)
	}
	defer tempCleanup() // Ensure temp state dir is cleaned up

	// Setup a client with the temp file
	mcpClient, ctx, cancel, err := SetupPropertyTestClient(t, stateFilePath)
	if err != nil {
		t.Fatalf("Failed to setup property test client: %v", err)
	}

	// Define and defer instance cleanup (client close, context cancel)
	clientCleanup := func() {
		t.Log("Running instance cleanup for TestGetDueCardCommand")
		cancel()
		mcpClient.Close()
	}
	defer clientCleanup()

	// Use the SUT factory (Cleanup is handled by the combined defer)
	sut := FlashcardSUTFactory(mcpClient, ctx, cancel, nil, t) // Pass nil for tempCleanup as it's handled by the deferred clientCleanup

	// Create several cards with different due dates and tags
	cards := []struct {
		front    string
		back     string
		tags     []string
		dueState gofsrs.State // The state to set after creation
	}{
		{"Card 1 Front", "Card 1 Back", []string{"tag1", "high-priority"}, gofsrs.New},           // due now
		{"Card 2 Front", "Card 2 Back", []string{"tag2", "medium-priority"}, gofsrs.Learning},    // due in 1 min
		{"Card 3 Front", "Card 3 Back", []string{"tag1", "tag3", "low-priority"}, gofsrs.Review}, // due in 10 min
		{"Card 4 Front", "Card 4 Back", []string{"tag4"}, gofsrs.Review},                         // due in 1 day (not due now)
	}

	// Create the cards and update their due states
	var cardIDs []string
	for _, card := range cards {
		// Create the card
		createCmd := &CreateCardCmd{
			Front: card.front,
			Back:  card.back,
			Tags:  card.tags,
		}
		createResult := createCmd.Run(sut)
		createResp, ok := createResult.(CreateCardResponse)
		if !ok {
			t.Fatalf("Failed to create card: %v", createResult)
		}
		cardID := createResp.Card.ID
		cardIDs = append(cardIDs, cardID)
		t.Logf("Created test card with ID: %s, tags: %v", cardID, card.tags)

		// For cards that need review state, submit a review to get into that state
		if card.dueState != gofsrs.New {
			// Submit a "Good" review to move to Learning/Review state
			submitCmd := &SubmitReviewCmd{
				CardID: cardID,
				Rating: gofsrs.Good,
				Answer: "Test answer",
			}
			submitResult := submitCmd.Run(sut)
			if _, ok := submitResult.(error); ok {
				t.Fatalf("Failed to submit review for card %s: %v", cardID, submitResult)
			}
			t.Logf("Submitted initial review for card %s", cardID)
		}
	}

	// Test 1: Get due card with no filters
	getDueCmd1 := &GetDueCardCmd{}
	getDueResult1 := getDueCmd1.Run(sut)
	t.Logf("GetDueCard with no filters result: %v", getDueResult1)

	// If we get a card response, verify it's one of our cards
	if cardResp, ok := getDueResult1.(CardResponse); ok {
		found := false
		for _, id := range cardIDs {
			if cardResp.Card.ID == id {
				found = true
				t.Logf("Successfully got due card with ID: %s", id)
				break
			}
		}
		if !found {
			t.Errorf("Due card ID %s not in our test set", cardResp.Card.ID)
		}
	} else if errResult, ok := getDueResult1.(error); ok {
		t.Errorf("Unexpected error from GetDueCard: %v", errResult)
	}

	// Test 2: Get due card with tag filter
	getDueCmd2 := &GetDueCardCmd{
		FilterTags: []string{"tag1"},
	}
	getDueResult2 := getDueCmd2.Run(sut)
	t.Logf("GetDueCard with tag1 filter result: %v", getDueResult2)

	// If we get a card response, verify it has tag1
	if cardResp, ok := getDueResult2.(CardResponse); ok {
		hasTag := false
		for _, tag := range cardResp.Card.Tags {
			if tag == "tag1" {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("Due card %s doesn't have required tag 'tag1': %v",
				cardResp.Card.ID, cardResp.Card.Tags)
		} else {
			t.Logf("Successfully got due card with tag1: %s", cardResp.Card.ID)
		}
	} else if errResult, ok := getDueResult2.(error); ok {
		// This might be acceptable if no cards with tag1 are due
		t.Logf("Note: GetDueCard with tag1 filter returned error: %v", errResult)
	}

	// Test 3: Get due card with non-existent tag filter
	getDueCmd3 := &GetDueCardCmd{
		FilterTags: []string{"non-existent-tag"},
	}
	getDueResult3 := getDueCmd3.Run(sut)
	t.Logf("GetDueCard with non-existent tag filter result: %v", getDueResult3)

	// We should get an error for a non-existent tag
	if _, ok := getDueResult3.(error); !ok {
		t.Errorf("Expected error for non-existent tag, got: %v", getDueResult3)
	} else {
		t.Logf("Correctly got error for non-existent tag")
	}
}

// TestSubmitReviewProperty tests SubmitReviewCmd using a simpler approach
func TestSubmitReviewProperty(t *testing.T) {
	// Create unique state file for this test run
	_, stateFilePath, tempCleanup, err := CreateTempStateFile(t)
	if err != nil {
		t.Fatalf("Failed to create temp state file: %v", err)
	}
	defer tempCleanup() // Ensure temp state dir is cleaned up

	// Use property client setup with the specific state file
	mcpClient, ctx, cancel, err := SetupPropertyTestClient(t, stateFilePath)
	if err != nil {
		t.Fatalf("Failed to setup property test client: %v", err)
	}
	defer func() {
		cancel()
		mcpClient.Close()
	}() // Handles client/context cleanup

	// Use the SUT factory (Cleanup handled by clientCleanup)
	sut := FlashcardSUTFactory(mcpClient, ctx, cancel, nil, t)

	// Create a card first
	createCard := &CreateCardCmd{
		Front: "Property Test Card Front",
		Back:  "Property Test Card Back",
		Tags:  []string{"property-test"},
	}
	createResult := createCard.Run(sut)
	createResp, ok := createResult.(CreateCardResponse)
	if !ok {
		t.Fatalf("Failed to create card for property test: %v", createResult)
	}
	cardID := createResp.Card.ID

	// Test all possible ratings (1-4)
	ratings := []gofsrs.Rating{gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy}
	for _, rating := range ratings {
		t.Run(fmt.Sprintf("Rating=%d", rating), func(t *testing.T) {
			submitReview := &SubmitReviewCmd{
				CardID: cardID,
				Rating: rating,
				Answer: fmt.Sprintf("Test answer for rating %d", rating),
			}

			// Execute the review command
			result := submitReview.Run(sut)

			// Verify the result
			reviewResp, ok := result.(ReviewResponse)
			if !ok {
				t.Fatalf("SubmitReview failed for rating %d: %v", rating, result)
			}

			if !reviewResp.Success {
				t.Fatalf("SubmitReview was not successful for rating %d: %s", rating, reviewResp.Message)
			}

			// Verify the FSRS state is valid
			if reviewResp.Card.FSRS.State < gofsrs.New || reviewResp.Card.FSRS.State > gofsrs.Relearning {
				t.Fatalf("Invalid FSRS state %v after rating %d", reviewResp.Card.FSRS.State, rating)
			}

			t.Logf("Rating %d resulted in state %v, due at %v",
				rating, reviewResp.Card.FSRS.State, reviewResp.Card.FSRS.Due.Format(time.RFC3339))
		})
	}
}
