// Package main provides implementation for the flashcards MCP service.
package main

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
	"github.com/danieldreier/mcp-flashcards/internal/storage"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// FlashcardService manages operations for flashcards with storage and FSRS algorithm
type FlashcardService struct {
	Storage     storage.Storage // Interface for storage operations
	FSRSManager fsrs.FSRSManager
}

// NewFlashcardService creates a new FlashcardService
func NewFlashcardService(storage storage.Storage) *FlashcardService {
	return &FlashcardService{
		Storage:     storage,
		FSRSManager: fsrs.NewFSRSManager(),
	}
}

// CreateCard creates a new flashcard using the Storage layer
func (s *FlashcardService) CreateCard(front, back string, tags []string) (Card, error) {
	// Delegate creation to the storage layer, which handles FSRS initialization
	storageCard, err := s.Storage.CreateCard(front, back, tags)
	if err != nil {
		return Card{}, fmt.Errorf("error creating card in storage: %w", err)
	}

	// Persist changes to disk (Save should ideally be part of the storage method)
	// Assuming storage methods don't auto-save for now.
	if err := s.Storage.Save(); err != nil {
		// Attempt to rollback? Difficult. Log error.
		fmt.Printf("Warning: failed to save storage after creating card %s: %v\n", storageCard.ID, err)
		// Continue anyway, card exists in memory layer of storage
	}

	// Convert storage.Card to our main Card type for the response
	createdCard := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	return createdCard, nil
}

// UpdateCard updates an existing flashcard
func (s *FlashcardService) UpdateCard(cardID string, front, back string, tags []string) (Card, error) {
	// Get the card from storage
	storageCard, err := s.Storage.GetCard(cardID)
	if err != nil {
		return Card{}, fmt.Errorf("error getting card: %w", err)
	}

	// Update card fields (only update non-empty fields)
	if front != "" {
		storageCard.Front = front
	}
	if back != "" {
		storageCard.Back = back
	}
	// Note: We should decide if providing an empty tag list means clearing tags
	// or doing nothing. Current storage interface updates the whole card.
	// Let's assume [] means update to empty, nil means no change.
	// For now, service passes tags directly if provided.
	if tags != nil { // Allow updating to empty list by passing []string{}
		storageCard.Tags = tags
	}

	// Save the updated card back to storage
	if err := s.Storage.UpdateCard(storageCard); err != nil {
		return Card{}, fmt.Errorf("error updating card: %w", err)
	}

	// Persist changes to disk
	if err := s.Storage.Save(); err != nil {
		return Card{}, fmt.Errorf("error saving storage: %w", err)
	}

	// Convert storage.Card back to our main Card type for the response
	updatedCard := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	return updatedCard, nil
}

// DeleteCard deletes a flashcard
func (s *FlashcardService) DeleteCard(cardID string) error {
	// Delete the card from storage
	if err := s.Storage.DeleteCard(cardID); err != nil {
		return fmt.Errorf("error deleting card: %w", err)
	}

	// Persist changes to disk
	if err := s.Storage.Save(); err != nil {
		return fmt.Errorf("error saving storage: %w", err)
	}

	return nil
}

// ListCards lists all flashcards, optionally filtered by tags
func (s *FlashcardService) ListCards(filterTags []string, includeStats bool) ([]Card, CardStats, error) {
	// Use storage ListCards with the filter
	storageCards, err := s.Storage.ListCards(filterTags)
	if err != nil {
		return nil, CardStats{}, fmt.Errorf("error listing cards from storage: %w", err)
	}

	// Convert storage.Card array to our main Card type array
	cards := make([]Card, 0, len(storageCards))
	for _, storageCard := range storageCards {
		card := Card{
			ID:        storageCard.ID,
			Front:     storageCard.Front,
			Back:      storageCard.Back,
			CreatedAt: storageCard.CreatedAt,
			Tags:      storageCard.Tags,
			FSRS:      storageCard.FSRS,
		}
		cards = append(cards, card)
	}

	// Calculate stats if requested
	var stats CardStats
	if includeStats {
		// Fetch all cards for stats calculation, regardless of filter
		allStorageCards, err := s.Storage.ListCards(nil)
		if err != nil {
			// Log error but proceed with potentially empty stats
			fmt.Printf("Warning: error getting all cards for stats: %v\n", err)
			stats = CardStats{TotalCards: len(storageCards)} // Use filtered count as fallback?
		} else {
			stats = s.calculateStats(allStorageCards)
		}
	}

	return cards, stats, nil
}

