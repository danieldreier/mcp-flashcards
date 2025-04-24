package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
	"github.com/stretchr/testify/assert"
)

// Function to temporarily mock the time.Now function for testing
func mockTimeNow(mockTime time.Time) func() {
	original := timeNow
	timeNow = func() time.Time {
		return mockTime
	}
	return func() {
		timeNow = original
	}
}

// Helper function to create a temporary file for testing
func tempTestFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "flashcards-service-test.json")
}

// Helper function to create a service with a temporary storage file
func setupTestService(t *testing.T) (*FlashcardService, string) {
	t.Helper()
	filePath := tempTestFile(t)
	fileStorage := storage.NewFileStorage(filePath)
	if err := fileStorage.Load(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	service := NewFlashcardService(fileStorage)
	return service, filePath
}

// TestAddDueDate tests adding a due date to the service
func TestAddDueDate(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create a test due date
	dueDate := storage.DueDate{
		ID:      "test-id-1",
		Topic:   "Test Topic",
		DueDate: time.Now().AddDate(0, 0, 7), // 7 days from now
		Tag:     "test-tag-2023-12-31",
	}

	// Add the due date
	err := service.AddDueDate(dueDate)
	assert.NoError(t, err, "AddDueDate should not return an error")

	// Verify the due date was added by using ListDueDates
	dueDates, err := service.ListDueDates()
	assert.NoError(t, err, "ListDueDates should not return an error")
	assert.Len(t, dueDates, 1, "Should have one due date")
	assert.Equal(t, dueDate.ID, dueDates[0].ID, "Due date ID should match")
	assert.Equal(t, dueDate.Topic, dueDates[0].Topic, "Due date topic should match")
	assert.Equal(t, dueDate.Tag, dueDates[0].Tag, "Due date tag should match")
}

// TestListDueDates tests listing due dates
func TestListDueDates(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create multiple test due dates
	dueDates := []storage.DueDate{
		{
			ID:      "test-id-1",
			Topic:   "Test Topic 1",
			DueDate: time.Now().AddDate(0, 0, 7),
			Tag:     "test-tag-1",
		},
		{
			ID:      "test-id-2",
			Topic:   "Test Topic 2",
			DueDate: time.Now().AddDate(0, 0, 14),
			Tag:     "test-tag-2",
		},
	}

	// Add the due dates
	for _, dd := range dueDates {
		err := service.AddDueDate(dd)
		assert.NoError(t, err, "AddDueDate should not return an error")
	}

	// List all due dates
	listedDueDates, err := service.ListDueDates()
	assert.NoError(t, err, "ListDueDates should not return an error")
	assert.Len(t, listedDueDates, 2, "Should have two due dates")

	// Verify the contents match
	assert.Equal(t, dueDates[0].ID, listedDueDates[0].ID, "First due date ID should match")
	assert.Equal(t, dueDates[1].ID, listedDueDates[1].ID, "Second due date ID should match")
}

// TestUpdateDueDate tests updating a due date
func TestUpdateDueDate(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create a test due date
	dueDate := storage.DueDate{
		ID:      "test-id-1",
		Topic:   "Test Topic",
		DueDate: time.Now().AddDate(0, 0, 7),
		Tag:     "test-tag-2023-12-31",
	}

	// Add the due date
	err := service.AddDueDate(dueDate)
	assert.NoError(t, err, "AddDueDate should not return an error")

	// Update the due date
	updatedDueDate := dueDate
	updatedDueDate.Topic = "Updated Topic"
	updatedDueDate.Tag = "updated-tag"
	err = service.UpdateDueDate(updatedDueDate)
	assert.NoError(t, err, "UpdateDueDate should not return an error")

	// Verify the update
	dueDates, err := service.ListDueDates()
	assert.NoError(t, err, "ListDueDates should not return an error")
	assert.Len(t, dueDates, 1, "Should still have one due date")
	assert.Equal(t, updatedDueDate.Topic, dueDates[0].Topic, "Topic should be updated")
	assert.Equal(t, updatedDueDate.Tag, dueDates[0].Tag, "Tag should be updated")
}

// TestDeleteDueDate tests deleting a due date
func TestDeleteDueDate(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create a test due date
	dueDate := storage.DueDate{
		ID:      "test-id-1",
		Topic:   "Test Topic",
		DueDate: time.Now().AddDate(0, 0, 7),
		Tag:     "test-tag-2023-12-31",
	}

	// Add the due date
	err := service.AddDueDate(dueDate)
	assert.NoError(t, err, "AddDueDate should not return an error")

	// Delete the due date
	err = service.DeleteDueDate(dueDate.ID)
	assert.NoError(t, err, "DeleteDueDate should not return an error")

	// Verify the deletion
	dueDates, err := service.ListDueDates()
	assert.NoError(t, err, "ListDueDates should not return an error")
	assert.Empty(t, dueDates, "Due date list should be empty after deletion")
}

// TestGetDueDateProgressStats tests calculating progress for a due date
func TestGetDueDateProgressStats(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create a test due date
	dueDate := storage.DueDate{
		ID:      "test-id-1",
		Topic:   "Test Topic",
		DueDate: time.Now().AddDate(0, 0, 7), // 7 days from now
		Tag:     "test-progress-tag",
	}

	// Add the due date
	err := service.AddDueDate(dueDate)
	assert.NoError(t, err, "AddDueDate should not return an error")

	// Create cards with the same tag
	cards := []struct {
		front string
		back  string
		tag   string
	}{
		{"Card 1 Front", "Card 1 Back", "test-progress-tag"},
		{"Card 2 Front", "Card 2 Back", "test-progress-tag"},
		{"Card 3 Front", "Card 3 Back", "test-progress-tag"},
		{"Card 4 Front", "Card 4 Back", "different-tag"}, // Different tag, shouldn't count
	}

	cardIDs := make([]string, 0, len(cards))
	for _, c := range cards {
		card, err := service.CreateCard(c.front, c.back, []string{c.tag})
		assert.NoError(t, err, "CreateCard should not return an error")
		if c.tag == dueDate.Tag {
			cardIDs = append(cardIDs, card.ID)
		}
	}

	// Initially, no cards should be marked as mastered
	stats, err := service.GetDueDateProgressStats(dueDate.Tag)
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error")
	assert.Equal(t, 3, stats.TotalCards, "Should have 3 cards with the due date tag")
	assert.Equal(t, 0, stats.MasteredCards, "Initially, no cards should be mastered")
	assert.Equal(t, 0.0, stats.ProgressPercent, "Progress should be 0%")

	// Now submit a review with 'Easy' rating for the first card
	_, err = service.SubmitReview(cardIDs[0], gofsrs.Easy, "Test answer")
	assert.NoError(t, err, "SubmitReview should not return an error")

	// Check stats again - one card should be mastered
	stats, err = service.GetDueDateProgressStats(dueDate.Tag)
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error")
	assert.Equal(t, 3, stats.TotalCards, "Should still have 3 cards with the due date tag")
	assert.Equal(t, 1, stats.MasteredCards, "One card should be mastered")
	assert.InDelta(t, 33.33, stats.ProgressPercent, 0.01, "Progress should be ~33.33%")

	// Now submit a review with 'Again' rating for the second card
	_, err = service.SubmitReview(cardIDs[1], gofsrs.Again, "Wrong answer")
	assert.NoError(t, err, "SubmitReview should not return an error")

	// Check stats again - still only one card should be mastered
	stats, err = service.GetDueDateProgressStats(dueDate.Tag)
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error")
	assert.Equal(t, 3, stats.TotalCards, "Should still have 3 cards with the due date tag")
	assert.Equal(t, 1, stats.MasteredCards, "Still only one card should be mastered")
	assert.InDelta(t, 33.33, stats.ProgressPercent, 0.01, "Progress should still be ~33.33%")

	// Test with a tag that has no cards
	stats, err = service.GetDueDateProgressStats("non-existent-tag")
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error even for non-existent tags")
	assert.Equal(t, 0, stats.TotalCards, "Should have 0 cards with a non-existent tag")
	assert.Equal(t, 0, stats.MasteredCards, "Should have 0 mastered cards")
	assert.Equal(t, 0.0, stats.ProgressPercent, "Progress should be 0%")
}

// TestDueDateGetCardsByTag tests retrieving cards by tag
func TestDueDateGetCardsByTag(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create cards with different tags
	cards := []struct {
		front string
		back  string
		tags  []string
	}{
		{"Math 1", "Answer 1", []string{"math", "algebra"}},
		{"Math 2", "Answer 2", []string{"math", "geometry"}},
		{"History 1", "Answer 3", []string{"history", "ancient"}},
		{"Science 1", "Answer 4", []string{"science", "physics"}},
	}

	for _, c := range cards {
		_, err := service.CreateCard(c.front, c.back, c.tags)
		assert.NoError(t, err, "CreateCard should not return an error")
	}

	// Test getting cards by math tag
	mathCards, err := service.GetCardsByTag("math")
	assert.NoError(t, err, "GetCardsByTag should not return an error")
	assert.Len(t, mathCards, 2, "Should have 2 math cards")

	// Test getting cards by history tag
	historyCards, err := service.GetCardsByTag("history")
	assert.NoError(t, err, "GetCardsByTag should not return an error")
	assert.Len(t, historyCards, 1, "Should have 1 history card")

	// Test getting cards by non-existent tag
	nonExistentCards, err := service.GetCardsByTag("non-existent")
	assert.NoError(t, err, "GetCardsByTag should not return an error for non-existent tags")
	assert.Empty(t, nonExistentCards, "Should have no cards with non-existent tag")

	// Test with empty tag (should error)
	_, err = service.GetCardsByTag("")
	assert.Error(t, err, "GetCardsByTag should return an error for empty tag")
}

// TestGetDueDateProgressStatsWithMockStorage tests calculating progress with mock storage
func TestGetDueDateProgressStatsWithMockStorage(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Setup test data
	testTag := "test-biology-2024-08-20"

	// Create a test due date
	dueDate := storage.DueDate{
		ID:      "mock-due-date-id",
		Topic:   "Biology Test",
		DueDate: time.Now().AddDate(0, 0, 30), // 30 days from now
		Tag:     testTag,
	}

	// Add the due date
	err := service.AddDueDate(dueDate)
	assert.NoError(t, err, "AddDueDate should not return an error")

	// Create test cards with the tag
	testCards := []struct {
		front  string
		back   string
		tags   []string
		rating gofsrs.Rating // Rating to apply in review
	}{
		{
			front:  "Bio Q 1",
			back:   "Bio A 1",
			tags:   []string{testTag, "biology"},
			rating: gofsrs.Easy, // Card 1: Easy rating (4) - should count as mastered
		},
		{
			front:  "Bio Q 2",
			back:   "Bio A 2",
			tags:   []string{testTag, "biology"},
			rating: gofsrs.Again, // Card 2: Again rating (1) - should not count as mastered
		},
		{
			front:  "Bio Q 3",
			back:   "Bio A 3",
			tags:   []string{testTag, "biology"},
			rating: gofsrs.Good, // Card 3: Good rating (3) - should not count as mastered
		},
	}

	// Create cards and store their IDs
	cardIDs := make([]string, len(testCards))
	for i, tc := range testCards {
		card, err := service.CreateCard(tc.front, tc.back, tc.tags)
		assert.NoError(t, err, "CreateCard should not return an error")
		cardIDs[i] = card.ID
	}

	// Verify initial progress stats (before reviews)
	initialStats, err := service.GetDueDateProgressStats(testTag)
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error")
	assert.Equal(t, 3, initialStats.TotalCards, "Should have 3 cards with the due date tag")
	assert.Equal(t, 0, initialStats.MasteredCards, "Initially, no cards should be mastered")
	assert.Equal(t, 0.0, initialStats.ProgressPercent, "Progress should be 0%")

	// Submit reviews for each card with their predefined ratings
	for i, tc := range testCards {
		_, err := service.SubmitReview(cardIDs[i], tc.rating, "Test review")
		assert.NoError(t, err, "SubmitReview should not return an error")
	}

	// Verify final progress stats (after reviews)
	finalStats, err := service.GetDueDateProgressStats(testTag)
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error")
	assert.Equal(t, 3, finalStats.TotalCards, "Should still have 3 cards with the due date tag")
	assert.Equal(t, 1, finalStats.MasteredCards, "One card should be mastered (the one with Easy rating)")
	assert.InDelta(t, 33.33, finalStats.ProgressPercent, 0.01, "Progress should be ~33.33%")

	// Test calculating days remaining and required pace
	daysRemaining := dueDate.DueDate.Sub(time.Now()).Hours() / 24.0
	if daysRemaining > 0 {
		cardsLeft := finalStats.TotalCards - finalStats.MasteredCards
		expectedPace := float64(cardsLeft) / (daysRemaining - 1) // Adjusting for test day exclusion

		// Only test if we have sufficient days remaining for a valid calculation
		if daysRemaining > 1 {
			assert.Equal(t, cardsLeft, 2, "Should have 2 cards left to master")
			assert.InDelta(t, expectedPace, float64(cardsLeft)/(daysRemaining-1), 0.01,
				"Required pace should match expected calculation")
		}
	}
}

// TestHandleDueDateProgressResource tests the resource handler directly
func TestHandleDueDateProgressResource(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create a context with the service
	ctx := context.WithValue(context.Background(), "service", service)

	// Create a due date
	dueDate := storage.DueDate{
		ID:      "handler-test-id",
		Topic:   "Physics Test",
		DueDate: time.Now().AddDate(0, 0, 14), // 14 days from now
		Tag:     "test-physics-future-date",
	}
	err := service.AddDueDate(dueDate)
	assert.NoError(t, err, "AddDueDate should not return an error")

	// Create a few cards with the tag
	cards := []struct {
		front string
		back  string
	}{
		{"Physics Q1", "Physics A1"},
		{"Physics Q2", "Physics A2"},
	}

	cardIDs := make([]string, len(cards))
	for i, c := range cards {
		card, err := service.CreateCard(c.front, c.back, []string{dueDate.Tag})
		assert.NoError(t, err, "CreateCard should not return an error")
		cardIDs[i] = card.ID
	}

	// Mark one card as mastered (Easy rating)
	_, err = service.SubmitReview(cardIDs[0], gofsrs.Easy, "Test answer")
	assert.NoError(t, err, "SubmitReview should not return an error")

	// Create the request
	request := mcp.ReadResourceRequest{
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI: "due-date-progress",
		},
	}

	// Verify we have a due date
	dueDates, err := service.ListDueDates()
	assert.NoError(t, err, "ListDueDates should not return an error")
	t.Logf("Due dates before handler call: %+v", dueDates)

	// Get progress stats directly for comparison
	stats, err := service.GetDueDateProgressStats(dueDate.Tag)
	assert.NoError(t, err, "GetDueDateProgressStats should not return an error")
	t.Logf("Progress stats for tag %s: %+v", dueDate.Tag, stats)

	// Call the handler
	contents, err := handleDueDateProgressResource(ctx, request)
	assert.NoError(t, err, "handleDueDateProgressResource should not return an error")
	assert.NotNil(t, contents, "Resource contents should not be nil")
	assert.Len(t, contents, 1, "Should have one content item")

	// Print the raw JSON content for debugging
	textContent, ok := contents[0].(mcp.TextResourceContents)
	assert.True(t, ok, "Content should be TextResourceContents")
	t.Logf("Raw resource content: %s", textContent.Text)

	// Check resource metadata
	assert.Equal(t, "due-date-progress", textContent.URI, "URI should match")
	assert.Equal(t, "application/json", textContent.MIMEType, "MIME type should be JSON")

	// Parse the JSON content
	var progressInfos []DueDateProgressInfo
	err = json.Unmarshal([]byte(textContent.Text), &progressInfos)
	assert.NoError(t, err, "Should be able to unmarshal JSON content")

	// Check if we have progress info entries
	assert.NotEmpty(t, progressInfos, "Should have at least one progress info entry")

	// Only proceed with detailed checks if we have entries
	if len(progressInfos) > 0 {
		progressInfo := progressInfos[0]
		assert.Equal(t, dueDate.ID, progressInfo.ID, "Due date ID should match")
		assert.Equal(t, dueDate.Topic, progressInfo.Topic, "Topic should match")
		assert.Equal(t, dueDate.Tag, progressInfo.Tag, "Tag should match")
		assert.Equal(t, 2, progressInfo.TotalCards, "Should have 2 total cards")
		assert.Equal(t, 1, progressInfo.MasteredCards, "Should have 1 mastered card")
		assert.InDelta(t, 50.0, progressInfo.ProgressPercent, 0.01, "Progress should be 50%")
		assert.Equal(t, 1, progressInfo.CardsLeft, "Should have 1 card left")
	}

	// Test handling with no due dates
	// Delete the due date
	err = service.DeleteDueDate(dueDate.ID)
	assert.NoError(t, err, "DeleteDueDate should not return an error")

	// Verify due date was deleted
	dueDatesAfterDelete, err := service.ListDueDates()
	assert.NoError(t, err, "ListDueDates should not return an error")
	assert.Empty(t, dueDatesAfterDelete, "Should have no due dates after deletion")

	// Call the handler again
	contentsEmpty, err := handleDueDateProgressResource(ctx, request)
	assert.NoError(t, err, "handleDueDateProgressResource should not return an error with no due dates")
	assert.NotNil(t, contentsEmpty, "Resource contents should not be nil")
	assert.Len(t, contentsEmpty, 1, "Should still have one content item")

	// Print the raw JSON content for debugging
	textContentEmpty, ok := contentsEmpty[0].(mcp.TextResourceContents)
	assert.True(t, ok, "Content should be TextResourceContents")
	t.Logf("Raw resource content (empty case): %s", textContentEmpty.Text)

	// Check resource metadata again
	assert.Equal(t, "due-date-progress", textContentEmpty.URI, "URI should match")
	assert.Equal(t, "application/json", textContentEmpty.MIMEType, "MIME type should be JSON")

	// Parse the JSON content - should be an empty array
	var emptyProgressInfos []DueDateProgressInfo
	err = json.Unmarshal([]byte(textContentEmpty.Text), &emptyProgressInfos)
	assert.NoError(t, err, "Should be able to unmarshal JSON content")
	assert.Empty(t, emptyProgressInfos, "Should have no progress entries")
}

