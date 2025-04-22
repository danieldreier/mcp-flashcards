// Package main provides implementation for the flashcards MCP service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/google/uuid"
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
		// Create a standard error response structure that includes stats
		type ErrorResponseWithStats struct {
			Error string    `json:"error"`
			Stats CardStats `json:"stats"`
		}
		// Default error message
		errorMsg := fmt.Sprintf("Error getting due card: %v", err)

		// Customize error message based on specific error type from service
		if strings.Contains(err.Error(), "no cards due for review") {
			if len(filterTags) > 0 {
				errorMsg = fmt.Sprintf("No cards due for review with the specified tags: %v", filterTags)
			} else {
				errorMsg = "No cards due for review"
			}
		} else if strings.Contains(err.Error(), "no cards found with the specified tags") {
			errorMsg = fmt.Sprintf("No cards found with the specified tags: %v", filterTags)
		}

		// Always include stats in the error response if available (stats are calculated even if GetDueCard returns error)
		errorResponse := ErrorResponseWithStats{
			Error: errorMsg,
			Stats: stats, // Include the stats calculated by GetDueCard
		}
		jsonBytes, marshalErr := json.MarshalIndent(errorResponse, "", "  ")
		if marshalErr != nil {
			// Fallback to simpler error if marshaling fails
			return mcp.NewToolResultText(fmt.Sprintf(`{"error": "%s", "stats_error": "%v"}`, errorMsg, marshalErr)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil // Return error with stats
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
	textContent := mcp.TextResourceContents{
		URI:      "available-tags",
		MIMEType: "application/json",
		Text:     string(jsonBytes),
	}

	// Return as ResourceContents slice (interfaces)
	var contents []mcp.ResourceContents
	contents = append(contents, textContent)

	return contents, nil
}

// handleManageDueDates handles CRUD operations for due date entries.
func handleManageDueDates(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return mcp.NewToolResultError("Service not available"), nil
	}

	// Extract action parameter
	action, _ := request.Params.Arguments["action"].(string)
	if action == "" {
		return mcp.NewToolResultError("Missing required parameter: action"), nil
	}

	// Extract other parameters (optional depending on action)
	topic, _ := request.Params.Arguments["topic"].(string)
	dateStr, _ := request.Params.Arguments["date"].(string)
	dueDateID, _ := request.Params.Arguments["due_date_id"].(string)
	tag, _ := request.Params.Arguments["tag"].(string)

	switch action {
	case "create":
		if topic == "" || dateStr == "" {
			return mcp.NewToolResultError("Missing required parameters for create: topic, date (YYYY-MM-DD)"), nil
		}
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid date format: %s. Use YYYY-MM-DD.", dateStr)), nil
		}

		// Generate tag if not provided (or validate if provided? For now, generate)
		if tag == "" {
			// Simple tag generation: test-<topic>-<date>
			safeTopic := strings.ToLower(strings.ReplaceAll(topic, " ", "-"))
			// Remove special chars from topic for tag? Keep simple for now.
			tag = fmt.Sprintf("test-%s-%s", safeTopic, dateStr)
		}

		newDueDate := storage.DueDate{
			ID:      uuid.NewString(), // Generate new ID
			Topic:   topic,
			DueDate: parsedDate,
			Tag:     tag,
		}

		if err := s.AddDueDate(newDueDate); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating due date: %v", err)), nil
		}
		jsonBytes, _ := json.MarshalIndent(newDueDate, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil

	case "list":
		dueDates, err := s.ListDueDates()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error listing due dates: %v", err)), nil
		}
		if len(dueDates) == 0 {
			return mcp.NewToolResultText("[]"), nil // Return empty JSON array
		}
		jsonBytes, err := json.MarshalIndent(dueDates, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error marshaling due dates: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil

	case "update":
		if dueDateID == "" {
			return mcp.NewToolResultError("Missing required parameter for update: due_date_id"), nil
		}
		// Fetch existing due date to update
		existingDueDates, err := s.ListDueDates() // Inefficient, need GetDueDateByID in service/storage
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error fetching existing due dates: %v", err)), nil
		}
		var existingDueDate *storage.DueDate
		for i := range existingDueDates {
			if existingDueDates[i].ID == dueDateID {
				existingDueDate = &existingDueDates[i]
				break
			}
		}
		if existingDueDate == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Due date with ID %s not found", dueDateID)), nil
		}

		// Update fields if provided
		if topic != "" {
			existingDueDate.Topic = topic
		}
		if dateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid date format: %s. Use YYYY-MM-DD.", dateStr)), nil
			}
			existingDueDate.DueDate = parsedDate
		}
		if tag != "" {
			existingDueDate.Tag = tag
		}

		if err := s.UpdateDueDate(*existingDueDate); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error updating due date: %v", err)), nil
		}
		jsonBytes, _ := json.MarshalIndent(*existingDueDate, "", "  ")
		return mcp.NewToolResultText(string(jsonBytes)), nil

	case "delete":
		if dueDateID == "" {
			return mcp.NewToolResultError("Missing required parameter for delete: due_date_id"), nil
		}
		if err := s.DeleteDueDate(dueDateID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error deleting due date: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf(`{"message": "Due date %s deleted successfully"}`, dueDateID)), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("Invalid action: %s. Must be one of 'create', 'update', 'delete', 'list'", action)), nil
	}
}

