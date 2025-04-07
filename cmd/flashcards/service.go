// Package main provides implementation for the flashcards MCP service.
package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
	"github.com/danieldreier/mcp-flashcards/internal/storage"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// FlashcardService manages operations for flashcards with storage and FSRS algorithm
type FlashcardService struct {
	Storage     storage.Storage
	FSRSManager fsrs.FSRSManager
}

// NewFlashcardService creates a new FlashcardService
func NewFlashcardService(storage storage.Storage) *FlashcardService {
	return &FlashcardService{
		Storage:     storage,
		FSRSManager: fsrs.NewFSRSManager(),
	}
}

// UpdateCard updates an existing flashcard
// Only updates content fields, preserves FSRS algorithm data
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
	if len(tags) > 0 {
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

	// Convert storage.Card to our Card type
	card := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	return card, nil
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
	// Get all cards from storage
	// If no filter tags, get all cards
	storageCards, err := s.Storage.ListCards(nil)
	if err != nil {
		return nil, CardStats{}, fmt.Errorf("error listing cards: %w", err)
	}

	// If filter tags provided, filter the cards to only include those with ALL specified tags
	filteredCards := storageCards
	if len(filterTags) > 0 {
		filteredCards = []storage.Card{}
		for _, card := range storageCards {
			// Check if card has all the required tags
			hasAllTags := true
			for _, requiredTag := range filterTags {
				tagFound := false
				for _, cardTag := range card.Tags {
					if cardTag == requiredTag {
						tagFound = true
						break
					}
				}
				if !tagFound {
					hasAllTags = false
					break
				}
			}

			// Only include cards that have all the required tags
			if hasAllTags {
				filteredCards = append(filteredCards, card)
			}
		}
	}

	// Convert filtered storage.Card array to our Card type array
	cards := make([]Card, 0, len(filteredCards))
	for _, storageCard := range filteredCards {
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
		stats = s.calculateStats(storageCards) // Use all cards for stats calculation
	}

	return cards, stats, nil
}

// GetDueCard returns the next card due for review with statistics
func (s *FlashcardService) GetDueCard() (Card, CardStats, error) {
	// Get all cards from storage
	cards, err := s.Storage.ListCards(nil)
	if err != nil {
		return Card{}, CardStats{}, fmt.Errorf("error listing cards: %w", err)
	}

	// Current time for priority calculation
	now := time.Now()

	// Find due cards and calculate priority
	var dueCards []struct {
		card     Card
		priority float64
	}

	for _, storageCard := range cards {
		// Convert storage.Card to our Card type
		card := Card{
			ID:        storageCard.ID,
			Front:     storageCard.Front,
			Back:      storageCard.Back,
			CreatedAt: storageCard.CreatedAt,
			Tags:      storageCard.Tags,
			FSRS:      storageCard.FSRS,
		}

		// Consider cards due now or in the past
		if !card.FSRS.Due.After(now) {
			priority := s.FSRSManager.GetReviewPriority(card.FSRS.State, card.FSRS.Due, now)
			dueCards = append(dueCards, struct {
				card     Card
				priority float64
			}{card, priority})
		}
	}

	// Sort by priority (highest first)
	sort.Slice(dueCards, func(i, j int) bool {
		return dueCards[i].priority > dueCards[j].priority
	})

	// Return highest priority card or error if none due
	if len(dueCards) == 0 {
		return Card{}, CardStats{}, fmt.Errorf("no cards due for review")
	}

	// Calculate statistics
	stats := s.calculateStats(cards)

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
	var reviews []storage.Review
	for _, card := range cards {
		cardReviews, err := s.Storage.GetCardReviews(card.ID)
		if err == nil {
			for _, review := range cardReviews {
				if !review.Timestamp.Before(today) {
					reviews = append(reviews, review)
				}
			}
		}
	}

	// Calculate retention rate (correct answers / total reviews)
	correctReviews := 0
	for _, review := range reviews {
		// Rating 3 (Good) or 4 (Easy) is considered correct
		if review.Rating >= gofsrs.Good {
			correctReviews++
		}
	}

	retentionRate := 0.0
	if len(reviews) > 0 {
		retentionRate = float64(correctReviews) / float64(len(reviews)) * 100.0
	}

	return CardStats{
		TotalCards:    totalCards,
		DueCards:      dueCards,
		ReviewsToday:  len(reviews),
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

	// Convert storage.Card to our Card type
	card := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	now := time.Now()

	// Use FSRS manager to schedule the review using the go-fsrs library
	updatedState, newDueDate := s.FSRSManager.ScheduleReview(card.FSRS.State, rating, now)

	// Update the card with new state information
	card.FSRS.State = updatedState
	card.FSRS.Due = newDueDate

	// Save the updated card state back to storage
	storageCard.FSRS = card.FSRS
	if err := s.Storage.UpdateCard(storageCard); err != nil {
		return Card{}, fmt.Errorf("error updating card: %w", err)
	}

	// Add review to storage
	_, err = s.Storage.AddReview(cardID, rating, answer)
	if err != nil {
		return Card{}, fmt.Errorf("error adding review: %w", err)
	}

	// Persist changes to disk
	if err := s.Storage.Save(); err != nil {
		return Card{}, fmt.Errorf("error saving storage: %w", err)
	}

	return card, nil
}
