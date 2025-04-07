// Package main provides implementation for the flashcards MCP service.
package main

import (
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// Card represents a flashcard with content and FSRS algorithm data
type Card struct {
	ID        string    `json:"id"`
	Front     string    `json:"front"`
	Back      string    `json:"back"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags,omitempty"`
	// Algorithm data - from go-fsrs package which contains:
	// Due, Stability, Difficulty, ElapsedDays, ScheduledDays, Reps, Lapses, State, LastReview
	FSRS gofsrs.Card `json:"fsrs"`
}

// CardStats represents statistics for flashcard review
type CardStats struct {
	TotalCards    int     `json:"total_cards"`
	DueCards      int     `json:"due_cards"`
	ReviewsToday  int     `json:"reviews_today"`
	RetentionRate float64 `json:"retention_rate"`
}

// CardResponse represents the response structure for get_due_card
type CardResponse struct {
	Card  Card      `json:"card"`
	Stats CardStats `json:"stats"`
}

// ReviewResponse represents the response structure for submit_review
type ReviewResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Card    Card   `json:"card,omitempty"`
}

// CreateCardResponse represents the response structure for create_card
type CreateCardResponse struct {
	Card storage.Card `json:"card"`
}

// UpdateCardResponse represents the response structure for update_card
type UpdateCardResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeleteCardResponse represents the response structure for delete_card
type DeleteCardResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListCardsResponse represents the response structure for list_cards
type ListCardsResponse struct {
	Cards []Card    `json:"cards"`
	Stats CardStats `json:"stats,omitempty"`
}
