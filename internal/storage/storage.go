package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-spaced-repetition/go-fsrs"
)

// Card represents a flashcard in storage
type Card struct {
	ID             string    `json:"id"`
	Front          string    `json:"front"`
	Back           string    `json:"back"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	LastReviewedAt time.Time `json:"last_reviewed_at,omitempty"`
	// Using embedded fsrs.Card for algorithm data
	FSRS fsrs.Card `json:"fsrs"`
}

// Review represents a review record in storage
// Structured to align with fsrs.ReviewLog
type Review struct {
	ID        string      `json:"id"`
	CardID    string      `json:"card_id"`
	Rating    fsrs.Rating `json:"rating"` // Using fsrs.Rating type (Again=1, Hard=2, Good=3, Easy=4)
	Timestamp time.Time   `json:"timestamp"`
	Answer    string      `json:"answer,omitempty"`
	// Additional fields from fsrs.ReviewLog that track scheduling information
	ScheduledDays uint64     `json:"scheduled_days"`
	ElapsedDays   uint64     `json:"elapsed_days"`
	State         fsrs.State `json:"state"`
}

// DueDate represents a specific test or deadline associated with a tag.
type DueDate struct {
	ID      string    `json:"id"`       // Unique ID for the due date entry
	Topic   string    `json:"topic"`    // User-facing name (e.g., "Biology Test")
	DueDate time.Time `json:"due_date"` // The date of the test/deadline
	Tag     string    `json:"tag"`      // The tag associated with cards for this due date (e.g., "test-biology-20240715")
}

// FlashcardStore represents the data structure stored in the JSON file
type FlashcardStore struct {
	Cards       map[string]Card `json:"cards"`
	Reviews     []Review        `json:"reviews"`
	DueDates    []DueDate       `json:"due_dates"`
	LastUpdated time.Time       `json:"last_updated"`
}

// ErrCardNotFound is returned when a card is not found in the storage
var ErrCardNotFound = errors.New("card not found")
var ErrDueDateNotFound = errors.New("due date not found")

// Storage represents the storage interface for flashcards
type Storage interface {
	// Card operations
	CreateCard(front, back string, tags []string) (Card, error)
	GetCard(id string) (Card, error)
	UpdateCard(card Card) error
	DeleteCard(id string) error
	ListCards(tags []string) ([]Card, error)

	// Review operations
	AddReview(cardID string, rating fsrs.Rating, answer string) (Review, error)
	GetCardReviews(cardID string) ([]Review, error)

	// Due Date operations
	AddDueDate(dueDate DueDate) error
	ListDueDates() ([]DueDate, error)
	UpdateDueDate(dueDate DueDate) error
	DeleteDueDate(id string) error

	// File operations
	Load() error
	Save() error
}

// FileStorage implements the Storage interface using a JSON file for persistence
type FileStorage struct {
	filePath string
	store    FlashcardStore
	mu       sync.RWMutex
}

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(filePath string) *FileStorage {
	log.Printf("[Storage] Creating new FileStorage for: %s", filePath)
	return &FileStorage{
		filePath: filePath,
		store: FlashcardStore{
			Cards:    make(map[string]Card),
			Reviews:  []Review{},
			DueDates: []DueDate{},
		},
	}
}

// CreateCard creates a new flashcard
func (fs *FileStorage) CreateCard(front, back string, tags []string) (Card, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	// Initialize a new card with default FSRS values
	card := Card{
		ID:        id,
		Front:     front,
		Back:      back,
		CreatedAt: now,
		Tags:      tags,
		FSRS: fsrs.Card{
			Due:       now,
			State:     fsrs.New,
			Stability: 0,
		},
	}

	fs.store.Cards[id] = card
	fs.store.LastUpdated = now

	return card, nil
}

// GetCard retrieves a flashcard by ID
func (fs *FileStorage) GetCard(id string) (Card, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	card, exists := fs.store.Cards[id]
	if !exists {
		return Card{}, ErrCardNotFound
	}

	return card, nil
}

// UpdateCard updates an existing flashcard
func (fs *FileStorage) UpdateCard(card Card) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.store.Cards[card.ID]; !exists {
		return ErrCardNotFound
	}

	fs.store.Cards[card.ID] = card
	fs.store.LastUpdated = time.Now()

	return nil
}

// DeleteCard deletes a flashcard by ID
func (fs *FileStorage) DeleteCard(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.store.Cards[id]; !exists {
		return ErrCardNotFound
	}

	delete(fs.store.Cards, id)
	fs.store.LastUpdated = time.Now()

	return nil
}

// ListCards returns a list of all flashcards, optionally filtered by tags (must contain ANY of the tags)
func (fs *FileStorage) ListCards(tags []string) ([]Card, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check if map is initialized
	if fs.store.Cards == nil {
		return []Card{}, nil // Return empty slice if no cards exist
	}

	result := make([]Card, 0, len(fs.store.Cards))

	// If no tags specified, return all cards
	if len(tags) == 0 {
		for _, card := range fs.store.Cards {
			result = append(result, card)
		}
		return result, nil
	}

	// Filter cards: card must have ANY of the specified tags (OR logic)
	for _, card := range fs.store.Cards {
		if hasAnyTag(&card, tags) {
			result = append(result, card)
		}
	}

	return result, nil
}

// hasAnyTag checks if a card has any of the specified tags (OR logic).
func hasAnyTag(card *Card, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true // No filter means match
	}
	if card == nil || card.Tags == nil {
		return false // Cannot have any tags if card or tags are nil
	}

	// Create a map of the card's tags for efficient lookup
	cardTagsMap := make(map[string]bool)
	for _, tag := range card.Tags {
		cardTagsMap[tag] = true
	}

	// Check if the card has any of the required tags
	for _, reqTag := range requiredTags {
		if cardTagsMap[reqTag] {
			return true // Found at least one required tag
		}
	}

	return false // No required tags found
}

// hasAllTags checks if a card has all specified tags (AND logic).
// Copied from service layer for use here.
func hasAllTags(card *Card, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true // No filter means match
	}
	if card == nil || card.Tags == nil {
		return false // Cannot have all tags if card or tags are nil
	}
	cardTagsMap := make(map[string]bool)
	for _, tag := range card.Tags {
		cardTagsMap[tag] = true
	}
	for _, reqTag := range requiredTags {
		if !cardTagsMap[reqTag] {
			return false // Missing a required tag
		}
	}
	return true // All required tags found
}

// AddReview adds a new review for a card
func (fs *FileStorage) AddReview(cardID string, rating fsrs.Rating, answer string) (Review, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Check if the card exists
	card, exists := fs.store.Cards[cardID]
	if !exists {
		return Review{}, ErrCardNotFound
	}

	// Create a new review
	now := time.Now()
	review := Review{
		ID:            uuid.New().String(),
		CardID:        cardID,
		Rating:        rating,
		Timestamp:     now,
		Answer:        answer,
		ScheduledDays: card.FSRS.ScheduledDays,
		ElapsedDays:   card.FSRS.ElapsedDays,
		State:         card.FSRS.State,
	}

	fs.store.Reviews = append(fs.store.Reviews, review)
	fs.store.LastUpdated = now

	return review, nil
}

// GetCardReviews gets all reviews for a specific card
func (fs *FileStorage) GetCardReviews(cardID string) ([]Review, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check if the card exists
	if _, exists := fs.store.Cards[cardID]; !exists {
		return nil, ErrCardNotFound
	}

	// Filter reviews for the specific card
	var cardReviews []Review
	for _, review := range fs.store.Reviews {
		if review.CardID == cardID {
			cardReviews = append(cardReviews, review)
		}
	}

	return cardReviews, nil
}

// AddDueDate adds a new due date entry.
func (fs *FileStorage) AddDueDate(dueDate DueDate) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	log.Printf("[Storage:AddDueDate] Adding DueDate: ID=%s, Topic=%s, Tag=%s. Current count: %d", dueDate.ID, dueDate.Topic, dueDate.Tag, len(fs.store.DueDates))
	if fs.store.DueDates == nil {
		log.Printf("[Storage:AddDueDate] Initializing DueDates slice.")
		fs.store.DueDates = []DueDate{}
	}
	fs.store.DueDates = append(fs.store.DueDates, dueDate)
	fs.store.LastUpdated = time.Now()
	log.Printf("[Storage:AddDueDate] Added DueDate. New count: %d.", len(fs.store.DueDates))
	// DO NOT call Save() here, responsibility is in the service layer
	// return fs.Save()
	return nil
}

// ListDueDates retrieves all due date entries.
func (fs *FileStorage) ListDueDates() ([]DueDate, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	count := 0
	if fs.store.DueDates != nil {
		count = len(fs.store.DueDates)
	}
	log.Printf("[Storage:ListDueDates] Listing DueDates. Current count: %d", count)
	if fs.store.DueDates == nil {
		return []DueDate{}, nil
	}
	result := make([]DueDate, len(fs.store.DueDates))
	copy(result, fs.store.DueDates)
	return result, nil
}

// UpdateDueDate updates an existing due date entry by its ID.
func (fs *FileStorage) UpdateDueDate(updatedDueDate DueDate) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	log.Printf("[Storage:UpdateDueDate] Updating DueDate ID: %s", updatedDueDate.ID)
	if fs.store.DueDates == nil {
		return ErrDueDateNotFound
	}
	found := false
	for i, dd := range fs.store.DueDates {
		if dd.ID == updatedDueDate.ID {
			fs.store.DueDates[i] = updatedDueDate
			found = true
			log.Printf("[Storage:UpdateDueDate] Found and updated.")
			break
		}
	}
	if !found {
		return ErrDueDateNotFound
	}
	fs.store.LastUpdated = time.Now()
	log.Printf("[Storage:UpdateDueDate] Updated.")
	// DO NOT call Save() here
	// return fs.Save()
	return nil
}

// DeleteDueDate deletes a due date entry by its ID.
func (fs *FileStorage) DeleteDueDate(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	log.Printf("[Storage:DeleteDueDate] Deleting DueDate ID: %s", id)
	if fs.store.DueDates == nil {
		return ErrDueDateNotFound
	}
	initialCount := len(fs.store.DueDates)
	newDueDates := []DueDate{}
	found := false
	for _, dd := range fs.store.DueDates {
		if dd.ID != id {
			newDueDates = append(newDueDates, dd)
		} else {
			found = true
		}
	}
	if !found {
		return ErrDueDateNotFound
	}
	fs.store.DueDates = newDueDates
	fs.store.LastUpdated = time.Now()
	log.Printf("[Storage:DeleteDueDate] Deleted. Count changed from %d to %d.", initialCount, len(fs.store.DueDates))
	// DO NOT call Save() here
	// return fs.Save()
	return nil
}

// save is the internal helper for saving data without acquiring the lock again.
// Assumes the lock (write lock) is already held.
func (fs *FileStorage) save() error {
	// Ensure data structure is initialized before marshaling
	// (Redundant if Load initializes, but safe)
	if fs.store.Cards == nil {
		fs.store.Cards = make(map[string]Card)
	}
	if fs.store.Reviews == nil {
		fs.store.Reviews = []Review{}
	}
	if fs.store.DueDates == nil {
		fs.store.DueDates = []DueDate{}
	}
	fs.store.LastUpdated = time.Now() // Update timestamp

	log.Printf("[Storage:save internal] Data BEFORE Marshal - DueDate count: %d", len(fs.store.DueDates))
	if len(fs.store.DueDates) > 0 {
		log.Printf("[Storage:save internal] First DueDate Topic: %s", fs.store.DueDates[0].Topic)
	}

	dataBytes, err := json.MarshalIndent(fs.store, "", "  ")
	if err != nil {
		log.Printf("[Storage:save internal] Error marshaling data: %v", err)
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	log.Printf("[Storage:save internal] Marshaled JSON to save: %s", string(dataBytes))

	// Create directory if it doesn't exist
	dir := filepath.Dir(fs.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[Storage:save internal] Error creating directory: %v", err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to a temporary file
	tempFile := fs.filePath + ".tmp"
	if err := os.WriteFile(tempFile, dataBytes, 0644); err != nil {
		os.Remove(tempFile)
		log.Printf("[Storage:save internal] Error writing temp file: %v", err)
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename the temporary file to the target file (atomic operation on most systems)
	if err := os.Rename(tempFile, fs.filePath); err != nil {
		os.Remove(tempFile)
		log.Printf("[Storage:save internal] Error renaming temp file: %v", err)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	log.Printf("[Storage:save internal] Save successful.")
	return nil
}

// Load loads the flashcards data from the file
func (fs *FileStorage) Load() error {
	fs.mu.Lock() // Acquire Write lock for potential initial save
	defer fs.mu.Unlock()
	log.Printf("[Storage:Load] Attempting to load from: %s", fs.filePath)
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		log.Printf("[Storage:Load] File not found, initializing empty store.")
		fs.store = FlashcardStore{
			Cards:    make(map[string]Card),
			Reviews:  []Review{},
			DueDates: []DueDate{},
		}
		// Explicitly save the initial empty structure to ensure the file exists
		log.Printf("[Storage:Load] Saving initial empty store.")
		// Call internal save which assumes lock is held
		if saveErr := fs.save(); saveErr != nil {
			log.Printf("[Storage:Load] Error saving initial empty store: %v", saveErr)
			return fmt.Errorf("failed to save initial empty store: %w", saveErr)
		}
		return nil
	}

	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		log.Printf("[Storage:Load] Error reading file: %v", err)
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	if len(data) == 0 {
		log.Printf("[Storage:Load] File is empty, initializing empty store.")
		fs.store = FlashcardStore{
			Cards:    make(map[string]Card),
			Reviews:  []Review{},
			DueDates: []DueDate{},
		}
		return nil
	}

	log.Printf("[Storage:Load] Read raw data from file: %s", string(data))

	var store FlashcardStore
	if err := json.Unmarshal(data, &store); err != nil {
		log.Printf("[Storage:Load] Error unmarshaling JSON: %v", err)
		return fmt.Errorf("failed to unmarshal storage data: %w", err)
	}
	log.Printf("[Storage:Load] Successfully unmarshaled. DueDate count IMMEDIATELY after unmarshal: %d", len(store.DueDates))

	// Initialize maps/slices if they are nil after unmarshal (e.g., loading older format)
	if store.Cards == nil {
		store.Cards = make(map[string]Card)
	}
	if store.Reviews == nil {
		store.Reviews = []Review{}
	}
	if store.DueDates == nil {
		log.Printf("[Storage:Load] DueDates was nil in JSON, initializing empty slice.")
		store.DueDates = []DueDate{}
	}

	fs.store = store
	log.Printf("[Storage:Load] Load successful. In-memory DueDate count AFTER load: %d", len(fs.store.DueDates))
	if len(fs.store.DueDates) > 0 {
		log.Printf("[Storage:Load] First in-memory DueDate Topic AFTER load: %s", fs.store.DueDates[0].Topic)
	}
	return nil
}

// Save saves the flashcards data to the file atomically.
func (fs *FileStorage) Save() error {
	fs.mu.Lock() // Acquire Write lock for saving
	defer fs.mu.Unlock()
	// Call internal save helper which assumes lock is held
	return fs.save()
}
