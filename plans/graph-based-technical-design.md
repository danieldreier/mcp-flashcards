# Graph-Based Flashcard Technical Design

## 1. Overview

This document outlines the technical design for extending the Flashcards MCP server with graph-based features. The enhanced system will:

- Implement a directed graph structure to represent relationships between flashcards
- Filter cards based on prerequisite mastery (score of 3+)
- Detect repeated failures to trigger teaching mode
- Support problem decomposition through LLM-generated prerequisite cards

These enhancements will transform the system from a simple spaced repetition tool into an intelligent learning system that understands knowledge dependencies and can adapt to student learning gaps.

## 2. Libraries and Dependencies

```go
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "sync"
    "time"
    
    "github.com/google/uuid"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/open-spaced-repetition/go-fsrs"
    
    // New imports for graph functionality
    "github.com/danieldreier/mcp-flashcards/internal/graph"
)
```

For the graph implementation, we'll create a custom internal package rather than using a third-party library. This approach keeps the implementation focused on our specific needs and simplifies integration with the existing codebase.

## 3. Data Structures

### 3.1 Graph Data Structures

```go
// DirectedGraph represents a simple directed graph structure
type DirectedGraph struct {
    // Map from node ID to its prerequisite node IDs
    Prerequisites map[string][]string `json:"prerequisites"`
    // Map from node ID to the IDs of nodes that depend on it
    Dependents map[string][]string `json:"dependents"`
}

// ConceptNode represents a knowledge concept in the graph
type ConceptNode struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    Description   string    `json:"description,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
}

// CardRelationship represents how cards relate to concepts
type CardRelationship struct {
    // Map from concept ID to card IDs that teach this concept
    ConceptToCards map[string][]string `json:"concept_to_cards"`
    // Map from card ID to concept IDs it teaches
    CardToConcepts map[string][]string `json:"card_to_concepts"`
}
```

### 3.2 Enhanced Card Structure

```go
// Enhanced Card with graph relationships and failure tracking
type Card struct {
    ID              string     `json:"id"`
    Front           string     `json:"front"`
    Back            string     `json:"back"`
    CreatedAt       time.Time  `json:"created_at"`
    Tags            []string   `json:"tags,omitempty"`
    FSRS            fsrs.Card  `json:"fsrs"`
    
    // New fields for graph-based learning
    ConceptIDs       []string   `json:"concept_ids,omitempty"`
    ConsecutiveFails int        `json:"consecutive_fails,omitempty"`
    LastRating       fsrs.Rating `json:"last_rating,omitempty"`
}
```

### 3.3 Enhanced Storage Structure

```go
// Enhanced FlashcardStore with graph structures
type FlashcardStore struct {
    Cards           map[string]Card       `json:"cards"`
    Reviews         []Review              `json:"reviews"`
    Concepts        map[string]ConceptNode `json:"concepts"`
    CardGraph       DirectedGraph         `json:"card_graph"`
    ConceptGraph    DirectedGraph         `json:"concept_graph"`
    Relationships   CardRelationship      `json:"relationships"`
    LastUpdated     time.Time             `json:"last_updated"`
}
```

### 3.4 Response Types for New Features

```go
// EnhancedCardResponse includes graph information
type EnhancedCardResponse struct {
    Card              Card       `json:"card"`
    Stats             CardStats  `json:"stats"`
    Prerequisites     []Card     `json:"prerequisites,omitempty"`
    Concepts          []string   `json:"concepts,omitempty"`
    TeachingMode      bool       `json:"teaching_mode,omitempty"`
}

// ConceptAnalysisResponse for problem decomposition
type ConceptAnalysisResponse struct {
    CardID            string     `json:"card_id"`
    Content           string     `json:"content"`
    MissingConcepts   []string   `json:"missing_concepts,omitempty"`
    ExistingConcepts  []string   `json:"existing_concepts,omitempty"`
}

// DecompositionResponse for the teaching mode
type DecompositionResponse struct {
    OriginalCardID    string     `json:"original_card_id"`
    NewPrerequisites  []Card     `json:"new_prerequisites"`
    ConceptsCreated   []string   `json:"concepts_created"`
    ExplanationText   string     `json:"explanation_text"`
}
```

## 4. Enhanced Storage Interface

```go
// Enhanced Storage interface with graph operations
type Storage interface {
    // Existing operations
    CreateCard(front, back string, tags []string) (Card, error)
    GetCard(id string) (Card, error)
    UpdateCard(card Card) error
    DeleteCard(id string) error
    ListCards(tags []string) ([]Card, error)
    AddReview(cardID string, rating fsrs.Rating, answer string) (Review, error)
    GetCardReviews(cardID string) ([]Review, error)
    Load() error
    Save() error
    
    // New graph operations
    CreateConcept(name, description string) (ConceptNode, error)
    GetConcept(id string) (ConceptNode, error)
    ListConcepts() ([]ConceptNode, error)
    
    // Card-Concept relationships
    AssociateCardWithConcept(cardID, conceptID string) error
    GetConceptCards(conceptID string) ([]Card, error)
    GetCardConcepts(cardID string) ([]ConceptNode, error)
    
    // Graph relationships
    AddPrerequisiteCard(prerequisiteID, dependentID string) error
    GetCardPrerequisites(cardID string) ([]Card, error)
    GetCardDependents(cardID string) ([]Card, error)
    
    // Concept graph operations
    AddPrerequisiteConcept(prerequisiteID, dependentID string) error
    GetConceptPrerequisites(conceptID string) ([]ConceptNode, error)
    GetConceptDependents(conceptID string) ([]ConceptNode, error)
    
    // Failure tracking
    UpdateCardFailureCount(cardID string, rating fsrs.Rating) error
    GetCardFailureCount(cardID string) (int, error)
}
```

## 5. MCP Tool Definitions

### 5.1 Enhanced Existing Tools

```go
// Enhanced get_due_card tool with prerequisite checking
getDueCardTool := mcp.NewTool("get_due_card",
    mcp.WithDescription("Get the next flashcard due for review, ensuring all prerequisites are mastered (rating 3+)"),
    mcp.WithBoolean("check_prerequisites",
        mcp.Description("Whether to check if prerequisites are mastered before returning a card (default: true)"),
    ),
)

