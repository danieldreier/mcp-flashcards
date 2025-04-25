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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// FlashcardService manages operations for flashcards with storage and FSRS algorithm
type FlashcardService struct {
	Storage     storage.Storage // Interface for storage operations
	FSRSManager fsrs.FSRSManager
	Logger      *zap.Logger
}

// NewFlashcardService creates a new FlashcardService
func NewFlashcardService(storage storage.Storage) *FlashcardService {
	// Initialize Zap logger
	logConfig := zap.NewDevelopmentConfig() // Use development config for human-readable output
	// Customize encoder if needed (e.g., time format, level encoding)
	// logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// Set log level (e.g., Debug, Info, Warn, Error)
	logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel) // Default to Debug, can be configured

	logger, err := logConfig.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		// Fallback to standard logger if zap fails (shouldn't normally happen)
		fmt.Printf("Error initializing zap logger: %v. Falling back to stdlib log.\n", err)
		return &FlashcardService{
			Storage:     storage,
			FSRSManager: fsrs.NewFSRSManager(),
			Logger:      zap.NewNop(), // Use Nop logger as fallback
		}
	}

	// Optionally, sync logger on shutdown (e.g., in a Close method or via defer in main)
	// defer logger.Sync()

	return &FlashcardService{
		Storage:     storage,
		FSRSManager: fsrs.NewFSRSManager(),
		Logger:      logger,
	}
}

// CreateCard creates a new flashcard using the Storage layer
func (s *FlashcardService) CreateCard(front, back string, tags []string) (Card, error) {
	s.Logger.Debug("Service CreateCard called", zap.String("front", front), zap.String("back", back), zap.Strings("tags", tags))
	// Delegate creation to the storage layer, which handles FSRS initialization
	storageCard, err := s.Storage.CreateCard(front, back, tags)
	if err != nil {
		s.Logger.Error("Error creating card in storage", zap.Error(err))
		return Card{}, fmt.Errorf("error creating card in storage: %w", err)
	}
	s.Logger.Debug("Card created in storage layer", zap.String("card_id", storageCard.ID))

	// Persist changes to disk (Save should ideally be part of the storage method)
	// Assuming storage methods don't auto-save for now.
	if err := s.Storage.Save(); err != nil {
		// Attempt to rollback? Difficult. Log error.
		s.Logger.Warn("Failed to save storage after creating card, but card exists in memory", zap.String("card_id", storageCard.ID), zap.Error(err))
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
	s.Logger.Debug("Starting DeleteCard", zap.String("card_id", cardID))
	// Delete the card from storage
	if err := s.Storage.DeleteCard(cardID); err != nil {
		s.Logger.Error("Storage.DeleteCard returned error", zap.String("card_id", cardID), zap.Error(err))
		return fmt.Errorf("error deleting card: %w", err)
	}
	s.Logger.Debug("Card deleted from storage successfully", zap.String("card_id", cardID))

	// Persist changes to disk
	s.Logger.Debug("Calling Storage.Save() after delete", zap.String("card_id", cardID))
	if err := s.Storage.Save(); err != nil {
		s.Logger.Error("Storage.Save() returned error after delete", zap.String("card_id", cardID), zap.Error(err))
		return fmt.Errorf("error saving storage: %w", err)
	}
	s.Logger.Debug("Storage.Save() completed successfully after delete", zap.String("card_id", cardID))

	return nil
}

// ListCards lists all flashcards, optionally filtered by tags
func (s *FlashcardService) ListCards(filterTags []string, includeStats bool) ([]Card, CardStats, error) {
	s.Logger.Debug("Service ListCards called", zap.Strings("filterTags", filterTags), zap.Bool("includeStats", includeStats))
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
			s.Logger.Warn("Error getting all cards for stats calculation", zap.Error(err))
			stats = CardStats{TotalCards: len(storageCards)} // Use filtered count as fallback?
		} else {
			stats = s.calculateStats(allStorageCards)
		}
	}

	return cards, stats, nil
}

