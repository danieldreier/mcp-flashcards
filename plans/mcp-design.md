# Flashcards MCP - Technical Design Document

## 1. Overview

This document outlines the technical specification for a Flashcards Model Context Protocol (MCP) server implemented in Go. The system will:

- Provide a server for managing spaced repetition flashcards
- Utilize the go-fsrs library to implement the Free Spaced Repetition Scheduler algorithm
- Store cards and review history in a JSON file
- Expose tools via MCP for LLM integration
- Follow the MCP pattern demonstrated in the calculator example

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
    "time"
    
    "github.com/google/uuid"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    "github.com/open-spaced-repetition/go-fsrs"
)
```

## 3. Data Structures

### 3.1 Core Data Types

```go
// FlashcardStore represents the data persisted to file
type FlashcardStore struct {
    Cards        map[string]Card     `json:"cards"`
    Reviews      []ReviewRecord      `json:"reviews"`
    LastModified time.Time           `json:"last_modified"`
}

// Card represents a flashcard
type Card struct {
    ID           string       `json:"id"`
    Front        string       `json:"front"`
    Back         string       `json:"back"`
    CreatedAt    time.Time    `json:"created_at"`
    Due          time.Time    `json:"due"`
    State        fsrs.State   `json:"state"`  
    Tags         []string     `json:"tags,omitempty"`
}

// ReviewRecord stores information about a review session
type ReviewRecord struct {
    ID        string       `json:"id"`
    CardID    string       `json:"card_id"`
    Rating    fsrs.Rating  `json:"rating"`
    Timestamp time.Time    `json:"timestamp"`
    Answer    string       `json:"answer,omitempty"`
}

