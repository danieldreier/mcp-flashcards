package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/open-spaced-repetition/go-fsrs"
)

// createTempFile creates a temporary file for testing
func createTempFile(t *testing.T) string {
	t.Helper()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "flashcards-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}

	// Return the path to a file in the temp directory
	return filepath.Join(tempDir, "test-flashcards.json")
}

// cleanupTempFile removes the temporary file and its directory
func cleanupTempFile(t *testing.T, path string) {
	t.Helper()

	// Get the directory from the file path
	dir := filepath.Dir(path)

	// Remove the directory and all its contents
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Error cleaning up temp directory: %v", err)
	}
}

// TestFileStorage_CreateCard tests creating a new card
func TestFileStorage_CreateCard(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create a card
	card, err := storage.CreateCard("Test Front", "Test Back", []string{"tag1", "tag2"})
	if err != nil {
		t.Fatalf("Error creating card: %v", err)
	}

	// Verify the card
	if card.ID == "" {
		t.Error("Expected card to have an ID")
	}
	if card.Front != "Test Front" {
		t.Errorf("Expected card front to be 'Test Front', got %q", card.Front)
	}
	if card.Back != "Test Back" {
		t.Errorf("Expected card back to be 'Test Back', got %q", card.Back)
	}
	if len(card.Tags) != 2 || card.Tags[0] != "tag1" || card.Tags[1] != "tag2" {
		t.Errorf("Expected card tags to be ['tag1', 'tag2'], got %v", card.Tags)
	}
	if !card.CreatedAt.Before(time.Now().Add(time.Second)) {
		t.Error("Expected card creation time to be in the past")
	}
	if card.FSRS.State != fsrs.New {
		t.Errorf("Expected card state to be New, got %v", card.FSRS.State)
	}

	// Save the card to the file
	if err := storage.Save(); err != nil {
		t.Fatalf("Error saving storage: %v", err)
	}

	// Verify the file exists
	_, err = os.Stat(tempFile)
	if os.IsNotExist(err) {
		t.Error("Expected file to exist after save")
	}
}

// TestFileStorage_GetCard tests retrieving a card
func TestFileStorage_GetCard(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create a card
	card, err := storage.CreateCard("Test Front", "Test Back", []string{"tag1"})
	if err != nil {
		t.Fatalf("Error creating card: %v", err)
	}

	// Get the card
	retrievedCard, err := storage.GetCard(card.ID)
	if err != nil {
		t.Fatalf("Error getting card: %v", err)
	}

	// Verify the retrieved card
	if retrievedCard.ID != card.ID {
		t.Errorf("Expected card ID to be %q, got %q", card.ID, retrievedCard.ID)
	}
	if retrievedCard.Front != card.Front {
		t.Errorf("Expected card front to be %q, got %q", card.Front, retrievedCard.Front)
	}

	// Try to get a non-existent card
	_, err = storage.GetCard("non-existent-id")
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound, got %v", err)
	}
}

// TestFileStorage_UpdateCard tests updating a card
func TestFileStorage_UpdateCard(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create a card
	card, err := storage.CreateCard("Original Front", "Original Back", []string{"tag1"})
	if err != nil {
		t.Fatalf("Error creating card: %v", err)
	}

	// Update the card
	card.Front = "Updated Front"
	card.Back = "Updated Back"
	card.Tags = []string{"tag1", "tag2"}

	err = storage.UpdateCard(card)
	if err != nil {
		t.Fatalf("Error updating card: %v", err)
	}

	// Get the updated card
	retrievedCard, err := storage.GetCard(card.ID)
	if err != nil {
		t.Fatalf("Error getting card: %v", err)
	}

	// Verify the updated card
	if retrievedCard.Front != "Updated Front" {
		t.Errorf("Expected card front to be 'Updated Front', got %q", retrievedCard.Front)
	}
	if retrievedCard.Back != "Updated Back" {
		t.Errorf("Expected card back to be 'Updated Back', got %q", retrievedCard.Back)
	}
	if len(retrievedCard.Tags) != 2 || retrievedCard.Tags[0] != "tag1" || retrievedCard.Tags[1] != "tag2" {
		t.Errorf("Expected card tags to be ['tag1', 'tag2'], got %v", retrievedCard.Tags)
	}

	// Try to update a non-existent card
	nonExistentCard := Card{
		ID:    "non-existent-id",
		Front: "Front",
		Back:  "Back",
	}
	err = storage.UpdateCard(nonExistentCard)
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound, got %v", err)
	}
}