// GetDueCard returns the next card due for review with statistics, optionally filtered by tags
func (s *FlashcardService) GetDueCard(filterTags []string) (Card, CardStats, error) {
	s.Logger.Debug("GetDueCard called", zap.Strings("filterTags", filterTags))
	// Get all cards from storage first to calculate overall statistics
	allCards, err := s.Storage.ListCards(nil)
	if err != nil {
		s.Logger.Error("GetDueCard: error listing all cards", zap.Error(err))
		return Card{}, CardStats{}, fmt.Errorf("error listing all cards: %w", err)
	}
	s.Logger.Debug("GetDueCard: Found total cards in storage.", zap.Int("count", len(allCards)))

	// Debug - print all card IDs
	if len(allCards) > 0 {
		ids := make([]string, len(allCards))
		for i, card := range allCards {
			ids[i] = card.ID
		}
		s.Logger.Debug("All card IDs in storage", zap.Strings("ids", ids))
	}

	// Calculate overall statistics based on all cards
	stats := s.calculateStats(allCards)

	// If no filter tags were provided, get all cards
	var cardsToConsider []storage.Card
	if len(filterTags) == 0 {
		s.Logger.Debug("GetDueCard: No filter tags provided, considering all cards.", zap.Int("count", len(allCards)))
		cardsToConsider = allCards
	} else {
		s.Logger.Debug("GetDueCard: Filtering cards by tags", zap.Int("card_count", len(allCards)), zap.Strings("filterTags", filterTags))
		// When filter tags are provided, we need to find cards with ALL the specified tags
		for i, card := range allCards {
			matches := hasAllRequiredTags(&card, filterTags)
			s.Logger.Debug("GetDueCard: Checking card against tag filter",
				zap.Int("index", i),
				zap.String("card_id", card.ID),
				zap.Strings("card_tags", card.Tags),
				zap.Strings("filter_tags", filterTags),
				zap.Bool("matches", matches))
			if matches {
				cardsToConsider = append(cardsToConsider, card)
			}
		}
		s.Logger.Debug("GetDueCard: Filtering complete.", zap.Int("matched_count", len(cardsToConsider)))

		// If no cards match the tag filter, return an error
		if len(cardsToConsider) == 0 {
			s.Logger.Debug("GetDueCard: No cards matched tags, returning error.")
			return Card{}, stats, fmt.Errorf("no cards found with the specified tags: %v", filterTags)
		}
	}

	// Current time for priority calculation
	now := time.Now()
	s.Logger.Debug("GetDueCard: Finding due cards", zap.Int("considered_count", len(cardsToConsider)))

	// Find due cards from the filtered list and calculate priority
	var dueCards []struct {
		card     Card
		priority float64
	}

	for _, storageCard := range cardsToConsider { // Iterate over the filtered list
		cardIsDue := !storageCard.FSRS.Due.After(now)
		s.Logger.Debug("GetDueCard: Checking considered card due status",
			zap.String("card_id", storageCard.ID),
			zap.Time("due", storageCard.FSRS.Due),
			zap.Bool("is_due", cardIsDue))
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
			s.Logger.Debug("GetDueCard: Added due card to list", zap.String("card_id", card.ID), zap.Float64("priority", priority))
		}
	}
	s.Logger.Debug("GetDueCard: Found due cards among considered cards.", zap.Int("due_count", len(dueCards)))

	// Sort the due cards (from the filtered list) by priority (highest first)
	sort.Slice(dueCards, func(i, j int) bool {
		return dueCards[i].priority > dueCards[j].priority
	})

	// Return highest priority card from the filtered set or error if none due
	if len(dueCards) == 0 {
		if len(filterTags) > 0 {
			s.Logger.Debug("GetDueCard: No DUE cards matched tags, returning error.")
			return Card{}, stats, fmt.Errorf("no cards due for review with the specified tags: %v", filterTags)
		}
		// No filter, but no cards due
		s.Logger.Debug("GetDueCard: No cards are due for review, returning error.")
		return Card{}, stats, fmt.Errorf("no cards due for review")
	}

	// Return the highest priority card from the filtered due list, along with overall stats
	s.Logger.Debug("GetDueCard: Returning highest priority card", zap.String("card_id", dueCards[0].card.ID))
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
	s.Logger.Debug("SubmitReview starting",
		zap.String("card_id", cardID),
		zap.Int("rating", int(rating)),
		zap.Time("start_time", startTime))

	// Get the card from storage
	s.Logger.Debug("Retrieving card from storage", zap.String("card_id", cardID))
	storageCard, err := s.Storage.GetCard(cardID)
	if err != nil {
		s.Logger.Error("Error getting card", zap.String("card_id", cardID), zap.Error(err))
		return Card{}, fmt.Errorf("error getting card: %w", err)
	}
	s.Logger.Debug("Retrieved card from storage",
		zap.String("card_id", cardID),
		zap.Int("current_state", int(storageCard.FSRS.State)),
		zap.Time("current_due", storageCard.FSRS.Due))

	// Get previous reviews to calculate actual elapsed time
	s.Logger.Debug("Retrieving previous reviews", zap.String("card_id", cardID))
	previousReviews, err := s.Storage.GetCardReviews(cardID)
	if err != nil {
		s.Logger.Warn("Error getting reviews, proceeding with default elapsed days", zap.String("card_id", cardID), zap.Error(err))
		// Don't fail the operation, just continue with default elapsed days
	}
	s.Logger.Debug("Found previous reviews", zap.String("card_id", cardID), zap.Int("review_count", len(previousReviews)))

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

		s.Logger.Debug("Calculated elapsed days",
			zap.String("card_id", cardID),
			zap.Time("last_review_time", lastReviewTime),
			zap.Time("now", now),
			zap.Uint64("elapsed_days", elapsedDays))
	}

	s.Logger.Debug("Calling GetSchedulingInfo",
		zap.String("card_id", cardID),
		zap.Uint64("fsrs_elapsed_days", storageCard.FSRS.ElapsedDays))

	// Get the complete updated FSRS card with all metadata using the new method
	updatedFSRSCard := s.FSRSManager.GetSchedulingInfo(
		storageCard.FSRS, // Pass the entire FSRS card with updated ElapsedDays
		rating,
		now,
	)
	s.Logger.Debug("FSRS scheduling result",
		zap.String("card_id", cardID),
		zap.Int("new_state", int(updatedFSRSCard.State)),
		zap.Time("new_due_date", updatedFSRSCard.Due),
		zap.Float64("stability", updatedFSRSCard.Stability),
		zap.Float64("difficulty", updatedFSRSCard.Difficulty),
		zap.Uint64("reps", updatedFSRSCard.Reps))

	// Update the storage card with the complete FSRS data
	s.Logger.Debug("Updating card in memory with complete FSRS state", zap.String("card_id", cardID))
	storageCard.FSRS = updatedFSRSCard // Replace entire FSRS card with updated version
	storageCard.LastReviewedAt = now   // Record last reviewed time (field should exist now)

	// Save the updated card state back to storage
	s.Logger.Debug("Updating card in storage", zap.String("card_id", cardID), zap.Time("timestamp", timeNow()))
	if err := s.Storage.UpdateCard(storageCard); err != nil {
		s.Logger.Error("Error updating card in storage", zap.String("card_id", cardID), zap.Error(err))
		return Card{}, fmt.Errorf("error updating card: %w", err)
	}

	// Add review to storage
	s.Logger.Debug("Adding review to storage", zap.String("card_id", cardID), zap.Time("timestamp", timeNow()))
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
		s.Logger.Error("Error adding review to storage", zap.String("card_id", cardID), zap.Error(err))
		return Card{}, fmt.Errorf("error adding review: %w", err)
	}
	s.Logger.Debug("Review added to storage successfully", zap.String("card_id", cardID))

	// Persist changes to disk
	s.Logger.Debug("Saving storage to disk", zap.String("card_id", cardID), zap.Time("timestamp", timeNow()))
	if err := s.Storage.Save(); err != nil {
		s.Logger.Error("Error saving storage", zap.String("card_id", cardID), zap.Error(err))
		return Card{}, fmt.Errorf("error saving storage: %w", err)
	}
	s.Logger.Debug("Storage saved successfully", zap.String("card_id", cardID))

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
	s.Logger.Debug("SubmitReview completed",
		zap.String("card_id", cardID),
		zap.Duration("elapsed", elapsed),
		zap.Time("end_time", timeNow()))

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

	s.Logger.Debug("GetDueDateProgressStats called", zap.String("tag", tag))

	cards, err := s.GetCardsByTag(tag) // Uses the corrected GetCardsByTag
	if err != nil {
		s.Logger.Error("Error getting cards by tag", zap.String("tag", tag), zap.Error(err))
		return stats, fmt.Errorf("error getting cards for tag '%s': %w", tag, err)
	}

	stats.TotalCards = len(cards)
	s.Logger.Debug("Found cards with tag", zap.String("tag", tag), zap.Int("count", stats.TotalCards))

	if stats.TotalCards == 0 {
		return stats, nil // No cards for this tag, progress is 0
	}

	masteredCount := 0
	for i, card := range cards {
		s.Logger.Debug("Checking card for mastery", zap.Int("index", i+1), zap.String("card_id", card.ID), zap.String("tag", tag))
		reviews, err := s.Storage.GetCardReviews(card.ID)
		if err != nil {
			// Log or handle error? For now, skip card if reviews can't be fetched.
			s.Logger.Warn("Could not get reviews for card", zap.String("card_id", card.ID), zap.Error(err))
			continue
		}
		s.Logger.Debug("Card review count", zap.String("card_id", card.ID), zap.Int("review_count", len(reviews)))
		if len(reviews) > 0 {
			// Sort reviews by timestamp descending to get the latest
			sort.Slice(reviews, func(i, j int) bool {
				return reviews[i].Timestamp.After(reviews[j].Timestamp)
			})
			lastReview := reviews[0]
			s.Logger.Debug("Card last review details", zap.String("card_id", card.ID), zap.Int("rating", int(lastReview.Rating)), zap.Time("timestamp", lastReview.Timestamp))
			if lastReview.Rating == gofsrs.Easy { // Check if last rating was Easy (4)
				masteredCount++
				s.Logger.Debug("Card counted as mastered", zap.String("card_id", card.ID))
			}
		}
	}

	stats.MasteredCards = masteredCount
	stats.ProgressPercent = (float64(masteredCount) / float64(stats.TotalCards)) * 100.0

	s.Logger.Debug("GetDueDateProgressStats result", zap.String("tag", tag), zap.Any("stats", stats))

	return stats, nil
}