// CardStats represents statistics for flashcard review
type CardStats struct {
    TotalCards     int     `json:"total_cards"`
    DueCards       int     `json:"due_cards"`
    ReviewsToday   int     `json:"reviews_today"`
    RetentionRate  float64 `json:"retention_rate"`
}
```

### 3.2 Service Interface

```go
// FlashcardService defines the operations for managing flashcards
type FlashcardService interface {
    // Card operations
    CreateCard(front, back string, tags []string) (Card, error)
    GetCard(id string) (Card, error)
    UpdateCard(card Card) error
    DeleteCard(id string) error
    ListCards() ([]Card, error)
    
    // Review operations
    GetDueCard() (Card, CardStats, error)
    SubmitReview(cardID string, rating fsrs.Rating, answer string) error
    
    // Persistence
    LoadFromFile() error
    SaveToFile() error
}
```

## 4. MCP Server Specification

### 4.1 Server Configuration

```go
// Create and configure the MCP server
s := server.NewMCPServer(
    "Flashcards MCP",
    "1.0.0",
    server.WithResourceCapabilities(true, true),
    server.WithLogging(),
)
```

### 4.2 Tool Definitions

#### Get Due Card Tool

```go
getDueCardTool := mcp.NewTool("get_due_card",
    mcp.WithDescription("Get the next flashcard due for review with statistics"),
)
```

#### Submit Review Tool

```go
submitReviewTool := mcp.NewTool("submit_review",
    mcp.WithDescription("Submit a review rating for a flashcard"),
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

#### Create Card Tool

```go
createCardTool := mcp.NewTool("create_card",
    mcp.WithDescription("Create a new flashcard"),
    mcp.WithString("front",
        mcp.Required(),
        mcp.Description("Front side of the flashcard (question)"),
    ),
    mcp.WithString("back",
        mcp.Required(),
        mcp.Description("Back side of the flashcard (answer)"),
    ),
    mcp.WithArray("tags",
        mcp.Description("List of tags for categorizing the flashcard"),
        mcp.Items(mcp.String()),
    ),
)
```

#### Update Card Tool

```go
updateCardTool := mcp.NewTool("update_card",
    mcp.WithDescription("Update an existing flashcard"),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card to update"),
    ),
    mcp.WithString("front",
        mcp.Description("New front side of the flashcard (question)"),
    ),
    mcp.WithString("back",
        mcp.Description("New back side of the flashcard (answer)"),
    ),
    mcp.WithArray("tags",
        mcp.Description("New list of tags for the flashcard"),
        mcp.Items(mcp.String()),
    ),
)
```

#### Delete Card Tool

```go
deleteCardTool := mcp.NewTool("delete_card",
    mcp.WithDescription("Delete a flashcard"),
    mcp.WithString("card_id",
        mcp.Required(),
        mcp.Description("ID of the card to delete"),
    ),
)
```

#### List Cards Tool

```go
listCardsTool := mcp.NewTool("list_cards",
    mcp.WithDescription("List all flashcards"),
    mcp.WithArray("tags",
        mcp.Description("Filter cards by these tags (optional)"),
        mcp.Items(mcp.String()),
    ),
    mcp.WithBoolean("include_stats",
        mcp.Description("Include statistics in the response"),
    ),
)
```

### 4.3 Handler Function Signatures

```go
// GetDueCardHandler handles get_due_card requests
func GetDueCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Return the next card due for review with statistics
}

// SubmitReviewHandler handles submit_review requests
func SubmitReviewHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Process and store flashcard review data
}

// CreateCardHandler handles create_card requests
func CreateCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Create a new flashcard
}

// UpdateCardHandler handles update_card requests
func UpdateCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Update an existing flashcard
}

// DeleteCardHandler handles delete_card requests
func DeleteCardHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Delete a flashcard
}

// ListCardsHandler handles list_cards requests
func ListCardsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Return a list of flashcards, optionally filtered by tags
}
```

## 5. Storage Implementation

### 5.1 File Operations

```go
// loadFromFile loads flashcard data from a JSON file
func (s *FlashcardServiceImpl) LoadFromFile() error {
    // Load and deserialize JSON data from file
    // Initialize empty data structure if file doesn't exist
}

// saveToFile saves flashcard data to a JSON file
func (s *FlashcardServiceImpl) SaveToFile() error {
    // Serialize and save data to file with atomic write
}
```

## 6. FSRS Integration

### 6.1 FSRS Manager

```go
// FSRSManager handles the spaced repetition scheduling algorithm
type FSRSManager struct {
    params fsrs.Parameters
}

// NewFSRSManager creates a new FSRS manager with default parameters
func NewFSRSManager() *FSRSManager {
    return &FSRSManager{
        params: fsrs.DefaultParam(),
    }
}

// ScheduleReview calculates the next review time based on the rating
func (m *FSRSManager) ScheduleReview(card Card, rating fsrs.Rating) (fsrs.State, time.Time) {
    // Use the FSRS algorithm to determine the next review time
    // Return the updated state and next due date
}
```

### 6.2 Card Selection Algorithm

```go
// getDueCard selects the next card for review
func (s *FlashcardServiceImpl) GetDueCard() (Card, CardStats, error) {
    // Select the next due card based on due date and priority
    // Calculate and return card statistics
}
```

## 7. Main Application

```go
func main() {
    // Parse command-line flags for file path
    filePath := flag.String("file", "./flashcards.json", "Path to flashcard data file")
    flag.Parse()
    
    // Initialize flashcard service
    service := NewFlashcardService(*filePath)
    if err := service.LoadFromFile(); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load flashcard data: %v\n", err)
        os.Exit(1)
    }
    
    // Create MCP server
    s := server.NewMCPServer(
        "Flashcards MCP",
        "1.0.0",
        server.WithResourceCapabilities(true, true),
        server.WithLogging(),
    )
    
    // Add and register tools
    // ...
    
    // Start the server
    if err := server.ServeStdio(s); err != nil {
        fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
    }
}
```

## 8. Testing Strategy

### 8.1 Test File Structure

```go
// TestFlashcardsMCP tests the flashcards MCP functionality
func TestFlashcardsMCP(t *testing.T) {
    // Get the path to the flashcards binary
    flashcardsPath := filepath.Join(os.TempDir(), "flashcards")
    
    // Create a client that connects to the flashcards server
    c, err := client.NewStdioMCPClient(
        "go",
        []string{}, // Empty ENV
        "run",
        ".",  // Current directory
    )
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer c.Close()
    
    // Initialize the client
    // Test card creation
    // Test getting due cards
    // Test submitting reviews
    // Test scheduling algorithm
}
```

### 8.2 Test Cases

The test suite should include tests for:

1. Creating, updating, and deleting flashcards
2. Getting due cards and proper ordering
3. Submitting reviews with different ratings
4. Verifying the FSRS algorithm's scheduling
5. Persistence of data between sessions
6. Error handling for invalid inputs
7. Concurrent operations

## 9. Implementation Files and Organization

The implementation will be organized into the following files:

1. **cmd/flashcards/main.go**: Main application entry point
2. **cmd/flashcards/service.go**: FlashcardService implementation
3. **cmd/flashcards/fsrs.go**: FSRS algorithm integration
4. **cmd/flashcards/storage.go**: JSON file persistence
5. **cmd/flashcards/handlers.go**: MCP tool handlers
6. **cmd/flashcards/main_test.go**: Integration tests

## 10. LLM Integration Guidelines

When using this MCP server with an LLM:

1. The LLM should use the `get_due_card` tool to retrieve the next card to review
2. Present the front of the card to the user
3. After the user provides an answer, compare it with the back of the card
4. Use the `submit_review` tool with an appropriate rating:
   - Rating 1 (Again): User couldn't recall or answered incorrectly
   - Rating 2 (Hard): User recalled with significant difficulty
   - Rating 3 (Good): User recalled correctly with some effort
   - Rating 4 (Easy): User recalled easily and quickly
5. Provide feedback to the user based on their answer accuracy
6. Continue with the next due card

## 11. Future Enhancements

Potential future enhancements include:

1. Custom scheduling parameters
2. Bulk import/export of flashcards
3. Card organization with decks in addition to tags
4. Media attachment support for cards
5. Progress tracking and analytics
