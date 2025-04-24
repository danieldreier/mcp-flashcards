// Package main provides implementation for the flashcards MCP service.
package main

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/google/uuid"
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

// UpdateCard updates an existing flashcard selectively based on non-nil input pointers.
func (s *FlashcardService) UpdateCard(cardID string, front *string, back *string, tags *[]string) (Card, error) {
	// Get the card from storage
	storageCard, err := s.Storage.GetCard(cardID)
	if err != nil {
		return Card{}, fmt.Errorf("error getting card %s: %w", cardID, err)
	}

	updated := false
	// Update fields only if the corresponding pointer is not nil
	if front != nil {
		if storageCard.Front != *front {
			storageCard.Front = *front
			updated = true
		}
	}
	if back != nil {
		if storageCard.Back != *back {
			storageCard.Back = *back
			updated = true
		}
	}
	if tags != nil {
		// Need to compare slices carefully to see if an update is needed
		if !equalStringSlices(storageCard.Tags, *tags) {
			storageCard.Tags = *tags
			updated = true
		}
	}

	// Only save if changes were actually made
	if updated {
		// Save the updated card back to storage
		if err := s.Storage.UpdateCard(storageCard); err != nil {
			return Card{}, fmt.Errorf("error updating card %s in storage: %w", cardID, err)
		}

		// Persist changes to disk
		if err := s.Storage.Save(); err != nil {
			// Log this error but potentially return the in-memory updated card?
			// For consistency, let's return the error.
			return Card{}, fmt.Errorf("error saving storage after updating card %s: %w", cardID, err)
		}
	}

	// Convert storage.Card back to our main Card type for the response
	responseCard := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	return responseCard, nil
}