// TestFileStorage_DeleteCard tests deleting a card
func TestFileStorage_DeleteCard(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create a card
	card, err := storage.CreateCard("Test Front", "Test Back", nil)
	if err != nil {
		t.Fatalf("Error creating card: %v", err)
	}

	// Delete the card
	err = storage.DeleteCard(card.ID)
	if err != nil {
		t.Fatalf("Error deleting card: %v", err)
	}

	// Try to get the deleted card
	_, err = storage.GetCard(card.ID)
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound after deletion, got %v", err)
	}

	// Try to delete a non-existent card
	err = storage.DeleteCard("non-existent-id")
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound, got %v", err)
	}
}

// TestFileStorage_ListCards tests listing cards with and without tag filtering
func TestFileStorage_ListCards(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create cards with different tags
	card1, _ := storage.CreateCard("Card 1", "Back 1", []string{"tag1", "tag2"})
	_, _ = storage.CreateCard("Card 2", "Back 2", []string{"tag2", "tag3"})
	_, _ = storage.CreateCard("Card 3", "Back 3", []string{"tag3"})

	// List all cards
	allCards, err := storage.ListCards(nil)
	if err != nil {
		t.Fatalf("Error listing cards: %v", err)
	}
	if len(allCards) != 3 {
		t.Errorf("Expected 3 cards, got %d", len(allCards))
	}

	// List cards with tag1
	tag1Cards, err := storage.ListCards([]string{"tag1"})
	if err != nil {
		t.Fatalf("Error listing cards with tag1: %v", err)
	}
	if len(tag1Cards) != 1 {
		t.Errorf("Expected 1 card with tag1, got %d", len(tag1Cards))
	}
	if tag1Cards[0].ID != card1.ID {
		t.Errorf("Expected card with ID %q, got %q", card1.ID, tag1Cards[0].ID)
	}

	// List cards with tag2
	tag2Cards, err := storage.ListCards([]string{"tag2"})
	if err != nil {
		t.Fatalf("Error listing cards with tag2: %v", err)
	}
	if len(tag2Cards) != 2 {
		t.Errorf("Expected 2 cards with tag2, got %d", len(tag2Cards))
	}

	// List cards with tag3
	tag3Cards, err := storage.ListCards([]string{"tag3"})
	if err != nil {
		t.Fatalf("Error listing cards with tag3: %v", err)
	}
	if len(tag3Cards) != 2 {
		t.Errorf("Expected 2 cards with tag3, got %d", len(tag3Cards))
	}

	// List cards with multiple tags (tag1 OR tag3) - must explicitly use OR logic
	multiTagCards, err := storage.ListCards([]string{"tag1", "tag3"}, false) // false = OR logic
	if err != nil {
		t.Fatalf("Error listing cards with multiple tags: %v", err)
	}
	if len(multiTagCards) != 3 {
		t.Errorf("Expected 3 cards with tag1 OR tag3, got %d", len(multiTagCards))
	}

	// List cards with non-existent tag
	nonExistentTagCards, err := storage.ListCards([]string{"non-existent-tag"})
	if err != nil {
		t.Fatalf("Error listing cards with non-existent tag: %v", err)
	}
	if len(nonExistentTagCards) != 0 {
		t.Errorf("Expected 0 cards with non-existent tag, got %d", len(nonExistentTagCards))
	}
}

// TestFileStorage_AddReview tests adding a review
func TestFileStorage_AddReview(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create a card
	card, _ := storage.CreateCard("Test Front", "Test Back", nil)

	// Add a review
	review, err := storage.AddReview(card.ID, fsrs.Good, "User answer")
	if err != nil {
		t.Fatalf("Error adding review: %v", err)
	}

	// Verify the review
	if review.ID == "" {
		t.Error("Expected review to have an ID")
	}
	if review.CardID != card.ID {
		t.Errorf("Expected review card ID to be %q, got %q", card.ID, review.CardID)
	}
	if review.Rating != fsrs.Good {
		t.Errorf("Expected review rating to be Good, got %v", review.Rating)
	}
	if review.Answer != "User answer" {
		t.Errorf("Expected review answer to be 'User answer', got %q", review.Answer)
	}
	if !review.Timestamp.Before(time.Now().Add(time.Second)) {
		t.Error("Expected review timestamp to be in the past")
	}

	// Try to add a review for a non-existent card
	_, err = storage.AddReview("non-existent-id", fsrs.Good, "")
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound, got %v", err)
	}
}