// Enhanced submit_review tool with teaching mode detection
submitReviewTool := mcp.NewTool("submit_review",
    mcp.WithDescription("Submit a review rating for a flashcard and track repeated failures"),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card being reviewed"),
    ),
    mcp.WithNumber("rating",
        mcp.Required(),
        mcp.Description("Rating from 1-4 (Again=1, Hard=2, Good=3, Easy=4)"),
        mcp.MinValue(1),
        mcp.MaxValue(4),
    ),
    mcp.WithString("answer",
        mcp.Description("The user's answer (optional)"),
    ),
)
```

### 5.2 New Graph-Related Tools

```go
// Tool for creating a concept
createConceptTool := mcp.NewTool("create_concept",
    mcp.WithDescription("Create a new concept in the knowledge graph"),
    mcp.WithString("name",
        mcp.Required(),
        mcp.Description("Name of the concept"),
    ),
    mcp.WithString("description",
        mcp.Required(),
        mcp.Description("Description of the concept"),
    ),
)

// Tool for linking cards
linkCardsTool := mcp.NewTool("link_cards",
    mcp.WithDescription("Create a prerequisite relationship between cards"),
    mcp.WithString("prerequisite_card_id",
        mcp.Required(),
        mcp.Description("ID of the prerequisite card"),
    ),
    mcp.WithString("dependent_card_id",
        mcp.Required(),
        mcp.Description("ID of the dependent card"),
    ),
)

// Tool for linking concepts
linkConceptsTool := mcp.NewTool("link_concepts",
    mcp.WithDescription("Create a prerequisite relationship between concepts"),
    mcp.WithString("prerequisite_concept_id",
        mcp.Required(),
        mcp.Description("ID of the prerequisite concept"),
    ),
    mcp.WithString("dependent_concept_id",
        mcp.Required(),
        mcp.Description("ID of the dependent concept"),
    ),
)

// Tool for associating cards with concepts
associateCardWithConceptTool := mcp.NewTool("associate_card_with_concept",
    mcp.WithDescription("Associate a card with a concept"),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card"),
    ),
    mcp.WithString("concept_id",
        mcp.Required(),
        mcp.Description("ID of the concept"),
    ),
)
```

### 5.3 Problem Decomposition Tools

```go
// Tool for analyzing knowledge gaps when a card has too many failures
analyzeKnowledgeGapsTool := mcp.NewTool("analyze_knowledge_gaps",
    mcp.WithDescription("Analyze why a student is struggling with a card and identify missing prerequisite knowledge"),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card the student is struggling with"),
    ),
    mcp.WithArray("review_history",
        mcp.Description("Recent review history for this card"),
        mcp.Required(),
    ),
)

// Tool for creating a prerequisite card 
createPrerequisiteCardTool := mcp.NewTool("create_prerequisite_card",
    mcp.WithDescription("Create a new flashcard as a prerequisite for an existing card"),
    mcp.WithString("dependent_card_id",
        mcp.Required(),
        mcp.Description("ID of the dependent card"),
    ),
    mcp.WithString("concept_id",
        mcp.Required(),
        mcp.Description("ID of the concept this card teaches"),
    ),
    mcp.WithString("front",
        mcp.Required(),
        mcp.Description("Front side of the flashcard (question)"),
    ),
    mcp.WithString("back",
        mcp.Required(),
        mcp.Description("Back side of the flashcard (answer)"),
    ),
)

// Tool for decomposing a problem into prerequisite knowledge
decomposeKnowledgeTool := mcp.NewTool("decompose_knowledge",
    mcp.WithDescription("Break down a complex topic into prerequisite concepts and create cards for them"),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card to decompose"),
    ),
    mcp.WithString("explanation",
        mcp.Required(),
        mcp.Description("Explanation of how to break down this topic"),
    ),
)
```

## 6. Handler Function Implementations

### 6.1 Enhanced Existing Handlers

```go
// Enhanced GetDueCardHandler with prerequisite checking
func GetDueCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract check_prerequisites parameter (default to true)
    checkPrerequisites := true
    if val, exists := request.Arguments["check_prerequisites"]; exists {
        if boolVal, ok := val.(bool); ok {
            checkPrerequisites = boolVal
        }
    }

    // Get due cards with prerequisite filtering if enabled
    var card Card
    var stats CardStats
    var prerequisites []Card
    var err error

    if checkPrerequisites {
        card, stats, prerequisites, err = service.GetDueCardWithPrerequisites()
    } else {
        card, stats, err = service.GetDueCard()
    }

    if err != nil {
        return nil, err
    }

    // Build response with additional prerequisite information
    response := EnhancedCardResponse{
        Card:           card,
        Stats:          stats,
        Prerequisites:  prerequisites,
    }

    return &mcp.CallToolResult{
        Result: response,
    }, nil
}

