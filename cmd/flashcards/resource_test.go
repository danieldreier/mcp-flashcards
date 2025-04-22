package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDueDateProgressResource is a unit test for the due-date-progress resource.
func TestDueDateProgressResource(t *testing.T) {
	// Setup temp file and service
	filePath := filepath.Join(t.TempDir(), "test-due-date-progress.json")
	fileStorage := storage.NewFileStorage(filePath)
	defer os.Remove(filePath)

	err := fileStorage.Load()
	require.NoError(t, err, "Failed to initialize storage")

	service := NewFlashcardService(fileStorage)

	// Create a test due date
	dueDate := storage.DueDate{
		ID:      "test-due-date-id",
		Topic:   "Test Exam",
		DueDate: time.Now().AddDate(0, 0, 10), // 10 days from now
		Tag:     "test-exam-tag",
	}

	err = service.AddDueDate(dueDate)
	require.NoError(t, err, "Failed to add due date")

	// Create some cards with the test tag
	for i := 0; i < 3; i++ {
		_, err := service.CreateCard(
			"Question "+string(rune('A'+i)),
			"Answer "+string(rune('A'+i)),
			[]string{"test-exam-tag"},
		)
		require.NoError(t, err, "Failed to create card")
	}

	// Create a card with a different tag
	_, err = service.CreateCard("Other Question", "Other Answer", []string{"other-tag"})
	require.NoError(t, err, "Failed to create card with different tag")

	// Review one card with Easy rating
	cards, _, err := service.ListCards([]string{"test-exam-tag"}, false)
	require.NoError(t, err, "Failed to list cards")
	require.NotEmpty(t, cards, "No cards found with test tag")

	_, err = service.SubmitReview(cards[0].ID, gofsrs.Easy, "Test answer")
	require.NoError(t, err, "Failed to submit review")

	// Now create and execute the resource request
	ctx := context.WithValue(context.Background(), "service", service)
	request := mcp.ReadResourceRequest{}

	contents, err := handleDueDateProgressResource(ctx, request)
	require.NoError(t, err, "handleDueDateProgressResource returned an error")
	require.NotNil(t, contents, "Resource contents should not be nil")
	require.Len(t, contents, 1, "Expected 1 resource content")

	// Get the text content
	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok, "Resource content should be TextResourceContents")
	require.NotEmpty(t, textContent.Text, "Resource text should not be empty")

	t.Logf("Resource content: %s", textContent.Text)

	// Unmarshal and check the content
	var progressInfos []DueDateProgressInfo
	err = json.Unmarshal([]byte(textContent.Text), &progressInfos)
	require.NoError(t, err, "Failed to unmarshal resource text")

	// We should have one progress info for our test due date
	require.Len(t, progressInfos, 1, "Expected 1 progress info")

	progressInfo := progressInfos[0]
	assert.Equal(t, dueDate.ID, progressInfo.ID, "Progress info ID should match due date ID")
	assert.Equal(t, dueDate.Topic, progressInfo.Topic, "Progress info topic should match due date topic")
	assert.Equal(t, dueDate.Tag, progressInfo.Tag, "Progress info tag should match due date tag")
	assert.Equal(t, 3, progressInfo.TotalCards, "Should have 3 total cards with the due date tag")
	assert.Equal(t, 1, progressInfo.MasteredCards, "Should have 1 mastered card")
	assert.InDelta(t, 33.33, progressInfo.ProgressPercent, 0.01, "Progress percentage should be about 33.33%")
	assert.Equal(t, 2, progressInfo.CardsLeft, "Should have 2 cards left to master")
	assert.Greater(t, progressInfo.DaysRemaining, 0.0, "Days remaining should be positive")
	assert.Greater(t, progressInfo.RequiredPace, 0.0, "Required pace should be positive")
}

// TestEmptyDueDateProgressResource tests the resource when there are no due dates.
func TestEmptyDueDateProgressResource(t *testing.T) {
	// Setup temp file and service
	filePath := filepath.Join(t.TempDir(), "test-empty-due-date-progress.json")
	fileStorage := storage.NewFileStorage(filePath)
	defer os.Remove(filePath)

	err := fileStorage.Load()
	require.NoError(t, err, "Failed to initialize storage")

	service := NewFlashcardService(fileStorage)

	// Execute the resource request with no due dates
	ctx := context.WithValue(context.Background(), "service", service)
	request := mcp.ReadResourceRequest{}

	contents, err := handleDueDateProgressResource(ctx, request)
	require.NoError(t, err, "handleDueDateProgressResource returned an error")
	require.NotNil(t, contents, "Resource contents should not be nil")
	require.Len(t, contents, 1, "Expected 1 resource content")

	// Get the text content
	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok, "Resource content should be TextResourceContents")

	t.Logf("Empty resource content: %s", textContent.Text)

	// Unmarshal and check the content - should be an empty array
	var progressInfos []DueDateProgressInfo
	err = json.Unmarshal([]byte(textContent.Text), &progressInfos)
	require.NoError(t, err, "Failed to unmarshal resource text")

	// Should be an empty array, not null
	assert.NotNil(t, progressInfos, "Progress infos should not be nil")
	assert.Empty(t, progressInfos, "Progress infos should be empty")
}

// TestPastDueDatesResource tests that past due dates are not included in the resource.
func TestPastDueDatesResource(t *testing.T) {
	// Setup temp file and service
	filePath := filepath.Join(t.TempDir(), "test-past-due-dates.json")
	fileStorage := storage.NewFileStorage(filePath)
	defer os.Remove(filePath)

	err := fileStorage.Load()
	require.NoError(t, err, "Failed to initialize storage")

	service := NewFlashcardService(fileStorage)

	// Create a past due date
	pastDueDate := storage.DueDate{
		ID:      "past-due-date-id",
		Topic:   "Past Exam",
		DueDate: time.Now().AddDate(0, 0, -1), // 1 day ago
		Tag:     "past-exam-tag",
	}

	err = service.AddDueDate(pastDueDate)
	require.NoError(t, err, "Failed to add past due date")

	// Create a future due date
	futureDueDate := storage.DueDate{
		ID:      "future-due-date-id",
		Topic:   "Future Exam",
		DueDate: time.Now().AddDate(0, 0, 10), // 10 days from now
		Tag:     "future-exam-tag",
	}

	err = service.AddDueDate(futureDueDate)
	require.NoError(t, err, "Failed to add future due date")

	// Execute the resource request
	ctx := context.WithValue(context.Background(), "service", service)
	request := mcp.ReadResourceRequest{}

	contents, err := handleDueDateProgressResource(ctx, request)
	require.NoError(t, err, "handleDueDateProgressResource returned an error")

	// Get the text content
	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok, "Resource content should be TextResourceContents")

	t.Logf("Past/Future due dates resource content: %s", textContent.Text)

	// Unmarshal and check the content
	var progressInfos []DueDateProgressInfo
	err = json.Unmarshal([]byte(textContent.Text), &progressInfos)
	require.NoError(t, err, "Failed to unmarshal resource text")

	// Should only include the future due date
	require.Len(t, progressInfos, 1, "Expected 1 progress info (only future due date)")
	assert.Equal(t, futureDueDate.ID, progressInfos[0].ID, "Progress info should be for the future due date")
}