// equalStringSlices checks if two string slices are equal (considers order).
// TODO: Move to a utility package or consider sorting before comparison if order doesn't matter.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// DeleteCard deletes a flashcard
func (s *FlashcardService) DeleteCard(cardID string) error {
	fmt.Printf("[DEBUG-SVC-DELETE] Starting DeleteCard for ID %s\n", cardID)
	// Delete the card from storage
	if err := s.Storage.DeleteCard(cardID); err != nil {
		fmt.Printf("[DEBUG-SVC-DELETE] Storage.DeleteCard returned error: %v\n", err)
		return fmt.Errorf("error deleting card: %w", err)
	}
	fmt.Printf("[DEBUG-SVC-DELETE] Card deleted from storage successfully\n")

	// Persist changes to disk
	fmt.Printf("[DEBUG-SVC-DELETE] Now calling Storage.Save()\n")
	if err := s.Storage.Save(); err != nil {
		fmt.Printf("[DEBUG-SVC-DELETE] Storage.Save() returned error: %v\n", err)
		return fmt.Errorf("error saving storage: %w", err)
	}
	fmt.Printf("[DEBUG-SVC-DELETE] Storage.Save() completed successfully\n")

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
	fmt.Printf("[DEBUG-SVC] GetDueCard called with filterTags: %v\n", filterTags)
	// Get all cards from storage first to calculate overall statistics
	allCards, err := s.Storage.ListCards(nil)
	if err != nil {
		fmt.Printf("[DEBUG-SVC] GetDueCard: error listing all cards: %v\n", err)
		return Card{}, CardStats{}, fmt.Errorf("error listing all cards: %w", err)
	}
	fmt.Printf("[DEBUG-SVC] GetDueCard: Found %d total cards in storage.\n", len(allCards))

	// Debug - print all card IDs
	fmt.Printf("[DEBUG-SVC] All card IDs: [")
	for _, card := range allCards {
		fmt.Printf("%s, ", card.ID)
	}
	fmt.Printf("]\n")

	// Calculate overall statistics based on all cards
	stats := s.calculateStats(allCards)

	// If no filter tags were provided, get all cards
	var cardsToConsider []storage.Card
	if len(filterTags) == 0 {
		fmt.Printf("[DEBUG-SVC] GetDueCard: No filter tags provided, considering all %d cards.\n", len(allCards))
		cardsToConsider = allCards
	} else {
		fmt.Printf("[DEBUG-SVC] GetDueCard: Filtering %d cards by tags: %v\n", len(allCards), filterTags)
		// When filter tags are provided, we need to find cards with ALL the specified tags
		for i, card := range allCards {
			matches := hasAllRequiredTags(&card, filterTags)
			fmt.Printf("[DEBUG-SVC] GetDueCard: Checking card %d (ID: %s, Tags: %v) against filter %v -> Matches: %t\n", i, card.ID, card.Tags, filterTags, matches)
			if matches {
				cardsToConsider = append(cardsToConsider, card)
			}
		}
		fmt.Printf("[DEBUG-SVC] GetDueCard: Filtering complete. %d cards matched the tags.\n", len(cardsToConsider))

		// If no cards match the tag filter, return an error
		if len(cardsToConsider) == 0 {
			fmt.Printf("[DEBUG-SVC] GetDueCard: No cards matched tags, returning error.\n")
			return Card{}, stats, fmt.Errorf("no cards found with the specified tags: %v", filterTags)
		}
	}

	// Current time for priority calculation
	now := time.Now()
	fmt.Printf("[DEBUG-SVC] GetDueCard: Finding due cards among %d considered cards.\n", len(cardsToConsider))

	// Find due cards from the filtered list and calculate priority
	var dueCards []struct {
		card     Card
		priority float64
	}

	for _, storageCard := range cardsToConsider { // Iterate over the filtered list
		cardIsDue := !storageCard.FSRS.Due.After(now)
		fmt.Printf("[DEBUG-SVC] GetDueCard: Checking considered card ID %s (Due: %v, IsDue: %t)\n", storageCard.ID, storageCard.FSRS.Due, cardIsDue)
		// Consider cards due now or in the past
		if cardIsDue {
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
			fmt.Printf("[DEBUG-SVC] GetDueCard: Added due card ID %s to list (Priority: %f)\n", card.ID, priority)
		}
	}
	fmt.Printf("[DEBUG-SVC] GetDueCard: Found %d due cards among considered cards.\n", len(dueCards))

	// Sort the due cards (from the filtered list) by priority (highest first)
	sort.Slice(dueCards, func(i, j int) bool {
		return dueCards[i].priority > dueCards[j].priority
	})

	// Return highest priority card from the filtered set or error if none due
	if len(dueCards) == 0 {
		if len(filterTags) > 0 {
			fmt.Printf("[DEBUG-SVC] GetDueCard: No DUE cards matched tags, returning error.\n")
			return Card{}, stats, fmt.Errorf("no cards due for review with the specified tags: %v", filterTags)
		}
		// No filter, but no cards due
		fmt.Printf("[DEBUG-SVC] GetDueCard: No cards are due for review, returning error.\n")
		return Card{}, stats, fmt.Errorf("no cards due for review")
	}

	// Return the highest priority card from the filtered due list, along with overall stats
	fmt.Printf("[DEBUG-SVC] GetDueCard: Returning highest priority card ID %s.\n", dueCards[0].card.ID)
	return dueCards[0].card, stats, nil
}

// Helper function to ensure all required tags are present in a card
func hasAllRequiredTags(card *storage.Card, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true // No required tags means all cards match
	}

	if card == nil {
		return false // Can't match any tags if card is nil
	}

	// If the card has no tags but we have required tags, it can't match
	if len(card.Tags) == 0 {
		return false
	}

	// Create a map of the card's tags for efficient lookup
	cardTagsMap := make(map[string]bool)
	for _, tag := range card.Tags {
		cardTagsMap[tag] = true
	}

	// Check if the card has all required tags
	for _, reqTag := range requiredTags {
		if !cardTagsMap[reqTag] {
			return false // Missing a required tag
		}
	}

	return true // All required tags found
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
	return s.SubmitReviewWithTime(cardID, rating, answer, timeNow())
}

