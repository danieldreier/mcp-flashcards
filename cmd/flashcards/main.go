package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// handleGetDueCard handles the get_due_card tool request
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

// handleDeleteCard handles the delete_card tool request
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

func main() {
	// Parse command-line flags
	filePath := flag.String("file", "./flashcards.json", "Path to flashcard data file")
	flag.Parse()

	// Initialize storage
	fileStorage := storage.NewFileStorage(*filePath)
	if err := fileStorage.Load(); err != nil {
		fmt.Printf("Error loading storage: %v\n", err)
		os.Exit(1)
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Flashcards MCP",
		"1.0.0",
		server.WithResourceCapabilities(true, true), // Resource capabilities for subscribe and listChanged
		server.WithToolCapabilities(true),           // Enable tool capabilities
		server.WithLogging(),                        // Enable logging for the server
	)

	// Initialize the flashcard service
	flashcardService := NewFlashcardService(fileStorage)

	// Create context with the service for tool handlers
	ctx := context.WithValue(context.Background(), "service", flashcardService)

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
	s.AddTool(getDueCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Pass the context with service to the handler
		return handleGetDueCard(ctx, request)
	})
	s.AddTool(submitReviewTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSubmitReview(ctx, request)
	})
	s.AddTool(createCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateCard(ctx, request)
	})
	s.AddTool(updateCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateCard(ctx, request)
	})
	s.AddTool(deleteCardTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteCard(ctx, request)
	})
	s.AddTool(listCardsTool, func(reqCtx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListCards(ctx, request)
	})

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Error serving MCP server: %v", err)
	}
}
