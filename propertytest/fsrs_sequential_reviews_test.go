package propertytest

import (
	"fmt"
	"testing"
	"time"

	gofsrs "github.com/open-spaced-repetition/go-fsrs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestFSRSSequentialReviews tests multiple reviews on the same card
// to verify state transitions and due date calculations over time.
func TestFSRSSequentialReviews(t *testing.T) {
	// Initialize Zap logger for this test
	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logConfig.EncoderConfig.CallerKey = ""
	logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := logConfig.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		t.Fatalf("Failed to create zap logger: %v", err)
	}
	defer logger.Sync()

	mcpClient, ctx, _, clientCleanup, err := SetupTestClientWithLongTimeout(t, 300) // Longer timeout for sequential reviews
	if err != nil {
		// Use zap logger for fatal errors
		logger.Fatal("Failed to setup client", zap.Error(err))
	}
	defer clientCleanup()

	// Pass the logger to the SUT factory
	// Note: The factory function in common_test.go needs to be updated
	// if it doesn't already accept and use a passed-in logger.
	// Assuming for now common_test.go's factory handles this or we modify it later.
	sut := FlashcardSUTFactory(mcpClient, ctx, nil, nil, t)
	sut.Logger = logger // Explicitly assign logger if factory doesn't handle it

	// Create a card
	createCard := &CreateCardCmd{
		Front: "Sequential Review Front",
		Back:  "Sequential Review Back",
		Tags:  []string{"sequential-test"},
	}
	createResult := createCard.Run(sut)
	createResp, ok := createResult.(CreateCardResponse)
	if !ok {
		logger.Fatal("Failed to create card", zap.Any("result", createResult))
	}
	cardID := createResp.Card.ID
	logger.Debug("Created card", zap.String("card_id", cardID), zap.Any("initial_state", createResp.Card.FSRS.State))

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
		logger.Debug("Review comparison",
			zap.Int("review_num", i+1),
			zap.Int("rating", int(rating)),
			zap.Time("review_time", now),
			zap.Any("library_state", expectedLibraryCard.State),
			zap.Time("library_due", expectedLibraryCard.Due),
			zap.Duration("library_due_in", expectedLibraryCard.Due.Sub(now)),
			zap.Any("service_state", serviceCardFSRS.State),
			zap.Time("service_due", serviceCardFSRS.Due),
			zap.Duration("service_due_in", serviceCardFSRS.Due.Sub(now)))

		// Compare states
		if serviceCardFSRS.State != expectedLibraryCard.State {
			// Use logger for errors
			logger.Error("State mismatch",
				zap.Int("review_num", i+1),
				zap.Int("rating", int(rating)),
				zap.Any("expected_state", expectedLibraryCard.State),
				zap.Any("actual_state", serviceCardFSRS.State))
			t.Errorf("Review %d (Rating %d): State mismatch. Expected(Lib) %v, Got(Svc) %v",
				i+1, rating, expectedLibraryCard.State, serviceCardFSRS.State)
		}

		// Compare due dates (allow small tolerance)
		if serviceCardFSRS.Due.Sub(expectedLibraryCard.Due).Abs() > 5*time.Second {
			logger.Error("Due date mismatch",
				zap.Int("review_num", i+1),
				zap.Int("rating", int(rating)),
				zap.Time("expected_due", expectedLibraryCard.Due),
				zap.Time("actual_due", serviceCardFSRS.Due))
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