// GetDueCard returns the next card due for review with statistics, optionally filtered by tags
func (s *FlashcardService) GetDueCard(filterTags []string) (Card, CardStats, error) {
	// Get all cards from storage first to calculate overall statistics
	allCards, err := s.Storage.ListCards(nil)
	if err != nil {
		return Card{}, CardStats{}, fmt.Errorf("error listing all cards: %w", err)
	}

	// Calculate overall statistics based on all cards
	stats := s.calculateStats(allCards)

	// Get cards matching filter (reuse ListCards)
	cardsToConsider, err := s.Storage.ListCards(filterTags)
	if err != nil {
		// Should not happen if ListCards(nil) worked, but handle anyway
		return Card{}, stats, fmt.Errorf("error listing cards with filter tags: %w", err)
	}

	// Current time for priority calculation
	now := time.Now()

	// Find due cards from the filtered list and calculate priority
	var dueCards []struct {
		card     Card
		priority float64
	}

	for _, storageCard := range cardsToConsider { // Iterate over the filtered list
		// Consider cards due now or in the past
		if !storageCard.FSRS.Due.After(now) {
			priority := s.FSRSManager.GetReviewPriority(storageCard.FSRS.State, storageCard.FSRS.Due, now)
			// Convert storage.Card to our main Card type here
			card := Card{
				ID:        storageCard.ID,
				Front:     storageCard.Front,
				Back:      storageCard.Back,
				CreatedAt: storageCard.CreatedAt,
				Tags:      storageCard.Tags,
				FSRS:      storageCard.FSRS,
			}
			dueCards = append(dueCards, struct {
				card     Card
				priority float64
			}{card, priority})
		}
	}

	// Sort the due cards (from the filtered list) by priority (highest first)
	sort.Slice(dueCards, func(i, j int) bool {
		return dueCards[i].priority > dueCards[j].priority
	})

	// Return highest priority card from the filtered set or error if none due
	if len(dueCards) == 0 {
		// Check if the filter was the reason for no due cards
		if len(filterTags) > 0 && len(cardsToConsider) == 0 {
			return Card{}, stats, fmt.Errorf("no cards found with the specified tags: %v", filterTags)
		} else if len(filterTags) > 0 {
			return Card{}, stats, fmt.Errorf("no cards due for review with the specified tags: %v", filterTags)
		}
		// No filter, but no cards due
		return Card{}, stats, fmt.Errorf("no cards due for review")
	}

	// Return the highest priority card from the filtered due list, along with overall stats
	return dueCards[0].card, stats, nil
}

// calculateStats calculates statistics from card and review data
func (s *FlashcardService) calculateStats(cards []storage.Card) CardStats {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Count total and due cards
	totalCards := len(cards)
	dueCards := 0
	for _, card := range cards {
		if !card.FSRS.Due.After(now) {
			dueCards++
		}
	}

	// Get today's reviews and count correct answers
	var reviewsToday []storage.Review
	correctReviewsToday := 0
	for _, card := range cards {
		cardReviews, err := s.Storage.GetCardReviews(card.ID)
		if err == nil {
			for _, review := range cardReviews {
				if !review.Timestamp.Before(today) {
					reviewsToday = append(reviewsToday, review)
					// Rating 3 (Good) or 4 (Easy) is considered correct
					if review.Rating >= gofsrs.Good {
						correctReviewsToday++
					}
				}
			}
		}
	}

	// Calculate retention rate (correct answers / total reviews today)
	retentionRate := 0.0
	if len(reviewsToday) > 0 {
		retentionRate = float64(correctReviewsToday) / float64(len(reviewsToday)) * 100.0
	}

	return CardStats{
		TotalCards:    totalCards,
		DueCards:      dueCards,
		ReviewsToday:  len(reviewsToday),
		RetentionRate: retentionRate,
	}
}