// Enhanced SubmitReviewHandler with teaching mode detection
func SubmitReviewHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract parameters
    cardID := request.Arguments["card_id"].(string)
    ratingValue := int(request.Arguments["rating"].(float64))
    answer, _ := request.Arguments["answer"].(string)

    // Convert to FSRS rating
    rating := fsrs.Rating(ratingValue)

    // Submit review and track failures
    card, teachingMode, err := service.SubmitReviewWithFailureTracking(cardID, rating, answer)
    if err != nil {
        return nil, err
    }

    // Build response with teaching mode flag
    response := struct {
        Success          bool   `json:"success"`
        Message          string `json:"message"`
        Card             Card   `json:"card"`
        TeachingMode     bool   `json:"teaching_mode"`
        ConsecutiveFails int    `json:"consecutive_fails"`
    }{
        Success:          true,
        Message:          "Review submitted successfully",
        Card:             card,
        TeachingMode:     teachingMode,
        ConsecutiveFails: card.ConsecutiveFails,
    }

    return &mcp.CallToolResult{
        Result: response,
    }, nil
}
```

### 6.2 New Graph-Related Handlers

```go
// CreateConceptHandler handles create_concept requests
func CreateConceptHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract concept information
    name := request.Arguments["name"].(string)
    description := request.Arguments["description"].(string)

    // Create concept in storage
    concept, err := service.CreateConcept(name, description)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Result: concept,
    }, nil
}

// LinkCardsHandler handles link_cards requests
func LinkCardsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract card IDs
    prerequisiteID := request.Arguments["prerequisite_card_id"].(string)
    dependentID := request.Arguments["dependent_card_id"].(string)

    // Create the prerequisite relationship
    err := service.AddPrerequisiteCard(prerequisiteID, dependentID)
    if err != nil {
        return nil, err
    }

    // Build response
    response := struct {
        Success  bool   `json:"success"`
        Message  string `json:"message"`
    }{
        Success: true,
        Message: fmt.Sprintf("Card %s is now a prerequisite for card %s", prerequisiteID, dependentID),
    }

    return &mcp.CallToolResult{
        Result: response,
    }, nil
}
```

### 6.3 Problem Decomposition Handlers

```go
// AnalyzeKnowledgeGapsHandler handles analyze_knowledge_gaps requests
func AnalyzeKnowledgeGapsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract card ID
    cardID := request.Arguments["card_id"].(string)

    // Get the card and its review history
    card, err := service.GetCard(cardID)
    if err != nil {
        return nil, err
    }

    reviews, err := service.GetCardReviews(cardID)
    if err != nil {
        return nil, err
    }

    // Get concept information for the card
    concepts, err := service.GetCardConcepts(cardID)
    if err != nil {
        return nil, err
    }

    conceptNames := make([]string, len(concepts))
    for i, concept := range concepts {
        conceptNames[i] = concept.Name
    }

    // Build response for LLM analysis
    response := ConceptAnalysisResponse{
        CardID:           cardID,
        Content:          fmt.Sprintf("Front: %s\nBack: %s", card.Front, card.Back),
        ExistingConcepts: conceptNames,
    }

    return &mcp.CallToolResult{
        Result: response,
    }, nil
}

// CreatePrerequisiteCardHandler handles create_prerequisite_card requests
func CreatePrerequisiteCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract parameters
    dependentCardID := request.Arguments["dependent_card_id"].(string)
    conceptID := request.Arguments["concept_id"].(string)
    front := request.Arguments["front"].(string)
    back := request.Arguments["back"].(string)

    // Create the new prerequisite card
    card, err := service.CreateCard(front, back, nil)
    if err != nil {
        return nil, err
    }

    // Associate with concept
    err = service.AssociateCardWithConcept(card.ID, conceptID)
    if err != nil {
        return nil, err
    }

    // Link as prerequisite to the dependent card
    err = service.AddPrerequisiteCard(card.ID, dependentCardID)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Result: card,
    }, nil
}

// DecomposeKnowledgeHandler handles decompose_knowledge requests
func DecomposeKnowledgeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract parameters
    cardID := request.Arguments["card_id"].(string)
    explanation := request.Arguments["explanation"].(string)

    // This handler primarily passes information back to the LLM
    // The actual decomposition is performed by the LLM using other tools
    
    // Build response
    response := struct {
        CardID      string `json:"card_id"`
        Explanation string `json:"explanation"`
        Success     bool   `json:"success"`
    }{
        CardID:      cardID,
        Explanation: explanation,
        Success:     true,
    }

    return &mcp.CallToolResult{
        Result: response,
    }, nil
}
```

## 7. Service Implementation

### 7.1 Graph-Related Service Methods

```go
// CreateConcept creates a new concept in the knowledge graph
func (s *Service) CreateConcept(name, description string) (ConceptNode, error) {
    // Create a new concept with a unique ID
    id := uuid.New().String()
    now := time.Now()
    
    concept := ConceptNode{
        ID:          id,
        Name:        name,
        Description: description,
        CreatedAt:   now,
    }
    
    // Store the concept
    s.storage.store.Concepts[id] = concept
    
    // Initialize graph entries for this concept
    if s.storage.store.ConceptGraph.Prerequisites == nil {
        s.storage.store.ConceptGraph.Prerequisites = make(map[string][]string)
    }
    if s.storage.store.ConceptGraph.Dependents == nil {
        s.storage.store.ConceptGraph.Dependents = make(map[string][]string)
    }
    
    s.storage.store.ConceptGraph.Prerequisites[id] = []string{}
    s.storage.store.ConceptGraph.Dependents[id] = []string{}
    
    // Save changes
    if err := s.storage.Save(); err != nil {
        return ConceptNode{}, err
    }
    
    return concept, nil
}