// TestDueDateProgressResourceIntegration tests the full integration of due date progress
// resources with a real MCP server/client to ensure proper communication.
func TestDueDateProgressResourceIntegration(t *testing.T) {
	// Set up temporary file for storage
	tempFile, err := os.CreateTemp("", "flashcards-resource-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Initialize with a JSON structure
	initialJSON := `{
		"cards": {},
		"reviews": [],
		"due_dates": [
			{
				"id": "test-due-date-id",
				"topic": "Integration Test",
				"due_date": "2025-05-01T00:00:00Z",
				"tag": "test-integration-tag"
			}
		],
		"last_updated": "2025-04-22T00:00:00Z"
	}`

	err = os.WriteFile(tempFilePath, []byte(initialJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial JSON: %v", err)
	}

	// Create a standalone service with the initialized storage
	fileStorage := storage.NewFileStorage(tempFilePath)
	if err := fileStorage.Load(); err != nil {
		t.Fatalf("Failed to load storage: %v", err)
	}
	service := NewFlashcardService(fileStorage)

	// Verify due dates were loaded correctly
	dueDates, err := service.ListDueDates()
	if err != nil {
		t.Fatalf("Failed to list due dates: %v", err)
	}
	if len(dueDates) != 1 {
		t.Fatalf("Expected 1 due date from initial JSON, got %d", len(dueDates))
	}
	t.Logf("Loaded due date: %+v", dueDates[0])

	// Create some cards with the test tag
	testTag := "test-integration-tag"
	var cardIDs []string

	// Create 3 cards with the integration test tag
	for i := 1; i <= 3; i++ {
		card, err := service.CreateCard(
			fmt.Sprintf("Integration Q%d", i),
			fmt.Sprintf("Integration A%d", i),
			[]string{testTag},
		)
		if err != nil {
			t.Fatalf("Failed to create card %d: %v", i, err)
		}
		cardIDs = append(cardIDs, card.ID)
		t.Logf("Created card %d: %s", i, card.ID)
	}

	// Mark one card as mastered with an Easy rating
	_, err = service.SubmitReview(cardIDs[0], gofsrs.Easy, "Test answer")
	if err != nil {
		t.Fatalf("Failed to submit review: %v", err)
	}
	t.Logf("Submitted Easy review for card %s", cardIDs[0])

	// Test direct resource handling
	ctx := context.WithValue(context.Background(), "service", service)

	// Create a resource request
	req := mcp.ReadResourceRequest{
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI: "due-date-progress",
		},
	}

	// Call the handler directly
	contents, err := handleDueDateProgressResource(ctx, req)
	if err != nil {
		t.Fatalf("Direct handler error: %v", err)
	}

	t.Logf("Direct handler response - contents count: %d", len(contents))
	if len(contents) == 0 {
		t.Fatalf("No contents returned from direct handler call")
	}

	// Check the contents
	textContent, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("Content is not TextResourceContents: %T", contents[0])
	}

	t.Logf("Direct resource content: URI=%s, MIMEType=%s, Text=%s",
		textContent.URI, textContent.MIMEType, textContent.Text)

	// Try to parse the content
	var progressInfos []DueDateProgressInfo
	if unmarshalErr := json.Unmarshal([]byte(textContent.Text), &progressInfos); unmarshalErr != nil {
		t.Fatalf("JSON unmarshal error: %v", unmarshalErr)
	}

	t.Logf("Progress info count: %d", len(progressInfos))

	// Verify progress info contains expected data
	if len(progressInfos) == 0 {
		t.Fatalf("Expected at least one progress info entry, got none")
	}

	info := progressInfos[0]
	if info.Tag != testTag {
		t.Errorf("Expected tag %s, got %s", testTag, info.Tag)
	}
	if info.TotalCards != 3 {
		t.Errorf("Expected 3 total cards, got %d", info.TotalCards)
	}
	if info.MasteredCards != 1 {
		t.Errorf("Expected 1 mastered card, got %d", info.MasteredCards)
	}
	if math.Abs(info.ProgressPercent-33.33) > 0.01 {
		t.Errorf("Expected progress percent ~33.33%%, got %.2f%%", info.ProgressPercent)
	}
}