// SubmitReviewWithTime processes a review for a card and updates its state using the FSRS algorithm
// with a specific timestamp. This allows tests to provide a simulated "now" timestamp.
func (s *FlashcardService) SubmitReviewWithTime(cardID string, rating gofsrs.Rating, answer string, now time.Time) (Card, error) {
	startTime := now
	fmt.Printf("[DEBUG-SVC] SubmitReview starting for cardID=%s, rating=%d at %v\n",
		cardID, rating, startTime.Format(time.RFC3339Nano))

	// Get the card from storage
	fmt.Printf("[DEBUG-SVC] Retrieving card from storage\n")
	storageCard, err := s.Storage.GetCard(cardID)
	if err != nil {
		fmt.Printf("[DEBUG-SVC] Error getting card: %v\n", err)
		return Card{}, fmt.Errorf("error getting card: %w", err)
	}
	fmt.Printf("[DEBUG-SVC] Retrieved card with current state=%v, due=%v\n",
		storageCard.FSRS.State, storageCard.FSRS.Due)

	// Get previous reviews to calculate actual elapsed time
	fmt.Printf("[DEBUG-SVC] Retrieving previous reviews for cardID=%s\n", cardID)
	previousReviews, err := s.Storage.GetCardReviews(cardID)
	if err != nil {
		fmt.Printf("[DEBUG-SVC] Error getting reviews: %v\n", err)
		// Don't fail the operation, just continue with default elapsed days
	}
	fmt.Printf("[DEBUG-SVC] Found %d previous reviews for card %s\n", len(previousReviews), cardID)

	// Calculate elapsed days since last review if we have review history
	if len(previousReviews) > 0 {
		// Sort reviews by timestamp (newest first)
		sort.Slice(previousReviews, func(i, j int) bool {
			return previousReviews[i].Timestamp.After(previousReviews[j].Timestamp)
		})

		// Get the most recent review
		lastReviewTime := previousReviews[0].Timestamp

		// Calculate elapsed days
		elapsedDuration := now.Sub(lastReviewTime)
		elapsedDays := uint64(elapsedDuration.Hours() / 24.0)

		// Update the ElapsedDays in the card's FSRS state
		storageCard.FSRS.ElapsedDays = elapsedDays

		fmt.Printf("[DEBUG-SVC] Last review at %v, now at %v, elapsed days: %d\n",
			lastReviewTime.Format(time.RFC3339), now.Format(time.RFC3339), elapsedDays)
	}

	fmt.Printf("[DEBUG-SVC] Calling GetSchedulingInfo with ElapsedDays=%d\n",
		storageCard.FSRS.ElapsedDays)

	// Get the complete updated FSRS card with all metadata using the new method
	updatedFSRSCard := s.FSRSManager.GetSchedulingInfo(
		storageCard.FSRS, // Pass the entire FSRS card with updated ElapsedDays
		rating,
		now,
	)
	fmt.Printf("[DEBUG-SVC] FSRS scheduling result: newState=%v, newDueDate=%v, stability=%.4f, difficulty=%.4f, reps=%d\n",
		updatedFSRSCard.State, updatedFSRSCard.Due, updatedFSRSCard.Stability, updatedFSRSCard.Difficulty, updatedFSRSCard.Reps)

	// Update the storage card with the complete FSRS data
	fmt.Printf("[DEBUG-SVC] Updating card with complete FSRS state\n")
	storageCard.FSRS = updatedFSRSCard // Replace entire FSRS card with updated version
	storageCard.LastReviewedAt = now   // Record last reviewed time (field should exist now)

	// Save the updated card state back to storage
	fmt.Printf("[DEBUG-SVC] Updating card in storage at %v\n", timeNow().Format(time.RFC3339Nano))
	if err := s.Storage.UpdateCard(storageCard); err != nil {
		fmt.Printf("[DEBUG-SVC] Error updating card: %v\n", err)
		return Card{}, fmt.Errorf("error updating card: %w", err)
	}

	// Add review to storage
	fmt.Printf("[DEBUG-SVC] Adding review to storage at %v\n", timeNow().Format(time.RFC3339Nano))
	reviewLog := storage.Review{
		ID:            uuid.New().String(),
		CardID:        cardID,
		Rating:        rating,
		Timestamp:     now, // Use the provided time for consistency
		Answer:        answer,
		ScheduledDays: updatedFSRSCard.ScheduledDays,
		ElapsedDays:   updatedFSRSCard.ElapsedDays,
		State:         updatedFSRSCard.State,
	}

	if err := s.Storage.AddReviewDirect(reviewLog); err != nil {
		fmt.Printf("[DEBUG-SVC] Error adding review: %v\n", err)
		return Card{}, fmt.Errorf("error adding review: %w", err)
	}
	fmt.Printf("[DEBUG-SVC] Review added successfully\n")

	// Persist changes to disk
	fmt.Printf("[DEBUG-SVC] Saving storage to disk at %v\n", timeNow().Format(time.RFC3339Nano))
	if err := s.Storage.Save(); err != nil {
		fmt.Printf("[DEBUG-SVC] Error saving storage: %v\n", err)
		return Card{}, fmt.Errorf("error saving storage: %w", err)
	}
	fmt.Printf("[DEBUG-SVC] Storage saved successfully\n")

	// Convert updated storage.Card to our main Card type
	updatedCard := Card{
		ID:        storageCard.ID,
		Front:     storageCard.Front,
		Back:      storageCard.Back,
		CreatedAt: storageCard.CreatedAt,
		Tags:      storageCard.Tags,
		FSRS:      storageCard.FSRS,
	}

	elapsed := time.Since(startTime)
	fmt.Printf("[DEBUG-SVC] SubmitReview completed in %v at %v\n",
		elapsed, timeNow().Format(time.RFC3339Nano))

	return updatedCard, nil
}