// AddPrerequisiteCard creates a prerequisite relationship between cards
func (s *Service) AddPrerequisiteCard(prerequisiteID, dependentID string) error {
    // Check if both cards exist
    if _, err := s.storage.GetCard(prerequisiteID); err != nil {
        return err
    }
    if _, err := s.storage.GetCard(dependentID); err != nil {
        return err
    }
    
    // Initialize graph maps if needed
    if s.storage.store.CardGraph.Prerequisites == nil {
        s.storage.store.CardGraph.Prerequisites = make(map[string][]string)
    }
    if s.storage.store.CardGraph.Dependents == nil {
        s.storage.store.CardGraph.Dependents = make(map[string][]string)
    }
    
    // Add prerequisite relationship
    // Check if the relationship already exists
    found := false
    for _, id := range s.storage.store.CardGraph.Prerequisites[dependentID] {
        if id == prerequisiteID {
            found = true
            break
        }
    }
    
    if !found {
        // Add prerequisite to dependent
        s.storage.store.CardGraph.Prerequisites[dependentID] = append(
            s.storage.store.CardGraph.Prerequisites[dependentID], 
            prerequisiteID,
        )
        
        // Add dependent to prerequisite
        s.storage.store.CardGraph.Dependents[prerequisiteID] = append(
            s.storage.store.CardGraph.Dependents[prerequisiteID], 
            dependentID,
        )
    }
    
    // Save changes
    if err := s.storage.Save(); err != nil {
        return err
    }
    
    return nil
}

// GetCardPrerequisites gets all prerequisites for a card
func (s *Service) GetCardPrerequisites(cardID string) ([]Card, error) {
    // Check if card exists
    if _, err := s.storage.GetCard(cardID); err != nil {
        return nil, err
    }
    
    // Get prerequisite IDs
    prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
    if prereqIDs == nil {
        return []Card{}, nil
    }
    
    // Get all prerequisite cards
    prereqs := make([]Card, 0, len(prereqIDs))
    for _, id := range prereqIDs {
        card, err := s.storage.GetCard(id)
        if err != nil {
            // Skip cards that can't be found
            continue
        }
        prereqs = append(prereqs, card)
    }
    
    return prereqs, nil
}
```

### 7.2 Prerequisite Checking Implementation

```go
// CheckPrerequisitesMastered checks if all prerequisites for a card have a rating of 3+
func (s *Service) CheckPrerequisitesMastered(cardID string) (bool, error) {
    // Get all prerequisites for the card
    prereqs, err := s.GetCardPrerequisites(cardID)
    if err != nil {
        return false, err
    }
    
    // If no prerequisites, consider them mastered
    if len(prereqs) == 0 {
        return true, nil
    }
    
    // Check each prerequisite
    for _, prereq := range prereqs {
        // Get the most recent review for this prerequisite
        isNew := prereq.FSRS.State == fsrs.New
        isLearning := prereq.FSRS.State == fsrs.Learning
        
        // If the card is new or still being learned, it's not mastered
        if isNew || isLearning {
            return false, nil
        }
        
        // If the card is in review state, check if the last rating was 3+
        if prereq.FSRS.State == fsrs.Review {
            if prereq.LastRating < 3 {
                return false, nil
            }
        }
    }
    
    // All prerequisites are mastered
    return true, nil
}