// TestFileStorage_GetCardReviews tests retrieving reviews for a card
func TestFileStorage_GetCardReviews(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create a card
	card, _ := storage.CreateCard("Test Front", "Test Back", nil)

	// Add reviews
	storage.AddReview(card.ID, fsrs.Again, "")
	storage.AddReview(card.ID, fsrs.Hard, "")
	storage.AddReview(card.ID, fsrs.Good, "")

	// Get the reviews
	reviews, err := storage.GetCardReviews(card.ID)
	if err != nil {
		t.Fatalf("Error getting reviews: %v", err)
	}

	// Verify the reviews
	if len(reviews) != 3 {
		t.Errorf("Expected 3 reviews, got %d", len(reviews))
	}
	if reviews[0].Rating != fsrs.Again {
		t.Errorf("Expected first review rating to be Again, got %v", reviews[0].Rating)
	}
	if reviews[1].Rating != fsrs.Hard {
		t.Errorf("Expected second review rating to be Hard, got %v", reviews[1].Rating)
	}
	if reviews[2].Rating != fsrs.Good {
		t.Errorf("Expected third review rating to be Good, got %v", reviews[2].Rating)
	}

	// Try to get reviews for a non-existent card
	_, err = storage.GetCardReviews("non-existent-id")
	if err != ErrCardNotFound {
		t.Errorf("Expected ErrCardNotFound, got %v", err)
	}
}

// TestFileStorage_SaveAndLoad tests saving and loading data
func TestFileStorage_SaveAndLoad(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage1 := NewFileStorage(tempFile)

	// Create some cards and reviews
	card1, _ := storage1.CreateCard("Card 1", "Back 1", []string{"tag1"})
	card2, _ := storage1.CreateCard("Card 2", "Back 2", []string{"tag2"})

	storage1.AddReview(card1.ID, fsrs.Good, "")
	storage1.AddReview(card2.ID, fsrs.Hard, "")

	// Save the data
	if err := storage1.Save(); err != nil {
		t.Fatalf("Error saving data: %v", err)
	}

	// Create a new storage instance with the same file
	storage2 := NewFileStorage(tempFile)

	// Load the data
	if err := storage2.Load(); err != nil {
		t.Fatalf("Error loading data: %v", err)
	}

	// Verify the loaded cards
	loadedCards, _ := storage2.ListCards(nil)
	if len(loadedCards) != 2 {
		t.Errorf("Expected 2 loaded cards, got %d", len(loadedCards))
	}

	// Get card1 and verify its data
	loadedCard1, err := storage2.GetCard(card1.ID)
	if err != nil {
		t.Fatalf("Error getting loaded card1: %v", err)
	}
	if loadedCard1.Front != "Card 1" {
		t.Errorf("Expected loaded card1 front to be 'Card 1', got %q", loadedCard1.Front)
	}

	// Verify the loaded reviews
	loadedReviews1, _ := storage2.GetCardReviews(card1.ID)
	if len(loadedReviews1) != 1 {
		t.Errorf("Expected 1 loaded review for card1, got %d", len(loadedReviews1))
	}

	loadedReviews2, _ := storage2.GetCardReviews(card2.ID)
	if len(loadedReviews2) != 1 {
		t.Errorf("Expected 1 loaded review for card2, got %d", len(loadedReviews2))
	}
}

// TestFileStorage_NonExistingFile tests loading from a non-existing file
func TestFileStorage_NonExistingFile(t *testing.T) {
	// Create a temporary file path that doesn't exist
	tempDir, _ := os.MkdirTemp("", "flashcards-test")
	defer os.RemoveAll(tempDir)
	tempFile := filepath.Join(tempDir, "non-existing.json")

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Load should create an empty store
	err := storage.Load()
	if err != nil {
		t.Fatalf("Error loading from non-existing file: %v", err)
	}

	// Verify we have an empty store
	cards, _ := storage.ListCards(nil)
	if len(cards) != 0 {
		t.Errorf("Expected 0 cards after loading non-existing file, got %d", len(cards))
	}

	// Create a card and save
	storage.CreateCard("Test Front", "Test Back", nil)

	err = storage.Save()
	if err != nil {
		t.Fatalf("Error saving to new file: %v", err)
	}

	// Verify the file was created
	_, err = os.Stat(tempFile)
	if os.IsNotExist(err) {
		t.Error("Expected file to be created after save")
	}
}

// TestFileStorage_CorruptedFile tests handling a corrupted file
func TestFileStorage_CorruptedFile(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Write invalid JSON to the file
	err := os.WriteFile(tempFile, []byte("This is not valid JSON"), 0644)
	if err != nil {
		t.Fatalf("Error writing corrupted file: %v", err)
	}

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Load should return an error
	err = storage.Load()
	if err == nil {
		t.Error("Expected error when loading corrupted file, got nil")
	}
}

