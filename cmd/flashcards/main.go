package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/open-spaced-repetition/go-fsrs"
)

// Card represents a flashcard with content and FSRS algorithm data
type Card struct {
	ID        string    `json:"id"`
	Front     string    `json:"front"`
	Back      string    `json:"back"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags,omitempty"`
	// Algorithm data - from fsrs.Card which contains:
	// Due, Stability, Difficulty, ElapsedDays, ScheduledDays, Reps, Lapses, State, LastReview
	FSRS fsrs.Card `json:"fsrs"`
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

// handleGetDueCard handles the get_due_card tool request
func handleGetDueCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create a new FSRS card
	fsrsCard := fsrs.NewCard()
	fsrsCard.Due = time.Now().Add(-1 * time.Hour) // Due 1 hour ago

	// Create a hardcoded response
	response := CardResponse{
		Card: Card{
			ID:        "card1",
			Front:     "What is the capital of France?",
			Back:      "Paris",
			CreatedAt: time.Now().Add(-24 * time.Hour), // Created yesterday
			Tags:      []string{"geography", "europe"},
			FSRS:      fsrsCard,
		},
		Stats: CardStats{
			TotalCards:    10,
			DueCards:      3,
			ReviewsToday:  2,
			RetentionRate: 0.85,
		},
	}

	// We need to return the response as a structured result
	// For now, we'll convert the response to a simple text result
	// which is better supported across different MCP clients
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	// Return using the helper method for text results
	result := mcp.NewToolResultText(string(jsonBytes))
	return result, nil
}

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Flashcards MCP",
		"1.0.0",
		server.WithResourceCapabilities(true, true), // Resource capabilities for subscribe and listChanged
		server.WithLogging(),                        // Enable logging for the server
	)

	// Define the get_due_card tool
	getDueCardTool := mcp.NewTool("get_due_card",
		mcp.WithDescription("Get the next flashcard due for review"),
		// No parameters required for now
	)

	// Register the tool with the handler
	s.AddTool(getDueCardTool, handleGetDueCard)

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Error serving MCP server: %v", err)
	}
}