// GetDueCardWithPrerequisites gets the next due card that has all prerequisites mastered
func (s *Service) GetDueCardWithPrerequisites() (Card, CardStats, []Card, error) {
    // Get all cards
    allCards, err := s.storage.ListCards(nil)
    if err != nil {
        return Card{}, CardStats{}, nil, err
    }
    
    // Filter due cards
    now := time.Now()
    dueCards := []Card{}
    for _, card := range allCards {
        if card.FSRS.Due.Before(now) || card.FSRS.Due.Equal(now) {
            dueCards = append(dueCards, card)
        }
    }
    
    if len(dueCards) == 0 {
        return Card{}, CardStats{}, nil, errors.New("no cards due for review")
    }
    
    // Sort due cards by due date
    sort.Slice(dueCards, func(i, j int) bool {
        return dueCards[i].FSRS.Due.Before(dueCards[j].FSRS.Due)
    })
    
    // Find the first card with all prerequisites mastered
    var selectedCard Card
    var prerequisites []Card
    
    for _, card := range dueCards {
        mastered, err := s.CheckPrerequisitesMastered(card.ID)
        if err != nil {
            continue
        }
        
        if mastered {
            selectedCard = card
            // Get prerequisites for informational purposes
            prerequisites, _ = s.GetCardPrerequisites(card.ID)
            break
        }
    }
    
    // If no card with mastered prerequisites, pick the most critical prerequisite
    if selectedCard.ID == "" && len(dueCards) > 0 {
        // Get the first due card and its unmastered prerequisites
        firstCard := dueCards[0]
        allPrereqs, _ := s.GetCardPrerequisites(firstCard.ID)
        
        for _, prereq := range allPrereqs {
            mastered := prereq.FSRS.State == fsrs.Review && prereq.LastRating >= 3
            if !mastered {
                selectedCard = prereq
                break
            }
        }
        
        // If still no card selected, use the first due card
        if selectedCard.ID == "" {
            selectedCard = firstCard
        }
    }
    
    // Calculate stats
    stats := s.CalculateStats(allCards)
    
    return selectedCard, stats, prerequisites, nil
}
```

### 7.3 Failure Tracking Implementation

```go
// SubmitReviewWithFailureTracking enhances SubmitReview to track consecutive failures
func (s *Service) SubmitReviewWithFailureTracking(cardID string, rating fsrs.Rating, answer string) (Card, bool, error) {
    // Get the card
    card, err := s.storage.GetCard(cardID)
    if err != nil {
        return Card{}, false, err
    }
    
    // Update consecutive failures counter
    if rating <= 2 { // Hard or Again
        card.ConsecutiveFails++
    } else {
        card.ConsecutiveFails = 0
    }
    
    // Set last rating
    card.LastRating = rating
    
    // Update the card
    if err := s.storage.UpdateCard(card); err != nil {
        return Card{}, false, err
    }
    
    // Process the review with FSRS algorithm
    reviewLog := fsrs.ReviewLog{
        Rating: rating,
    }
    
    newCard := s.scheduler.Repeat(&card.FSRS, reviewLog)
    card.FSRS = *newCard
    
    // Update the card again with FSRS data
    if err := s.storage.UpdateCard(card); err != nil {
        return Card{}, false, err
    }
    
    // Record the review
    if _, err := s.storage.AddReview(cardID, rating, answer); err != nil {
        return Card{}, false, err
    }
    
    // Determine if teaching mode should be activated (3+ consecutive fails)
    teachingMode := card.ConsecutiveFails >= 3
    
    return card, teachingMode, nil
}
```

## 8. Problem Decomposition Implementation

The problem decomposition feature relies heavily on LLM capabilities. The MCP tools and handlers provide the necessary interface for the LLM to:

1. Analyze why a student is struggling with a card
2. Identify missing prerequisite knowledge
3. Create new concepts and cards for these prerequisites
4. Link them in the graph
5. Present a friendly explanation to the student

The MCP instructions will be enhanced to guide the LLM through this process when teaching mode is activated:

```go
// LLM instruction snippet for problem decomposition
// This would be part of the larger MCP server configuration

// Enhanced MCP server configuration
s := server.NewMCPServer(
    "Flashcards MCP with Knowledge Graph",
    "2.0.0",
    server.WithResourceCapabilities(true, true),
    server.WithLogging(),
    server.WithInstructions(`When a student consistently struggles with a flashcard (3+ failed attempts), 
enter teaching mode and follow this process:

1. Use analyze_knowledge_gaps to understand what the student is missing
2. Identify 2-3 key prerequisite concepts that would help the student understand this topic
3. For each concept:
   a. Create it using create_concept
   b. Create a simple flashcard for it using create_prerequisite_card
   c. Link it to the original card using link_cards
4. Explain to the student in a conversational way:
   "I notice you're having trouble with [topic]. Let's break it down into smaller parts:
   - First, let's understand [prerequisite 1]
   - Then, we'll look at [prerequisite 2]
   - Finally, we'll come back to [original topic]"
5. Start by showing the simplest prerequisite card first`),
)
```

## 9. Graph Management Tools

In addition to the core functionality, we'll implement several utility methods for managing the graph:

```go
// DetectCycles checks if adding a new prerequisite would create a cycle
func (s *Service) DetectCycles(prerequisiteID, dependentID string) bool {
    // Simple DFS to detect cycles
    visited := make(map[string]bool)
    
    var checkCycle func(current, target string) bool
    checkCycle = func(current, target string) bool {
        if current == target {
            return true // Found a cycle
        }
        
        visited[current] = true
        
        for _, dependent := range s.storage.store.CardGraph.Dependents[current] {
            if !visited[dependent] {
                if checkCycle(dependent, target) {
                    return true
                }
            }
        }
        
        return false
    }
    
    // Check if adding this edge would create a cycle
    return checkCycle(dependentID, prerequisiteID)
}

