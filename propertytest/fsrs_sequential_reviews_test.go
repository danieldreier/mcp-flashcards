package propertytest

import (
	"fmt"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestFSRSSequentialReviews tests multiple reviews on the same card
// to verify state transitions and due date calculations over time.
func TestFSRSSequentialReviews(t *testing.T) {
	mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 300) // Longer timeout for sequential reviews
	if err != nil {
		t.Fatalf("Failed to setup client: %v", err)
	}
	defer clientCleanup()

	sut := FlashcardSUTFactory(mcpClient, ctx, nil, nil, t)

	// Create a card
	createCard := &CreateCardCmd{
		Front: "Sequential Review Front",
		Back:  "Sequential Review Back",
		Tags:  []string{"sequential-test"},
	}
	createResult := createCard.Run(sut)
	createResp, ok := createResult.(CreateCardResponse)
	if !ok {
		t.Fatalf("Failed to create card: %v", createResult)
	}
	cardID := createResp.Card.ID
	t.Logf("Created card %s with initial state: %v", cardID, createResp.Card.FSRS.State)

	// Review sequence
	reviews := []gofsrs.Rating{gofsrs.Good, gofsrs.Again, gofsrs.Good, gofsrs.Good, gofsrs.Easy, gofsrs.Hard}
	now := time.Now() // Base time for reviews

	// Variables to track expected FSRS state using direct library calls
	libraryCard := createResp.Card.FSRS // Start with the card's initial FSRS state
	libraryParams := gofsrs.DefaultParam()

	for i, rating := range reviews {
		// Run review in the service with the current 'now' time
		submitReview := &SubmitReviewCmd{
			CardID:    cardID,
			Rating:    rating,
			Answer:    fmt.Sprintf("Answer for review %d (rating %d)", i+1, rating),
			Timestamp: now, // Pass the current time explicitly
		}
		result := submitReview.Run(sut)
		reviewResp, ok := result.(ReviewResponse)
		if !ok {
			// Handle potential errors
			err, isErr := result.(error)
			if isErr {
				t.Fatalf("Review %d (Rating %d) failed: %v", i+1, rating, err)
			} else {
				t.Fatalf("Review %d (Rating %d) returned unexpected type: %T", i+1, rating, result)
			}
		}
		if !reviewResp.Success {
			t.Fatalf("Review %d (Rating %d) was not successful: %s", i+1, rating, reviewResp.Message)
		}

		// Calculate expected result using direct library call with the *current* libraryCard state
		schedule := libraryParams.Repeat(libraryCard, now)
		expectedLibraryCard := schedule[rating].Card

		// Compare service result with direct library calculation
		serviceCardFSRS := reviewResp.Card.FSRS
		t.Logf("--- Review %d (Rating: %d) --- NOW=%v", i+1, rating, now.Format(time.RFC3339))
		t.Logf("Library -> State: %v, Due: %v (in %v)", expectedLibraryCard.State, expectedLibraryCard.Due.Format(time.RFC3339), expectedLibraryCard.Due.Sub(now))
		t.Logf("Service -> State: %v, Due: %v (in %v)", serviceCardFSRS.State, serviceCardFSRS.Due.Format(time.RFC3339), serviceCardFSRS.Due.Sub(now))

		// Compare states
		if serviceCardFSRS.State != expectedLibraryCard.State {
			t.Errorf("Review %d (Rating %d): State mismatch. Expected(Lib) %v, Got(Svc) %v",
				i+1, rating, expectedLibraryCard.State, serviceCardFSRS.State)
		}

		// Compare due dates (allow small tolerance)
		if serviceCardFSRS.Due.Sub(expectedLibraryCard.Due).Abs() > 5*time.Second {
			t.Errorf("Review %d (Rating %d): Due date mismatch. Expected(Lib) %v, Got(Svc) %v",
				i+1, rating, expectedLibraryCard.Due.Format(time.RFC3339), serviceCardFSRS.Due.Format(time.RFC3339))
		}

		// Update library card state for the next iteration
		libraryCard = expectedLibraryCard

		// Advance time for the next review based on library prediction
		if libraryCard.ScheduledDays > 0 {
			now = now.AddDate(0, 0, int(libraryCard.ScheduledDays))
		} else {
			// If no days scheduled, simulate passing a short time (e.g., minutes)
			now = now.Add(10 * time.Minute)
		}
	}
}
