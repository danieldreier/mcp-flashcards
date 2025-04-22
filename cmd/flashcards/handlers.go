// Package main provides implementation for the flashcards MCP service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
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

	// Extract optional parameters
	var filterTags []string
	if tagsInterface, ok := request.Params.Arguments["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				filterTags = append(filterTags, tagStr)
			}
		}
	}

	// Call service method to get due card, passing filter tags
	card, stats, err := s.GetDueCard(filterTags)
	if err != nil {
		// If no cards are due (matching the filter), return a friendly message WITH stats
		if err.Error() == "no cards due for review" {
			// Create error response including stats
			errorResponse := struct {
				Error string    `json:"error"`
				Stats CardStats `json:"stats"`
			}{
				Error: "No cards due for review",
				Stats: stats, // Include the stats calculated by GetDueCard
			}
			jsonBytes, marshalErr := json.MarshalIndent(errorResponse, "", "  ")
			if marshalErr != nil {
				// Fallback to simpler error if marshaling fails
				return mcp.NewToolResultText(fmt.Sprintf(`{"error": "No cards due for review, stats unavailable: %v"}`, marshalErr)), nil
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		}
		// For other errors, return the error message (without stats)
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

// handleHelpAnalyzeLearning analyzes the student's learning progress by identifying
// low-scoring cards, finding patterns in difficult content, and providing data
// that assists the LLM in making personalized learning recommendations.
func handleHelpAnalyzeLearning(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultText("Error: Service not available"), nil
	}

	// Get all cards from storage to analyze
	allCards, err := s.Storage.ListCards(nil)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error": "Error listing cards: %v"}`, err)), nil
	}

	// Calculate overall stats
	stats := s.calculateStats(allCards)

	// If there are no cards, return early with empty result
	if len(allCards) == 0 {
		response := AnalyzeLearningResponse{
			LowScoringCards: []struct {
				Card        Card         `json:"card"`
				Reviews     []CardReview `json:"reviews"`
				AvgRating   float64      `json:"avg_rating"`
				ReviewCount int          `json:"review_count"`
			}{},
			CommonTags:   []string{},
			TotalReviews: 0,
			Stats:        stats,
		}

		jsonBytes, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}

	// Analyze each card's reviews to find difficult cards
	type cardAnalysis struct {
		Card        Card
		Reviews     []CardReview
		AvgRating   float64
		ReviewCount int
	}

	var analyzedCards []cardAnalysis
	tagFrequency := make(map[string]int)
	totalReviews := 0

	for _, storageCard := range allCards {
		// Convert storage.Card to our Card type
		card := Card{
			ID:        storageCard.ID,
			Front:     storageCard.Front,
			Back:      storageCard.Back,
			CreatedAt: storageCard.CreatedAt,
			Tags:      storageCard.Tags,
			FSRS:      storageCard.FSRS,
		}

		// Count tags for finding common patterns
		for _, tag := range card.Tags {
			tagFrequency[tag]++
		}

		// Get reviews for this card
		cardReviews, err := s.Storage.GetCardReviews(card.ID)
		if err != nil {
			continue // Skip this card if reviews can't be retrieved
		}

		// If there are no reviews, skip this card
		if len(cardReviews) == 0 {
			continue
		}

		// Convert storage.Review to CardReview for response
		simplifiedReviews := make([]CardReview, 0, len(cardReviews))
		ratingSum := 0
		for _, review := range cardReviews {
			ratingInt := int(review.Rating)
			ratingSum += ratingInt
			totalReviews++

			simplifiedReviews = append(simplifiedReviews, CardReview{
				Rating:    ratingInt,
				Timestamp: review.Timestamp,
				Answer:    review.Answer,
			})
		}

		// Calculate average rating
		avgRating := float64(ratingSum) / float64(len(cardReviews))

		// Store the analysis for this card
		analyzedCards = append(analyzedCards, cardAnalysis{
			Card:        card,
			Reviews:     simplifiedReviews,
			AvgRating:   avgRating,
			ReviewCount: len(cardReviews),
		})
	}

	// Sort cards by average rating (lowest first)
	sort.Slice(analyzedCards, func(i, j int) bool {
		return analyzedCards[i].AvgRating < analyzedCards[j].AvgRating
	})

	// Filter for low-scoring cards (avg rating <= 2.5)
	var lowScoringCards []cardAnalysis
	for _, analysis := range analyzedCards {
		if analysis.AvgRating <= 2.5 && analysis.ReviewCount > 0 {
			lowScoringCards = append(lowScoringCards, analysis)
		}

		// Limit to 10 most difficult cards
		if len(lowScoringCards) >= 10 {
			break
		}
	}

	// Find common tags among low-scoring cards
	lowScoringTagFrequency := make(map[string]int)
	for _, analysis := range lowScoringCards {
		for _, tag := range analysis.Card.Tags {
			lowScoringTagFrequency[tag]++
		}
	}

	// Sort tags by frequency for low-scoring cards
	type tagCount struct {
		Tag   string
		Count int
	}
	var commonTags []tagCount
	for tag, count := range lowScoringTagFrequency {
		if count > 1 { // Only include tags that appear in multiple cards
			commonTags = append(commonTags, tagCount{Tag: tag, Count: count})
		}
	}
	sort.Slice(commonTags, func(i, j int) bool {
		return commonTags[i].Count > commonTags[j].Count
	})

	// Extract just the tag names in order of frequency
	commonTagNames := make([]string, 0, len(commonTags))
	for _, tc := range commonTags {
		commonTagNames = append(commonTagNames, tc.Tag)
	}

	// Prepare response data structure
	responseData := AnalyzeLearningResponse{
		LowScoringCards: make([]struct {
			Card        Card         `json:"card"`
			Reviews     []CardReview `json:"reviews"`
			AvgRating   float64      `json:"avg_rating"`
			ReviewCount int          `json:"review_count"`
		}, len(lowScoringCards)),
		CommonTags:   commonTagNames,
		TotalReviews: totalReviews,
		Stats:        stats,
	}

	// Fill in the low-scoring cards data
	for i, analysis := range lowScoringCards {
		responseData.LowScoringCards[i] = struct {
			Card        Card         `json:"card"`
			Reviews     []CardReview `json:"reviews"`
			AvgRating   float64      `json:"avg_rating"`
			ReviewCount int          `json:"review_count"`
		}{
			Card:        analysis.Card,
			Reviews:     analysis.Reviews,
			AvgRating:   analysis.AvgRating,
			ReviewCount: analysis.ReviewCount,
		}
	}

	// Return formatted JSON response
	jsonBytes, err := json.MarshalIndent(responseData, "", "  ")
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleTagsResource generates a resource showing all available tags in the system
// and how many cards exist for each tag. This helps users and LLMs know what tags
// are available for filtering cards.
func handleTagsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return nil, fmt.Errorf("service not available")
	}

	// Get all cards from storage
	allCards, err := s.Storage.ListCards(nil)
	if err != nil {
		return nil, fmt.Errorf("error listing cards: %w", err)
	}

	// Map to count cards per tag
	tagCounts := make(map[string]int)
	for _, card := range allCards {
		for _, tag := range card.Tags {
			tagCounts[tag]++
		}
	}

	// Convert to sorted slice of tag info structs
	type TagInfo struct {
		Tag        string `json:"tag"`
		CardCount  int    `json:"card_count"`
		DueCount   int    `json:"due_count"`   // Count of cards with this tag that are due
		TotalCards int    `json:"total_cards"` // Total number of cards in the system
		DueCards   int    `json:"due_cards"`   // Total number of due cards in the system
	}

	// Calculate overall stats once
	now := time.Now()
	totalCards := len(allCards)
	dueCards := 0
	for _, card := range allCards {
		if !card.FSRS.Due.After(now) {
			dueCards++
		}
	}

	// Calculate due counts per tag
	tagDueCounts := make(map[string]int)
	for _, card := range allCards {
		if !card.FSRS.Due.After(now) {
			for _, tag := range card.Tags {
				tagDueCounts[tag]++
			}
		}
	}

	// Convert map to sorted slice
	tags := make([]TagInfo, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tags = append(tags, TagInfo{
			Tag:        tag,
			CardCount:  count,
			DueCount:   tagDueCounts[tag],
			TotalCards: totalCards,
			DueCards:   dueCards,
		})
	}

	// Sort tags alphabetically for consistent display
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Tag < tags[j].Tag
	})

	// Marshal to JSON for resource response
	jsonBytes, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling tags to JSON: %w", err)
	}

	// Create TextResourceContents with the JSON data
	textContent := &mcp.TextResourceContents{
		URI:      "available-tags",
		MIMEType: "application/json",
		Text:     string(jsonBytes),
	}

	// Return as ResourceContents slice (interfaces)
	var contents []mcp.ResourceContents
	contents = append(contents, textContent)

	return contents, nil
}