// GetLearningPath returns a topologically sorted list of cards to learn
func (s *Service) GetLearningPath(targetCardID string) ([]Card, error) {
    // Find all prerequisites recursively
    prereqMap := make(map[string]bool)
    
    var collectPrereqs func(cardID string)
    collectPrereqs = func(cardID string) {
        if _, visited := prereqMap[cardID]; visited {
            return
        }
        
        prereqMap[cardID] = true
        
        prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
        for _, prereqID := range prereqIDs {
            collectPrereqs(prereqID)
        }
    }
    
    // Start from the target card
    collectPrereqs(targetCardID)
    
    // Create a topologically sorted list using known relationships
    var sorted []string
    visited := make(map[string]bool)
    temp := make(map[string]bool) // For cycle detection
    
    var visit func(cardID string) error
    visit = func(cardID string) error {
        if temp[cardID] {
            return fmt.Errorf("cycle detected in prerequisites")
        }
        
        if visited[cardID] {
            return nil
        }
        
        temp[cardID] = true
        
        prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
        for _, prereqID := range prereqIDs {
            if err := visit(prereqID); err != nil {
                return err
            }
        }
        
        temp[cardID] = false
        visited[cardID] = true
        sorted = append(sorted, cardID)
        
        return nil
    }
    
    // Visit each card in the prereq map
    for cardID := range prereqMap {
        if err := visit(cardID); err != nil {
            return nil, err
        }
    }
    
    // Convert IDs to cards
    result := make([]Card, 0, len(sorted))
    for _, cardID := range sorted {
        card, err := s.storage.GetCard(cardID)
        if err != nil {
            continue
        }
        result = append(result, card)
    }
    
    return result, nil
}
```

## 10. Storage Implementation Extensions

The existing JSON file storage needs to be extended to support the new graph data structures:

```go
// Initialize storage with graph structures
func (fs *FileStorage) initializeGraph() {
    // Initialize card graph if nil
    if fs.store.CardGraph.Prerequisites == nil {
        fs.store.CardGraph.Prerequisites = make(map[string][]string)
    }
    if fs.store.CardGraph.Dependents == nil {
        fs.store.CardGraph.Dependents = make(map[string][]string)
    }
    
    // Initialize concept graph if nil
    if fs.store.ConceptGraph.Prerequisites == nil {
        fs.store.ConceptGraph.Prerequisites = make(map[string][]string)
    }
    if fs.store.ConceptGraph.Dependents == nil {
        fs.store.ConceptGraph.Dependents = make(map[string][]string)
    }
    
    // Initialize card-concept relationships if nil
    if fs.store.Relationships.ConceptToCards == nil {
        fs.store.Relationships.ConceptToCards = make(map[string][]string)
    }
    if fs.store.Relationships.CardToConcepts == nil {
        fs.store.Relationships.CardToConcepts = make(map[string][]string)
    }
    
    // Initialize concept map if nil
    if fs.store.Concepts == nil {
        fs.store.Concepts = make(map[string]ConceptNode)
    }
    
    // Ensure all cards have ConsecutiveFails field initialized
    for id, card := range fs.store.Cards {
        if card.ConsecutiveFails < 0 {
            card.ConsecutiveFails = 0
            fs.store.Cards[id] = card
        }
    }
}

// Enhanced Load method to initialize graph structures
func (fs *FileStorage) Load() error {
    // Original load implementation...
    
    // After loading, initialize graph structures
    fs.initializeGraph()
    
    return nil
}
```

## 11. Implementation Plan

The implementation will be divided into four phases, corresponding to the four main features:

### Phase 1: Basic Directed Graph Structure

1. Define graph data structures in storage.go
2. Extend FlashcardStore to include graph structures
3. Implement basic graph operations: add/remove nodes and edges
4. Update Load/Save methods to handle graph data
5. Add tests for graph operations

### Phase 2: Card Linking and Prerequisite Filtering

1. Implement CheckPrerequisitesMastered function
2. Enhance GetDueCard to filter by prerequisites
3. Implement MCP tools for creating and managing prerequisites
4. Add handlers for the new tools
5. Test prerequisite filtering

### Phase 3: Consecutive Failure Detection

1. Extend Card structure with ConsecutiveFails field
2. Implement SubmitReviewWithFailureTracking
3. Update MCP handler for submit_review to return teaching mode flag
4. Add tests for failure tracking

### Phase 4: Problem Decomposition

1. Implement concept management functions
2. Add MCP tools for analyzing knowledge gaps
3. Implement handlers for problem decomposition
4. Add LLM instructions for the decomposition process
5. Test the end-to-end teaching mode workflow

## 12. Testing Strategy

### 12.1 Unit Tests

Write unit tests for:

1. Graph operations (add/remove nodes and edges, cycle detection)
2. Prerequisite checking
3. Card selection with prerequisites
4. Failure tracking
5. Problem decomposition

### 12.2 Integration Tests

Test the integration between components:

1. Graph operations with storage
2. MCP tools with service methods
3. End-to-end workflows for teaching mode

### 12.3 LLM Interaction Tests

Test the LLM's ability to:

1. Recognize when to enter teaching mode
2. Decompose problems into prerequisites
3. Create appropriate flashcards
4. Link them correctly in the graph

## 13. Conclusion

This technical design provides a comprehensive blueprint for extending the Flashcards MCP with graph-based learning features. The implementation leverages the existing codebase while adding sophisticated knowledge representation and adaptive learning capabilities.

The directed graph structure enables prerequisite-based filtering, ensuring students master foundational concepts before tackling advanced ones. The failure detection and problem decomposition features create a personalized learning experience that adapts to each student's needs.

These enhancements transform the system from a simple flashcard tool into an intelligent learning system that understands knowledge dependencies and can dynamically adapt to help students overcome learning obstacles.
        
        prereqMap[cardID] = true
        
        prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
        for _, prereqID := range prereqIDs {
            collectPrereqs(prereqID)
        }
    }
    
    // Start from the target card
    collectPrereqs(targetCardID)
    
    // Create a topologically sorted list using known relationships
    var sorted []string
    visited := make(map[string]bool)
    temp := make(map[string]bool) // For cycle detection
    
    var visit func(cardID string) error
    visit = func(cardID string) error {
        if temp[cardID] {
            return fmt.Errorf("cycle detected in prerequisites")
        }
        
        if visited[cardID] {
            return nil
        }
        
        temp[cardID] = true
        
        prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
        for _, prereqID := range prereqIDs {
            if err := visit(prereqID); err != nil {
                return err
            }
        }
        
        temp[cardID] = false
        visited[cardID] = true
        sorted = append(sorted, cardID)
        
        return nil
    }
    
    // Visit each card in the prereq map
    for cardID := range prereqMap {
        if err := visit(cardID); err != nil {
            return nil, err
        }
    }
    
    // Convert IDs to cards
    result := make([]Card, 0, len(sorted))
    for _, cardID := range sorted {
        card, err := s.storage.GetCard(cardID)
        if err != nil {
            continue
        }
        result = append(result, card)
    }
    
    return result, nil
}
```