// SubmitReview processes a review for a card and updates its state using the FSRS algorithm
func (s *FlashcardService) SubmitReview(cardID string, rating gofsrs.Rating, answer string) (Card, error) {
	// Get the card from storage
	storageCard, err := s.Storage.GetCard(cardID)
	if err != nil {
		return Card{}, fmt.Errorf("error getting card: %w", err)
	}

	now := time.Now()

	// Use FSRS manager to schedule the review using the go-fsrs library
	// Pass the existing FSRS state from the storageCard.
	updatedState, newDueDate := s.FSRSManager.ScheduleReview(storageCard.FSRS.State, rating, now)

	// Update the card with new state information
	storageCard.FSRS.State = updatedState
	storageCard.FSRS.Due = newDueDate // Use the returned due date
	storageCard.LastReviewedAt = now  // Record last reviewed time (field should exist now)

	// Save the updated card state back to storage
	if err := s.Storage.UpdateCard(storageCard); err != nil {
		return Card{}, fmt.Errorf("error updating card: %w", err)
	}

	// Add review to storage
	// The storage AddReview should probably update the FSRS fields on the review log itself.
	reviewLog, err := s.Storage.AddReview(cardID, rating, answer)
	if err != nil {
		// Attempt to rollback card update? Maybe too complex for now.
		return Card{}, fmt.Errorf("error adding review: %w", err)
	}
	_ = reviewLog // Use reviewLog if needed later

	// Persist changes to disk
	if err := s.Storage.Save(); err != nil {
		return Card{}, fmt.Errorf("error saving storage: %w", err)
	}

	// Convert updated storage.Card to our main Card type
	updatedCard := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	return updatedCard, nil
}

// AnalyzeLearning provides insights based on review history
func (s *FlashcardService) AnalyzeLearning() (string, error) {
	// Fetch all cards and their review histories
	cards, err := s.Storage.ListCards(nil)
	if err != nil {
		return "", fmt.Errorf("error getting all cards for analysis: %w", err)
	}

	if len(cards) == 0 {
		return "No cards available to analyze yet. Let's create some!", nil
	}

	// Simple analysis: Find the card reviewed most recently with the lowest rating (1 or 2)
	var worstReview *storage.Review = nil
	var worstCard *storage.Card = nil // Use pointer to allow nil
	latestTime := time.Time{}

	for i := range cards { // Iterate using index to get addressable card
		card := cards[i] // Get a copy of the card for this iteration
		reviews, err := s.Storage.GetCardReviews(card.ID)
		if err != nil {
			continue // Skip cards with errors fetching reviews
		}
		for j := range reviews {
			review := reviews[j]              // Get a copy
			if review.Rating <= gofsrs.Hard { // Again or Hard
				if review.Timestamp.After(latestTime) {
					latestTime = review.Timestamp
					worstReview = &review
					// Assign the address of the card from the original slice
					worstCard = &cards[i]
				}
			}
		}
	}

	if worstCard != nil && worstReview != nil {
		return fmt.Sprintf("It looks like the card '%s' was challenging (rated %d on %s). Maybe we can break down the concept or create related cards?",
			worstCard.Front, worstReview.Rating, worstReview.Timestamp.Format(time.RFC822)), nil
	}

	return "Great job so far! All recent reviews look good. Keep up the excellent work!", nil
}

// GetTags returns a map of tags to the count of cards with that tag
func (s *FlashcardService) GetTags() (map[string]int, error) {
	cards, err := s.Storage.ListCards(nil)
	if err != nil {
		return nil, fmt.Errorf("error getting cards for tags: %w", err)
	}

	tagCounts := make(map[string]int)
	for _, card := range cards {
		for _, tag := range card.Tags {
			tagCounts[tag]++
		}
	}
	return tagCounts, nil
}

// --- Due Date Management ---

// AddDueDate adds a new due date entry.
func (s *FlashcardService) AddDueDate(dueDate storage.DueDate) error {
	if dueDate.Topic == "" || dueDate.Tag == "" || dueDate.DueDate.IsZero() {
		return errors.New("due date topic, tag, and date are required")
	}
	if err := s.Storage.AddDueDate(dueDate); err != nil {
		return fmt.Errorf("error adding due date to storage: %w", err)
	}
	// Check error on Save
	if err := s.Storage.Save(); err != nil {
		return fmt.Errorf("error saving storage after adding due date: %w", err)
	}
	return nil
}

