package fsrs

import (
	"time"

	"github.com/open-spaced-repetition/go-fsrs"
)

// FSRSManager defines the interface for scheduling flashcards using the FSRS algorithm
type FSRSManager interface {
	// ScheduleReview calculates the next review time based on the rating
	// From go-fsrs documentation:
	// type Rating int8
	// const Again Rating = iota + 1 // (1)
	// const Hard, Good, Easy Rating = 2, 3, 4
	//
	// type State int8
	// const New State = iota // (0)
	// const Learning, Review, Relearning State = 1, 2, 3
	ScheduleReview(currentCard fsrs.Card, rating fsrs.Rating, now time.Time) (fsrs.State, time.Time)

	// GetSchedulingInfo returns the complete card info after applying a rating
	// This includes all FSRS metadata fields needed for accurate scheduling
	GetSchedulingInfo(currentCard fsrs.Card, rating fsrs.Rating, now time.Time) fsrs.Card

	// GetReviewPriority calculates a priority score for a card (for sorting)
	GetReviewPriority(state fsrs.State, due time.Time, now time.Time) float64
}

// FSRSManagerImpl implements the FSRSManager interface
type FSRSManagerImpl struct {
	parameters fsrs.Parameters // Using Parameters from go-fsrs
}

// NewFSRSManager creates a new FSRS manager with default parameters
func NewFSRSManager() FSRSManager {
	return &FSRSManagerImpl{
		parameters: fsrs.DefaultParam(), // Using DefaultParam() from go-fsrs
	}
}

// NewFSRSManagerWithParams creates a new FSRS manager with custom parameters
func NewFSRSManagerWithParams(params fsrs.Parameters) FSRSManager {
	return &FSRSManagerImpl{
		parameters: params,
	}
}

// ScheduleReview implements the FSRSManager interface
func (f *FSRSManagerImpl) ScheduleReview(currentCard fsrs.Card, rating fsrs.Rating, now time.Time) (fsrs.State, time.Time) {
	// Use the Repeat method from the go-fsrs library to calculate next schedule
	// This properly implements the FSRS algorithm instead of manual calculations
	schedulingInfos := f.parameters.Repeat(currentCard, now)

	// Get the scheduling info for the provided rating
	schedulingInfo := schedulingInfos[rating]

	// Extract the updated state and due date from the scheduling information
	return schedulingInfo.Card.State, schedulingInfo.Card.Due
}

// GetSchedulingInfo implements the FSRSManager interface
func (f *FSRSManagerImpl) GetSchedulingInfo(currentCard fsrs.Card, rating fsrs.Rating, now time.Time) fsrs.Card {
	// Use the Repeat method from the go-fsrs library to calculate next schedule
	schedulingInfos := f.parameters.Repeat(currentCard, now)

	// Get the scheduling info for the provided rating
	schedulingInfo := schedulingInfos[rating]

	// Return the complete updated card with all metadata fields
	return schedulingInfo.Card
}

// GetReviewPriority calculates a priority score for a card
// Higher priority means the card should be reviewed sooner
// The priority is based on:
// 1. Overdue cards have higher priority (multiplier based on how overdue)
// 2. Cards in learning/relearning states have higher priority than review
// 3. New cards have lowest priority unless explicitly boosted
func (f *FSRSManagerImpl) GetReviewPriority(state fsrs.State, due time.Time, now time.Time) float64 {
	// Base priority by state (higher for learning states)
	var basePriority float64
	switch state {
	case fsrs.New:
		basePriority = 1.0 // Lowest priority for new cards
	case fsrs.Learning:
		basePriority = 3.0 // High priority for cards in learning
	case fsrs.Relearning:
		basePriority = 3.0 // High priority for cards being relearned
	case fsrs.Review:
		basePriority = 2.0 // Medium priority for review cards
	}

	// Calculate how overdue the card is (in days)
	overdueDays := now.Sub(due).Hours() / 24.0

	// For cards that are due or overdue
	if overdueDays >= 0 {
		// Overdue multiplier: gradually increases priority for overdue cards
		// We use a square root function to prevent extremely overdue cards from
		// completely dominating the queue
		overdueFactor := 1.0 + (overdueDays * 0.1)
		return basePriority * overdueFactor
	}

	// For cards that are not yet due, priority decreases the further in the future
	// they are due. This ensures cards due sooner have higher priority.
	daysToDue := -overdueDays // convert to positive
	return basePriority / (1.0 + daysToDue)
}
