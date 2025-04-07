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

// ReviewResponse represents the response structure for submit_review
type ReviewResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CreateCardResponse represents the response structure for create_card
type CreateCardResponse struct {
	Card Card `json:"card"`
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

// handleSubmitReview handles the submit_review tool request
func handleSubmitReview(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract required parameters
	cardID, ok := request.Params.Arguments["card_id"].(string)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: card_id"), nil
	}

	ratingFloat, ok := request.Params.Arguments["rating"].(float64)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: rating"), nil
	}

	rating := int(ratingFloat)
	if rating < 1 || rating > 4 {
		return mcp.NewToolResultText("Rating must be between 1 and 4"), nil
	}

	// Extract optional parameter
	answer, _ := request.Params.Arguments["answer"].(string)

	// Create a hardcoded response
	response := ReviewResponse{
		Success: true,
		Message: "Review submitted successfully for card " + cardID,
	}

	// Include used parameters in log for debugging
	log.Printf("Submitted review for card %s with rating %d and answer '%s'", cardID, rating, answer)

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleCreateCard handles the create_card tool request
func handleCreateCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract required parameters
	front, ok := request.Params.Arguments["front"].(string)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: front"), nil
	}

	back, ok := request.Params.Arguments["back"].(string)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: back"), nil
	}

	// Extract optional parameter (tags)
	tags := []string{}
	if tagsInterface, ok := request.Params.Arguments["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Create a new FSRS card
	fsrsCard := fsrs.NewCard()

	// Create a hardcoded response with the provided front/back but hardcoded ID
	newCard := Card{
		ID:        "new-card-123",
		Front:     front,
		Back:      back,
		CreatedAt: time.Now(),
		Tags:      tags,
		FSRS:      fsrsCard,
	}

	response := CreateCardResponse{
		Card: newCard,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleUpdateCard handles the update_card tool request
func handleUpdateCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract required parameter
	cardID, ok := request.Params.Arguments["card_id"].(string)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: card_id"), nil
	}

	// Extract optional parameters
	front, _ := request.Params.Arguments["front"].(string)
	back, _ := request.Params.Arguments["back"].(string)

	// Extract optional tags parameter
	tags := []string{}
	if tagsInterface, ok := request.Params.Arguments["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Log what would be updated
	log.Printf("Updating card %s - Front: '%s', Back: '%s', Tags: %v", cardID, front, back, tags)

	// Create a hardcoded response
	response := UpdateCardResponse{
		Success: true,
		Message: "Card " + cardID + " updated successfully",
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleDeleteCard handles the delete_card tool request
func handleDeleteCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract required parameter
	cardID, ok := request.Params.Arguments["card_id"].(string)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: card_id"), nil
	}

	// Log what would be deleted
	log.Printf("Deleting card %s", cardID)

	// Create a hardcoded response
	response := DeleteCardResponse{
		Success: true,
		Message: "Card " + cardID + " deleted successfully",
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleListCards handles the list_cards tool request
func handleListCards(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract optional parameters
	var filterTags []string
	if tagsInterface, ok := request.Params.Arguments["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				filterTags = append(filterTags, tagStr)
			}
		}
	}

	includeStats, _ := request.Params.Arguments["include_stats"].(bool)

	// Create some hardcoded cards
	card1 := Card{
		ID:        "card1",
		Front:     "What is the capital of France?",
		Back:      "Paris",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		Tags:      []string{"geography", "europe"},
		FSRS:      fsrs.NewCard(),
	}

	card2 := Card{
		ID:        "card2",
		Front:     "What is the capital of Japan?",
		Back:      "Tokyo",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		Tags:      []string{"geography", "asia"},
		FSRS:      fsrs.NewCard(),
	}

	card3 := Card{
		ID:        "card3",
		Front:     "What is the capital of Brazil?",
		Back:      "Brasília",
		CreatedAt: time.Now().Add(-72 * time.Hour),
		Tags:      []string{"geography", "south-america"},
		FSRS:      fsrs.NewCard(),
	}

	// Create a list of cards (for now, hardcoded)
	allCards := []Card{card1, card2, card3}

	// Filter cards by tags if provided
	filteredCards := allCards
	if len(filterTags) > 0 {
		filteredCards = []Card{}
		for _, card := range allCards {
			// Check if the card has at least one of the filter tags
			hasTag := false
			for _, cardTag := range card.Tags {
				for _, filterTag := range filterTags {
					if cardTag == filterTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if hasTag {
				filteredCards = append(filteredCards, card)
			}
		}
	}

	// Create response
	response := ListCardsResponse{
		Cards: filteredCards,
	}

	// Include stats if requested
	if includeStats {
		response.Stats = CardStats{
			TotalCards:    len(allCards),
			DueCards:      2,
			ReviewsToday:  5,
			RetentionRate: 0.85,
		}
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
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

	// Define the submit_review tool
	submitReviewTool := mcp.NewTool("submit_review",
		mcp.WithDescription("Submit a review for a flashcard"),
		// Define parameters
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the card being reviewed"),
		),
		mcp.WithNumber("rating",
			mcp.Required(),
			mcp.Description("Rating from 1-4: Again=1, Hard=2, Good=3, Easy=4"),
		),
		mcp.WithString("answer",
			mcp.Description("The answer provided by the user"),
		),
	)

	// Define the create_card tool
	createCardTool := mcp.NewTool("create_card",
		mcp.WithDescription("Create a new flashcard"),
		// Define parameters
		mcp.WithString("front",
			mcp.Required(),
			mcp.Description("The front text of the card"),
		),
		mcp.WithString("back",
			mcp.Required(),
			mcp.Description("The back text of the card"),
		),
		mcp.WithArray("tags",
			mcp.Description("Tags for categorizing the card"),
		),
	)

	// Define the update_card tool
	updateCardTool := mcp.NewTool("update_card",
		mcp.WithDescription("Update an existing flashcard"),
		// Define parameters
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the card to update"),
		),
		mcp.WithString("front",
			mcp.Description("The new front text of the card"),
		),
		mcp.WithString("back",
			mcp.Description("The new back text of the card"),
		),
		mcp.WithArray("tags",
			mcp.Description("New tags for the card"),
		),
	)

	// Define the delete_card tool
	deleteCardTool := mcp.NewTool("delete_card",
		mcp.WithDescription("Delete a flashcard"),
		// Define parameters
		mcp.WithString("card_id",
			mcp.Required(),
			mcp.Description("The ID of the card to delete"),
		),
	)

	// Define the list_cards tool
	listCardsTool := mcp.NewTool("list_cards",
		mcp.WithDescription("List all flashcards, optionally filtered by tags"),
		// Define parameters
		mcp.WithArray("tags",
			mcp.Description("Filter cards by tags"),
		),
		mcp.WithBoolean("include_stats",
			mcp.Description("Include statistics in the response"),
		),
	)

	// Register all tools with their handlers
	s.AddTool(getDueCardTool, handleGetDueCard)
	s.AddTool(submitReviewTool, handleSubmitReview)
	s.AddTool(createCardTool, handleCreateCard)
	s.AddTool(updateCardTool, handleUpdateCard)
	s.AddTool(deleteCardTool, handleDeleteCard)
	s.AddTool(listCardsTool, handleListCards)

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Error serving MCP server: %v", err)
	}
}
