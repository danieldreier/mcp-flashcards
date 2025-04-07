package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-spaced-repetition/go-fsrs"
)

// Card represents a flashcard in storage
type Card struct {
	ID        string    `json:"id"`
	Front     string    `json:"front"`
	Back      string    `json:"back"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags,omitempty"`
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

// FlashcardStore represents the data structure stored in the JSON file
type FlashcardStore struct {
	Cards       map[string]Card `json:"cards"`
	Reviews     []Review        `json:"reviews"`
	LastUpdated time.Time       `json:"last_updated"`
}

// ErrCardNotFound is returned when a card is not found in the storage
var ErrCardNotFound = errors.New("card not found")

// Storage represents the storage interface for flashcards
type Storage interface {
	// Card operations
	CreateCard(front, back string, tags []string) (Card, error)
	GetCard(id string) (Card, error)
	UpdateCard(card Card) error
	DeleteCard(id string) error
	ListCards(tags []string) ([]Card, error)

	// Review operations
	// Using fsrs.Rating type for proper integration with algorithm
	AddReview(cardID string, rating fsrs.Rating, answer string) (Review, error)
	GetCardReviews(cardID string) ([]Review, error)

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
	return &FileStorage{
		filePath: filePath,
		store: FlashcardStore{
			Cards:   make(map[string]Card),
			Reviews: []Review{},
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

// ListCards returns a list of all flashcards, optionally filtered by tags
func (fs *FileStorage) ListCards(tags []string) ([]Card, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []Card

	// If no tags specified, return all cards
	if len(tags) == 0 {
		result = make([]Card, 0, len(fs.store.Cards))
		for _, card := range fs.store.Cards {
			result = append(result, card)
		}
		return result, nil
	}

	// Filter cards by tags
	result = make([]Card, 0)
	for _, card := range fs.store.Cards {
		if containsAnyTag(card.Tags, tags) {
			result = append(result, card)
		}
	}

	return result, nil
}

// containsAnyTag checks if any of the target tags are in the source tags
func containsAnyTag(sourceTags, targetTags []string) bool {
	if len(targetTags) == 0 {
		return true
	}

	tagMap := make(map[string]bool)
	for _, tag := range sourceTags {
		tagMap[tag] = true
	}

	for _, tag := range targetTags {
		if tagMap[tag] {
			return true
		}
	}

	return false
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
		ScheduledDays: 0, // This should be calculated by FSRS
		ElapsedDays:   0, // This should be calculated by FSRS
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

// Load loads the flashcards data from the file
func (fs *FileStorage) Load() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// If the file doesn't exist, initialize an empty store
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		fs.store = FlashcardStore{
			Cards:       make(map[string]Card),
			Reviews:     []Review{},
			LastUpdated: time.Now(),
		}
		return nil
	}

	// Read the file
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	// Unmarshal the JSON
	var store FlashcardStore
	if err := json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("failed to unmarshal storage data: %w", err)
	}

	// Initialize maps if they are nil
	if store.Cards == nil {
		store.Cards = make(map[string]Card)
	}

	fs.store = store
	return nil
}

// Save saves the flashcards data to the file
func (fs *FileStorage) Save() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Update the last updated timestamp
	fs.store.LastUpdated = time.Now()

	// Marshal the store to JSON
	data, err := json.MarshalIndent(fs.store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fs.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to a temporary file and then rename to ensure atomic writes
	tempFile := fs.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Rename the temporary file to the target file (atomic operation)
	if err := os.Rename(tempFile, fs.filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}