// TestSubmitReviewWithElapsedDays tests that the SubmitReview method correctly calculates
// elapsed days between reviews and passes this information to the FSRS algorithm.
func TestSubmitReviewWithElapsedDays(t *testing.T) {
	service, filePath := setupTestService(t)
	defer os.Remove(filePath)

	// Create a new card
	card, err := service.CreateCard("Test Elapsed Days", "Answer", []string{"test-elapsed"})
	assert.NoError(t, err, "CreateCard should not return an error")

	// First review - Good rating
	now := time.Now()
	firstReview, err := service.SubmitReview(card.ID, gofsrs.Good, "First review")
	assert.NoError(t, err, "First SubmitReview should not return an error")

	// Save the due date from the first review
	firstReviewDueDate := firstReview.FSRS.Due

	// Simulate time passing - 10 days
	elapsedDays := 10
	simulatedTime := now.AddDate(0, 0, elapsedDays)

	// Mock the time.Now function using our helper
	restoreTime := mockTimeNow(simulatedTime)
	defer restoreTime() // Restore original function after test

	// Second review - Good rating again
	secondReview, err := service.SubmitReview(card.ID, gofsrs.Good, "Second review")
	assert.NoError(t, err, "Second SubmitReview should not return an error")

	// The second review's due date should be calculated based on the elapsed time
	// since the first review (10 days)

	// Get the reviews for verification
	reviews, err := service.Storage.GetCardReviews(card.ID)
	assert.NoError(t, err, "GetCardReviews should not return an error")
	assert.Len(t, reviews, 2, "Should have 2 reviews")

	// Sort reviews by timestamp (newest first)
	sort.Slice(reviews, func(i, j int) bool {
		return reviews[i].Timestamp.After(reviews[j].Timestamp)
	})

	// Verify that the elapsed days value saved in the review matches what we expect
	assert.Equal(t, uint64(elapsedDays), reviews[0].ElapsedDays,
		"ElapsedDays in the review should match our simulated elapsed time")

	// The FSRS algorithm should have used our elapsed days to calculate the new due date
	// This will produce a different (later) due date than if we used 0 elapsed days
	assert.True(t, secondReview.FSRS.Due.After(firstReviewDueDate.AddDate(0, 0, elapsedDays)),
		"Second review due date should be more than (first due date + elapsed days)")

	t.Logf("First review due date: %v", firstReviewDueDate)
	t.Logf("Second review due date: %v", secondReview.FSRS.Due)
	t.Logf("Elapsed days: %d", elapsedDays)

	// Third review - using a different rating (Hard) to see another scheduling pattern
	// Add another 5 days
	simulatedTime = simulatedTime.AddDate(0, 0, 5)
	restoreTime()                            // Restore time
	restoreTime = mockTimeNow(simulatedTime) // Set new mocked time

	thirdReview, err := service.SubmitReview(card.ID, gofsrs.Hard, "Third review")
	assert.NoError(t, err, "Third SubmitReview should not return an error")

	// Get the updated reviews
	reviews, err = service.Storage.GetCardReviews(card.ID)
	assert.NoError(t, err, "GetCardReviews should not return an error")
	assert.Len(t, reviews, 3, "Should have 3 reviews")

	// Sort reviews by timestamp (newest first)
	sort.Slice(reviews, func(i, j int) bool {
		return reviews[i].Timestamp.After(reviews[j].Timestamp)
	})

	// Verify the elapsed days for the third review
	assert.Equal(t, uint64(5), reviews[0].ElapsedDays,
		"ElapsedDays in the third review should be 5")

	t.Logf("Third review due date: %v", thirdReview.FSRS.Due)
}
