# Flashcards MCP Refactoring Plan

This document outlines a detailed step-by-step plan for refactoring the Flashcards MCP main.go file into more maintainable units using a horizontal slicing approach.

## Background and Context

The current `main.go` file has grown too large and contains several distinct responsibilities:
1. Data model definitions (Card, CardStats, etc.)
2. MCP tool handler functions
3. FlashcardService implementation with business logic
4. Server setup and initialization

This refactoring will separate these concerns into dedicated files, making the codebase more maintainable while preserving the existing functionality and ensuring tests continue to pass.

## Refactoring Approach

After evaluating several options, we've chosen a horizontal slicing approach that organizes code by layer of responsibility:

- `models.go` - Data structures
- `service.go` - Business logic in FlashcardService
- `handlers.go` - MCP tool handlers
- `main.go` - Server setup and initialization

This approach provides a clean separation of concerns that aligns with Go best practices, makes the code more testable, and follows patterns common in Go applications.

## Task 1: Extract Data Models to models.go

### Background and Context
The first step is to move all data structure definitions to a dedicated models file, establishing the foundation for the rest of the refactoring.

### My Task
Create `models.go` and move all struct definitions from `main.go`:

1. Create a new file `models.go` in the same package
2. Move all data structure definitions
3. Add proper documentation
4. Run tests to ensure functionality is preserved

### Files to Create/Modify
1. Create: `cmd/flashcards/models.go`
2. Modify: `cmd/flashcards/main.go` (remove moved structures)

### Implementation Details

Move the following structs to `models.go`:
- Card
- CardStats
- CardResponse
- ReviewResponse
- CreateCardResponse
- UpdateCardResponse
- DeleteCardResponse
- ListCardsResponse

```go
// models.go
package main

import (
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
)

// Card represents a flashcard with content and FSRS algorithm data
type Card struct {
	ID        string     `json:"id"`
	Front     string     `json:"front"`
	Back      string     `json:"back"`
	CreatedAt time.Time  `json:"created_at"`
	Tags      []string   `json:"tags,omitempty"`
	// FSRS algorithm data
	FSRS      fsrs.Card  `json:"fsrs"`
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

// Other struct definitions...
```

### Success Criteria
- [x] All struct definitions are moved to models.go
- [x] Imports are correctly set up
- [x] Documentation is comprehensive
- [x] Tests pass after this change

## Task 2: Extract Service Layer to service.go

### Background and Context
The service layer contains the business logic for managing flashcards. Moving it to a dedicated file will improve organization and testability.

### My Task
Create `service.go` and move the FlashcardService interface and implementation:

1. Create a new file `service.go` in the same package
2. Move the FlashcardService interface definition
3. Move the FlashcardService implementation and all methods
4. Add proper documentation
5. Run tests to ensure functionality is preserved

### Files to Create/Modify
1. Create: `cmd/flashcards/service.go`
2. Modify: `cmd/flashcards/main.go` (remove moved components)

### Implementation Details

The service.go file should contain:
- FlashcardService interface definition
- FlashcardServiceImpl struct
- NewFlashcardService constructor
- All service methods (GetDueCard, SubmitReview, UpdateCard, etc.)

```go
// service.go
package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
	"github.com/danieldreier/mcp-flashcards/internal/storage"
)

// FlashcardService provides methods for managing flashcards
type FlashcardService interface {
	// GetDueCard returns the next card due for review with statistics
	GetDueCard() (Card, CardStats, error)
	
	// SubmitReview processes a review of a card
	SubmitReview(cardID string, rating fsrs.Rating, answer string) (Card, error)
	
	// Additional methods...
}

// FlashcardServiceImpl implements the FlashcardService interface
type FlashcardServiceImpl struct {
	storage     storage.Storage
	fsrsManager fsrs.FSRSManager
}

// NewFlashcardService creates a new flashcard service
func NewFlashcardService(storage storage.Storage, fsrsManager fsrs.FSRSManager) FlashcardService {
	return &FlashcardServiceImpl{
		storage:     storage,
		fsrsManager: fsrsManager,
	}
}

// Implement all service methods...
```

