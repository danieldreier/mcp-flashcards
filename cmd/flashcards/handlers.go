// Package main provides implementation for the flashcards MCP service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// handleGetDueCard handles the get_due_card tool request by retrieving the next flashcard
// due for review from the flashcard service.
// It returns the card along with current review statistics.
// If no cards are due, it returns a friendly error message.
func handleGetDueCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// Call service method to get due card
	card, stats, err := s.GetDueCard()
	if err != nil {
		// If no cards are due, return a friendly message
		if err.Error() == "no cards due for review" {
			return mcp.NewToolResultText(`{"error": "No cards due for review"}`), nil
		}
		// For other errors, return the error message
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Error getting due card: %v"}`, err)), nil
	}

	// Create response
	response := CardResponse{
		Card:  card,
		Stats: stats,
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	// Return as text result
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleSubmitReview handles the submit_review tool request by processing a review
// for a flashcard with the given rating (1-4) and optional answer text.
// It updates the card's FSRS scheduling data based on the review result.
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

	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// Convert rating to fsrs.Rating
	fsrsRating := gofsrs.Rating(rating)

	// Call service method to submit review
	updatedCard, err := s.SubmitReview(cardID, fsrsRating, answer)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Error submitting review: %v"}`, err)), nil
	}

	// Create response
	response := ReviewResponse{
		Success: true,
		Message: "Review submitted successfully for card " + cardID,
		Card:    updatedCard,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleCreateCard handles the create_card tool request by creating a new flashcard
// with the provided front and back content and optional tags.
// It also supports setting an optional hour_offset for the due date (for testing purposes).
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

	// Get the storage from server context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// Create the card in storage
	newCard, err := s.Storage.CreateCard(front, back, tags)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error creating card: %v", err)), nil
	}

	// Check for optional hour_offset parameter (for testing only)
	if hourOffsetFloat, ok := request.Params.Arguments["hour_offset"].(float64); ok {
		// Set due date based on hour offset (relative to now)
		hourOffsetDuration := time.Duration(hourOffsetFloat * float64(time.Hour))
		newCard.FSRS.Due = time.Now().Add(hourOffsetDuration)

		// Update the card in storage
		if err := s.Storage.UpdateCard(newCard); err != nil {
			log.Printf("Warning: Failed to update card due date: %v", err)
		}
	}

	// Save changes to disk
	if err := s.Storage.Save(); err != nil {
		log.Printf("Warning: Failed to save storage after creating card: %v", err)
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

// handleUpdateCard handles the update_card tool request by updating an existing flashcard
// with the provided content. Only non-empty fields are updated, allowing for partial updates.
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

	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// Update the card using the service
	updatedCard, err := s.UpdateCard(cardID, front, back, tags)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Error updating card: %v"}`, err)), nil
	}

	// Create response
	response := UpdateCardResponse{
		Success: true,
		Message: fmt.Sprintf("Card %s updated successfully - Front: %s, Back: %s",
			cardID, updatedCard.Front, updatedCard.Back),
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleDeleteCard handles the delete_card tool request by removing a flashcard
// from storage. It first verifies the card exists before attempting deletion.
func handleDeleteCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract required parameter
	cardID, ok := request.Params.Arguments["card_id"].(string)
	if !ok {
		return mcp.NewToolResultText("Missing required parameter: card_id"), nil
	}

	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// First check if the card exists
	_, err := s.Storage.GetCard(cardID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Card not found: %v"}`, err)), nil
	}

	// Delete the card using the service
	err = s.DeleteCard(cardID)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Error deleting card: %v"}`, err)), nil
	}

	// Create response
	response := DeleteCardResponse{
		Success: true,
		Message: fmt.Sprintf("Card %s was successfully deleted", cardID),
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleListCards handles the list_cards tool request by retrieving all flashcards,
// optionally filtered by tags. It can also include statistics in the response if requested.
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

	includeStats := false
	if includeStatsVal, ok := request.Params.Arguments["include_stats"].(bool); ok {
		includeStats = includeStatsVal
	}

	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// Get cards from service
	cards, stats, err := s.ListCards(filterTags, includeStats)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Error listing cards: %v"}`, err)), nil
	}

	// Prepare the cards for the response
	var responseCards []Card
	for _, card := range cards {
		responseCards = append(responseCards, card)
	}

	// Create response
	response := ListCardsResponse{
		Cards: responseCards,
	}

	// Include stats if requested
	if includeStats {
		response.Stats = stats
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