// Test UUID generation
func TestFileStorage_UUIDs(t *testing.T) {
	// Create a temporary file for the test
	tempFile := createTempFile(t)
	defer cleanupTempFile(t, tempFile)

	// Create a new storage instance
	storage := NewFileStorage(tempFile)

	// Create multiple cards and verify unique IDs
	card1, _ := storage.CreateCard("Card 1", "Back 1", nil)
	card2, _ := storage.CreateCard("Card 2", "Back 2", nil)
	card3, _ := storage.CreateCard("Card 3", "Back 3", nil)

	// Verify unique IDs
	if card1.ID == card2.ID || card1.ID == card3.ID || card2.ID == card3.ID {
		t.Error("Expected unique IDs for cards")
	}

	// Try to parse the IDs as UUIDs
	_, err1 := uuid.Parse(card1.ID)
	_, err2 := uuid.Parse(card2.ID)
	_, err3 := uuid.Parse(card3.ID)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Error("Expected valid UUID format for card IDs")
	}

	// Add reviews and verify unique IDs
	review1, _ := storage.AddReview(card1.ID, fsrs.Good, "")
	review2, _ := storage.AddReview(card1.ID, fsrs.Good, "")

	if review1.ID == review2.ID {
		t.Error("Expected unique IDs for reviews")
	}

	// Try to parse the review IDs as UUIDs
	_, err4 := uuid.Parse(review1.ID)
	_, err5 := uuid.Parse(review2.ID)

	if err4 != nil || err5 != nil {
		t.Error("Expected valid UUID format for review IDs")
	}
}

func TestFileStorage_Load_WithDueDates(t *testing.T) {
	t.Parallel() // Mark test as parallelizable

	// Prepare sample data with DueDates
	testID := uuid.NewString()
	now := time.Now().UTC().Truncate(time.Second) // Truncate for easier comparison
	expectedDueDate := DueDate{
		ID:      testID,
		Topic:   "Unit Test Topic",
		DueDate: now.AddDate(0, 0, 7), // Due in 7 days
		Tag:     "test-unit-topic",
	}
	storeData := FlashcardStore{
		Cards:       make(map[string]Card),
		Reviews:     []Review{},
		DueDates:    []DueDate{expectedDueDate},
		LastUpdated: now,
	}

	jsonData, err := json.MarshalIndent(storeData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Create temp file and write data
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "load_due_dates_test.json")
	if err := os.WriteFile(tempFilePath, jsonData, 0644); err != nil {
		t.Fatalf("Failed to write test data file: %v", err)
	}

	// Create FileStorage and Load
	fs := NewFileStorage(tempFilePath)
	err = fs.Load()
	if err != nil {
		t.Fatalf("fs.Load() failed: %v", err)
	}

	// --- Assertions ---
	fs.mu.RLock() // Lock for reading internal store state
	defer fs.mu.RUnlock()

	if fs.store.DueDates == nil {
		t.Fatalf("fs.store.DueDates is nil after Load, expected a slice")
	}

	if len(fs.store.DueDates) != 1 {
		t.Fatalf("Expected 1 due date after Load, got %d", len(fs.store.DueDates))
	}

	loadedDueDate := fs.store.DueDates[0]

	// Use cmp.Diff for detailed comparison, handling potential time zone differences
	// Allow time comparison with a small tolerance (e.g., 1 second) if needed, though Truncate should help.
	if diff := cmp.Diff(expectedDueDate, loadedDueDate); diff != "" {
		t.Errorf("Loaded DueDate mismatch (-want +got):\n%s", diff)
	}

	// Explicit checks for key fields
	if loadedDueDate.ID != expectedDueDate.ID {
		t.Errorf("ID mismatch: want %s, got %s", expectedDueDate.ID, loadedDueDate.ID)
	}
	if loadedDueDate.Topic != expectedDueDate.Topic {
		t.Errorf("Topic mismatch: want %s, got %s", expectedDueDate.Topic, loadedDueDate.Topic)
	}
	if !loadedDueDate.DueDate.Equal(expectedDueDate.DueDate) {
		t.Errorf("DueDate mismatch: want %s, got %s", expectedDueDate.DueDate, loadedDueDate.DueDate)
	}
	if loadedDueDate.Tag != expectedDueDate.Tag {
		t.Errorf("Tag mismatch: want %s, got %s", expectedDueDate.Tag, loadedDueDate.Tag)
	}
}