// Variable to allow mocking time.Now in tests
var timeNow = time.Now

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

	// fmt.Printf("GetDueDateProgressStats called for tag: %s\n", tag)

	cards, err := s.GetCardsByTag(tag) // Uses the corrected GetCardsByTag
	if err != nil {
		return stats, fmt.Errorf("error getting cards for tag '%s': %w", tag, err)
	}

	stats.TotalCards = len(cards)
	// fmt.Printf("Found %d cards with tag %s\n", stats.TotalCards, tag)

	if stats.TotalCards == 0 {
		return stats, nil // No cards for this tag, progress is 0
	}

	masteredCount := 0
	for _, card := range cards {
		// fmt.Printf("Checking card %d: %s\n", i+1, card.ID)
		reviews, err := s.Storage.GetCardReviews(card.ID)
		if err != nil {
			// Log or handle error? For now, skip card if reviews can't be fetched.
			// fmt.Printf("Warning: could not get reviews for card %s: %v\n", card.ID, err)
			continue
		}
		// fmt.Printf("Card %s has %d reviews\n", card.ID, len(reviews))
		if len(reviews) > 0 {
			// Sort reviews by timestamp descending to get the latest
			sort.Slice(reviews, func(i, j int) bool {
				return reviews[i].Timestamp.After(reviews[j].Timestamp)
			})
			lastReview := reviews[0]
			// fmt.Printf("Card %s last review rating: %d\n", card.ID, lastReview.Rating)
			if lastReview.Rating == gofsrs.Easy { // Check if last rating was Easy (4)
				masteredCount++
				// fmt.Printf("Card %s counted as mastered\n", card.ID)
			}
		}
	}

	stats.MasteredCards = masteredCount
	stats.ProgressPercent = (float64(masteredCount) / float64(stats.TotalCards)) * 100.0

	// fmt.Printf("GetDueDateProgressStats result: %+v\n", stats)

	return stats, nil
}