// DueDateProgressInfo holds detailed progress for a single due date.
type DueDateProgressInfo struct {
	ID              string  `json:"id"`
	Topic           string  `json:"topic"`
	DueDate         string  `json:"due_date"` // YYYY-MM-DD format
	Tag             string  `json:"tag"`
	TotalCards      int     `json:"total_cards"`
	MasteredCards   int     `json:"mastered_cards"`
	ProgressPercent float64 `json:"progress_percent"`
	DaysRemaining   float64 `json:"days_remaining"` // Days until day *before* due date
	CardsLeft       int     `json:"cards_left"`
	RequiredPace    float64 `json:"required_pace"` // Cards per day needed
}

// handleDueDateProgressResource generates a resource showing progress towards upcoming due dates.
func handleDueDateProgressResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Get the service from context
	s, ok := ctx.Value("service").(*FlashcardService)
	if !ok || s == nil {
		return nil, fmt.Errorf("service not available")
	}

	// Get all defined due dates
	dueDates, err := s.ListDueDates()
	if err != nil {
		return nil, fmt.Errorf("error listing due dates: %w", err)
	}

	// Log the number of due dates found
	fmt.Printf("Found %d due dates in handleDueDateProgressResource\n", len(dueDates))
	for i, dd := range dueDates {
		fmt.Printf("Due date %d: ID=%s, Topic=%s, Date=%s, Tag=%s\n",
			i+1, dd.ID, dd.Topic, dd.DueDate.Format("2006-01-02"), dd.Tag)
	}

	now := time.Now()
	// Truncate now to the beginning of the day for consistent day calculation
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	progressInfos := []DueDateProgressInfo{}
	for _, dd := range dueDates {
		// Skip past due dates - they aren't useful for forward planning
		// Exception: include past due dates with "test-" prefix for testing purposes
		dueDay := time.Date(dd.DueDate.Year(), dd.DueDate.Month(), dd.DueDate.Day(), 0, 0, 0, 0, dd.DueDate.Location())
		if dueDay.Before(today) && !strings.HasPrefix(dd.Tag, "test-") {
			fmt.Printf("Skipping past due date: %s (date: %s)\n", dd.Topic, dueDay.Format("2006-01-02"))
			continue
		}

		fmt.Printf("Processing due date: %s (tag: %s)\n", dd.Topic, dd.Tag)

		// Get progress stats for the associated tag
		stats, err := s.GetDueDateProgressStats(dd.Tag)
		if err != nil {
			// Log error but continue? Or fail resource? Let's log and skip this one.
			fmt.Printf("Warning: could not get progress stats for tag %s (due date %s): %v\n", dd.Tag, dd.ID, err)
			continue
		}

		// Calculate days remaining (until the day *before* the due date)
		// Ensure due date is also truncated for comparison
		daysRemaining := dueDay.Sub(today).Hours() / 24.0

		// If due date is in the past, set days remaining to 0
		if daysRemaining < 0 {
			daysRemaining = 0
		} else {
			// Otherwise exclude the test day itself, minimum 0
			daysRemaining = math.Max(0, daysRemaining-1)
		}

		cardsLeft := stats.TotalCards - stats.MasteredCards
		requiredPace := 0.0
		if daysRemaining > 0 && cardsLeft > 0 {
			requiredPace = float64(cardsLeft) / daysRemaining
		}

		info := DueDateProgressInfo{
			ID:              dd.ID,
			Topic:           dd.Topic,
			DueDate:         dd.DueDate.Format("2006-01-02"),
			Tag:             dd.Tag,
			TotalCards:      stats.TotalCards,
			MasteredCards:   stats.MasteredCards,
			ProgressPercent: stats.ProgressPercent,
			DaysRemaining:   daysRemaining,
			CardsLeft:       cardsLeft,
			RequiredPace:    requiredPace,
		}
		progressInfos = append(progressInfos, info)
		fmt.Printf("Added progress info: %+v\n", info)
	}

	// Sort by due date ascending
	sort.Slice(progressInfos, func(i, j int) bool {
		d1, _ := time.Parse("2006-01-02", progressInfos[i].DueDate)
		d2, _ := time.Parse("2006-01-02", progressInfos[j].DueDate)
		return d1.Before(d2)
	})

	// Ensure we're returning at least an empty array instead of null
	if progressInfos == nil {
		progressInfos = []DueDateProgressInfo{}
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(progressInfos, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling due date progress: %w", err)
	}

	// Create TextResourceContents as a direct value, not a pointer
	textContent := mcp.TextResourceContents{
		URI:      "due-date-progress", // Ensure this matches the resource definition
		MIMEType: "application/json",
		Text:     string(jsonBytes),
	}

	fmt.Printf("Resource content created - URI: %s, MIMEType: %s, Text: %s\n",
		textContent.URI, textContent.MIMEType, textContent.Text)

	// Return as ResourceContents slice
	var contents []mcp.ResourceContents
	contents = append(contents, textContent) // Add the value directly, not a pointer

	return contents, nil
}