### Success Criteria
- [x] FlashcardService interface and implementation are moved to service.go
- [x] All service methods are implemented correctly
- [x] Constructor function is properly defined
- [x] Documentation is comprehensive
- [x] Tests pass after this change

## Task 3: Extract Handlers to handlers.go

### Background and Context
MCP tool handlers are responsible for processing requests and calling the service layer. Moving them to a dedicated file will improve organization.

### My Task
Create `handlers.go` and move all handler functions:

1. Create a new file `handlers.go` in the same package
2. Move all handler functions (handleGetDueCard, handleSubmitReview, etc.)
3. Add proper documentation
4. Run tests to ensure functionality is preserved

### Files to Create/Modify
1. Create: `cmd/flashcards/handlers.go`
2. Modify: `cmd/flashcards/main.go` (remove moved functions)

### Implementation Details

Move the following handler functions to handlers.go:
- handleGetDueCard
- handleSubmitReview
- handleCreateCard
- handleUpdateCard
- handleDeleteCard
- handleListCards

```go
// handlers.go
package main

import (
	"context"
	"fmt"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
	"github.com/modelcontextprotocol/mcp-go/mcp"
	"github.com/modelcontextprotocol/mcp-go/server"
)

// handleGetDueCard handles requests for the next card due for review
func handleGetDueCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract service from context
	s := server.ServerFromContext(ctx)
	service, ok := s.UserData.(FlashcardService)
	if !ok {
		return nil, fmt.Errorf("failed to get flashcard service from context")
	}
	
	// Existing implementation...
}

// Other handler functions...
```

### Success Criteria
- [  ] All handler functions are moved to handlers.go
- [  ] Imports are correctly set up
- [  ] Functions access the service layer correctly
- [  ] Documentation is comprehensive
- [  ] Tests pass after this change

## Task 4: Simplify main.go

### Background and Context
After moving all components to dedicated files, main.go should be simplified to focus only on server setup and initialization.

### My Task
Refactor `main.go` to use the components from the other files:

1. Remove the moved code
2. Update imports to reference the components from other files
3. Keep only the server setup and tool registration logic
4. Run tests to ensure full system functionality

### Files to Create/Modify
1. Modify: `cmd/flashcards/main.go`

### Implementation Details

The simplified main.go should:
- Import the necessary packages
- Define tool definitions 
- Set up and configure the MCP server
- Register the tools with their respective handlers
- Start the server

```go
// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/danieldreier/mcp-flashcards/internal/fsrs"
	"github.com/danieldreier/mcp-flashcards/internal/storage"
	"github.com/modelcontextprotocol/mcp-go/mcp"
	"github.com/modelcontextprotocol/mcp-go/server"
)

func main() {
	// Parse command-line flags
	filePath := flag.String("file", "./flashcards.json", "Path to flashcard data file")
	flag.Parse()
	
	// Initialize storage and FSRS manager
	storageImpl := storage.NewFileStorage(*filePath)
	if err := storageImpl.Load(); err != nil {
		fmt.Printf("Error loading storage: %v\n", err)
		os.Exit(1)
	}
	
	fsrsManager := fsrs.NewFSRSManager()
	
	// Create and configure MCP server
	s := server.NewMCPServer(
		"Flashcards MCP",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)
	
	// Initialize service
	flashcardService := NewFlashcardService(storageImpl, fsrsManager)
	s.UserData = flashcardService
	
	// Define tools
	// Register tools with handlers
	// Start server
}
```

### Success Criteria
- [  ] main.go is focused only on server setup and initialization
- [  ] All moved code has been removed
- [  ] Proper imports reference components in other files
- [  ] Tool registration is correctly implemented
- [  ] Full test suite passes
- [  ] The application functions exactly as before

## Implementation Sequence

To refactor the code safely while maintaining functionality, we'll implement the tasks in this order:

1. Extract data models to models.go
2. Extract service layer to service.go
3. Extract handlers to handlers.go
4. Simplify main.go

After each step, we'll run the tests to ensure the application continues to function correctly.