// ListDueDates retrieves all due date entries.
func (s *FlashcardService) ListDueDates() ([]storage.DueDate, error) {
	return s.Storage.ListDueDates()
}

// UpdateDueDate updates an existing due date entry.
func (s *FlashcardService) UpdateDueDate(dueDate storage.DueDate) error {
	if dueDate.ID == "" {
		return errors.New("due date ID is required for update")
	}
	if err := s.Storage.UpdateDueDate(dueDate); err != nil {
		return fmt.Errorf("error updating due date in storage: %w", err)
	}
	// Check error on Save
	if err := s.Storage.Save(); err != nil {
		return fmt.Errorf("error saving storage after updating due date: %w", err)
	}
	return nil
}

// DeleteDueDate deletes a due date entry by its ID.
func (s *FlashcardService) DeleteDueDate(id string) error {
	if id == "" {
		return errors.New("due date ID is required for delete")
	}
	if err := s.Storage.DeleteDueDate(id); err != nil {
		return fmt.Errorf("error deleting due date from storage: %w", err)
	}
	// Check error on Save
	if err := s.Storage.Save(); err != nil {
		return fmt.Errorf("error saving storage after deleting due date: %w", err)
	}
	return nil
}

// GetCardsByTag retrieves all cards that have a specific tag.
func (s *FlashcardService) GetCardsByTag(tag string) ([]storage.Card, error) {
	if tag == "" {
		return nil, errors.New("tag cannot be empty")
	}
	// Use the ListCards method from storage, passing the single tag in a slice
	matchingCards, err := s.Storage.ListCards([]string{tag})
	if err != nil {
		return nil, fmt.Errorf("error getting cards by tag '%s': %w", tag, err)
	}
	return matchingCards, nil
}

// DueDateProgressStats holds statistics for a specific due date.
type DueDateProgressStats struct {
	TotalCards      int     `json:"total_cards"`
	MasteredCards   int     `json:"mastered_cards"`
	ProgressPercent float64 `json:"progress_percent"`
}

// GetDueDateProgressStats calculates progress for cards associated with a due date tag.
// Mastery is defined as having a last review rating of 4 (Easy).
func (s *FlashcardService) GetDueDateProgressStats(tag string) (DueDateProgressStats, error) {
	stats := DueDateProgressStats{}

	fmt.Printf("GetDueDateProgressStats called for tag: %s\n", tag)

	cards, err := s.GetCardsByTag(tag) // Uses the corrected GetCardsByTag
	if err != nil {
		return stats, fmt.Errorf("error getting cards for tag '%s': %w", tag, err)
	}

	stats.TotalCards = len(cards)
	fmt.Printf("Found %d cards with tag %s\n", stats.TotalCards, tag)

	if stats.TotalCards == 0 {
		return stats, nil // No cards for this tag, progress is 0
	}

	masteredCount := 0
	for i, card := range cards {
		fmt.Printf("Checking card %d: %s\n", i+1, card.ID)
		reviews, err := s.Storage.GetCardReviews(card.ID)
		if err != nil {
			// Log or handle error? For now, skip card if reviews can't be fetched.
			fmt.Printf("Warning: could not get reviews for card %s: %v\n", card.ID, err)
			continue
		}
		fmt.Printf("Card %s has %d reviews\n", card.ID, len(reviews))
		if len(reviews) > 0 {
			// Sort reviews by timestamp descending to get the latest
			sort.Slice(reviews, func(i, j int) bool {
				return reviews[i].Timestamp.After(reviews[j].Timestamp)
			})
			lastReview := reviews[0]
			fmt.Printf("Card %s last review rating: %d\n", card.ID, lastReview.Rating)
			if lastReview.Rating == gofsrs.Easy { // Check if last rating was Easy (4)
				masteredCount++
				fmt.Printf("Card %s counted as mastered\n", card.ID)
			}
		}
	}

	stats.MasteredCards = masteredCount
	stats.ProgressPercent = (float64(masteredCount) / float64(stats.TotalCards)) * 100.0

	fmt.Printf("GetDueDateProgressStats result: %+v\n", stats)

	return stats, nil
}