## 10. Storage Implementation Extensions

The existing JSON file storage needs to be extended to support the new graph data structures:

```go
// Initialize storage with graph structures
func (fs *FileStorage) initializeGraph() {
    // Initialize card graph if nil
    if fs.store.CardGraph.Prerequisites == nil {
        fs.store.CardGraph.Prerequisites = make(map[string][]string)
    }
    if fs.store.CardGraph.Dependents == nil {
        fs.store.CardGraph.Dependents = make(map[string][]string)
    }
    
    // Initialize concept graph if nil
    if fs.store.ConceptGraph.Prerequisites == nil {
        fs.store.ConceptGraph.Prerequisites = make(map[string][]string)
    }
    if fs.store.ConceptGraph.Dependents == nil {
        fs.store.ConceptGraph.Dependents = make(map[string][]string)
    }
    
    // Initialize card-concept relationships if nil
    if fs.store.Relationships.ConceptToCards == nil {
        fs.store.Relationships.ConceptToCards = make(map[string][]string)
    }
    if fs.store.Relationships.CardToConcepts == nil {
        fs.store.Relationships.CardToConcepts = make(map[string][]string)
    }
    
    // Initialize concept map if nil
    if fs.store.Concepts == nil {
        fs.store.Concepts = make(map[string]ConceptNode)
    }
    
    // Ensure all cards have ConsecutiveFails field initialized
    for id, card := range fs.store.Cards {
        if card.ConsecutiveFails < 0 {
            card.ConsecutiveFails = 0
            fs.store.Cards[id] = card
        }
    }
}

// Enhanced Load method to initialize graph structures
func (fs *FileStorage) Load() error {
    // Original load implementation...
    
    // After loading, initialize graph structures
    fs.initializeGraph()
    
    return nil
}
```

## 11. Implementation Plan

The implementation will be divided into four phases, corresponding to the four main features:

### Phase 1: Basic Directed Graph Structure

1. Define graph data structures in storage.go
2. Extend FlashcardStore to include graph structures
3. Implement basic graph operations: add/remove nodes and edges
4. Update Load/Save methods to handle graph data
5. Add tests for graph operations

### Phase 2: Card Linking and Prerequisite Filtering

1. Implement CheckPrerequisitesMastered function
2. Enhance GetDueCard to filter by prerequisites
3. Implement MCP tools for creating and managing prerequisites
4. Add handlers for the new tools
5. Test prerequisite filtering

### Phase 3: Consecutive Failure Detection

1. Extend Card structure with ConsecutiveFails field
2. Implement SubmitReviewWithFailureTracking
3. Update MCP handler for submit_review to return teaching mode flag
4. Add tests for failure tracking

### Phase 4: Problem Decomposition

1. Implement concept management functions
2. Add MCP tools for analyzing knowledge gaps
3. Implement handlers for problem decomposition
4. Add LLM instructions for the decomposition process
5. Test the end-to-end teaching mode workflow

## 12. Testing Strategy

### 12.1 Unit Tests

Write unit tests for:

1. Graph operations (add/remove nodes and edges, cycle detection)
2. Prerequisite checking
3. Card selection with prerequisites
4. Failure tracking
5. Problem decomposition

### 12.2 Integration Tests

Test the integration between components:

1. Graph operations with storage
2. MCP tools with service methods
3. End-to-end workflows for teaching mode

### 12.3 LLM Interaction Tests

Test the LLM's ability to:

1. Recognize when to enter teaching mode
2. Decompose problems into prerequisites
3. Create appropriate flashcards
4. Link them correctly in the graph

## 13. Conclusion

This technical design provides a comprehensive blueprint for extending the Flashcards MCP with graph-based learning features. The implementation leverages the existing codebase while adding sophisticated knowledge representation and adaptive learning capabilities.

The directed graph structure enables prerequisite-based filtering, ensuring students master foundational concepts before tackling advanced ones. The failure detection and problem decomposition features create a personalized learning experience that adapts to each student's needs.

These enhancements transform the system from a simple flashcard tool into an intelligent learning system that understands knowledge dependencies and can dynamically adapt to help students overcome learning obstacles.
Write unit tests for:

1. Graph operations (add/remove nodes and edges, cycle detection)
2. Prerequisite checking
3. Card selection with prerequisites
4. Failure tracking
5. Problem decomposition

### 12.2 Integration Tests

Test the integration between components:

1. Graph operations with storage
2. MCP tools with service methods
3. End-to-end workflows for teaching mode

### 12.3 LLM Interaction Tests

Test the LLM's ability to:

1. Recognize when to enter teaching mode
2. Decompose problems into prerequisites
3. Create appropriate flashcards
4. Link them correctly in the graph

## 13. Conclusion

This technical design provides a comprehensive blueprint for extending the Flashcards MCP with graph-based learning features. The implementation leverages the existing codebase while adding sophisticated knowledge representation and adaptive learning capabilities.

The directed graph structure enables prerequisite-based filtering, ensuring students master foundational concepts before tackling advanced ones. The failure detection and problem decomposition features create a personalized learning experience that adapts to each student's needs.

These enhancements transform the system from a simple flashcard tool into an intelligent learning system that understands knowledge dependencies and can dynamically adapt to help students overcome learning obstacles.
        
        prereqMap[cardID] = true
        
        prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
        for _, prereqID := range prereqIDs {
            collectPrereqs(prereqID)
        }
    }
    
    // Start from the target card
    collectPrereqs(targetCardID)
    
    // Create a topologically sorted list using known relationships
    var sorted []string
    visited := make(map[string]bool)
    temp := make(map[string]bool) // For cycle detection
    
    var visit func(cardID string) error
    visit = func(cardID string) error {
        if temp[cardID] {
            return fmt.Errorf("cycle detected in prerequisites")
        }
        
        if visited[cardID] {
            return nil
        }
        
        temp[cardID] = true
        
        prereqIDs := s.storage.store.CardGraph.Prerequisites[cardID]
        for _, prereqID := range prereqIDs {
            if err := visit(prereqID); err != nil {
                return err
            }
        }
        
        temp[cardID] = false
        visited[cardID] = true
        sorted = append(sorted, cardID)
        
        return nil
    }
    
    // Visit each card in the prereq map
    for cardID := range prereqMap {
        if err := visit(cardID); err != nil {
            return nil, err
        }
    }
    
    // Convert IDs to cards
    result := make([]Card, 0, len(sorted))
    for _, cardID := range sorted {
        card, err := s.storage.GetCard(cardID)
        if err != nil {
            continue
        }
        result = append(result, card)
    }
    
    return result, nil
}
```

## 10. Storage Implementation Extensions

The existing JSON file storage needs to be extended to support the new graph data structures:

```go
// Initialize storage with graph structures
func (fs *FileStorage) initializeGraph() {
    // Initialize card graph if nil
    if fs.store.CardGraph.Prerequisites == nil {
        fs.store.CardGraph.Prerequisites = make(map[string][]string)
    }
    if fs.store.CardGraph.Dependents == nil {
        fs.store.CardGraph.Dependents = make(map[string][]string)
    }
    
    // Initialize concept graph if nil
    if fs.store.ConceptGraph.Prerequisites == nil {
        fs.store.ConceptGraph.Prerequisites = make(map[string][]string)
    }
    if fs.store.ConceptGraph.Dependents == nil {
        fs.store.ConceptGraph.Dependents = make(map[string][]string)
    }
    
    // Initialize card-concept relationships if nil
    if fs.store.Relationships.ConceptToCards == nil {
        fs.store.Relationships.ConceptToCards = make(map[string][]string)
    }
    if fs.store.Relationships.CardToConcepts == nil {
        fs.store.Relationships.CardToConcepts = make(map[string][]string)
    }
    
    // Initialize concept map if nil
    if fs.store.Concepts == nil {
        fs.store.Concepts = make(map[string]ConceptNode)
    }
    
    // Ensure all cards have ConsecutiveFails field initialized
    for id, card := range fs.store.Cards {
        if card.ConsecutiveFails < 0 {
            card.ConsecutiveFails = 0
            fs.store.Cards[id] = card
        }
    }
}

// Enhanced Load method to initialize graph structures
func (fs *FileStorage) Load() error {
    // Original load implementation...
    
    // After loading, initialize graph structures
    fs.initializeGraph()
    
    return nil
}
```

## 11. Implementation Plan

The implementation will be divided into four phases, corresponding to the four main features:

### Phase 1: Basic Directed Graph Structure

1. Define graph data structures in storage.go
2. Extend FlashcardStore to include graph structures
3. Implement basic graph operations: add/remove nodes and edges
4. Update Load/Save methods to handle graph data
5. Add tests for graph operations

### Phase 2: Card Linking and Prerequisite Filtering

1. Implement CheckPrerequisitesMastered function
2. Enhance GetDueCard to filter by prerequisites
3. Implement MCP tools for creating and managing prerequisites
4. Add handlers for the new tools
5. Test prerequisite filtering

### Phase 3: Consecutive Failure Detection

1. Extend Card structure with ConsecutiveFails field
2. Implement SubmitReviewWithFailureTracking
3. Update MCP handler for submit_review to return teaching mode flag
4. Add tests for failure tracking

### Phase 4: Problem Decomposition

1. Implement concept management functions
2. Add MCP tools for analyzing knowledge gaps
3. Implement handlers for problem decomposition
4. Add LLM instructions for the decomposition process
5. Test the end-to-end teaching mode workflow

## 12. Testing Strategy

### 12.1 Unit Tests

Write unit tests for:

1. Graph operations (add/remove nodes and edges, cycle detection)
2. Prerequisite checking
3. Card selection with prerequisites
4. Failure tracking
5. Problem decomposition

### 12.2 Integration Tests

Test the integration between components:

1. Graph operations with storage
2. MCP tools with service methods
3. End-to-end workflows for teaching mode

### 12.3 LLM Interaction Tests

Test the LLM's ability to:

1. Recognize when to enter teaching mode
2. Decompose problems into prerequisites
3. Create appropriate flashcards
4. Link them correctly in the graph

## 13. Conclusion

This technical design provides a comprehensive blueprint for extending the Flashcards MCP with graph-based learning features. The implementation leverages the existing codebase while adding sophisticated knowledge representation and adaptive learning capabilities.

The directed graph structure enables prerequisite-based filtering, ensuring students master foundational concepts before tackling advanced ones. The failure detection and problem decomposition features create a personalized learning experience that adapts to each student's needs.

These enhancements transform the system from a simple flashcard tool into an intelligent learning system that understands knowledge dependencies and can dynamically adapt to help students overcome learning obstacles.