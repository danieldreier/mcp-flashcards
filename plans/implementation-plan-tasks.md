# Flashcards MCP Implementation Plan

This document outlines a detailed step-by-step plan for implementing the Flashcards MCP, focusing on incremental development with end-to-end testing at each stage.

## Task 1: Implement Minimal MCP Server with `get_due_card` Tool

### Background and Context
We are building a Flashcards MCP server that will manage spaced repetition flashcards. Our first step is to create a minimal server that exposes a single MCP tool (`get_due_card`) with a hardcoded response for integration testing.

We'll follow the pattern established in the calculator example (`cmd/calculator/main.go` and `cmd/calculator/main_test.go`).

### My Task
Create a minimal MCP server that:
1. Exposes a single `get_due_card` tool that returns a hardcoded response
2. Includes end-to-end testing using the STDIO MCP client
3. Follows the same patterns as the calculator example

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Main application with MCP server definition
2. `cmd/flashcards/main_test.go` - Integration tests

### Implementation Details

#### Data Structures
Define these structures in `cmd/flashcards/main.go`:

```go
// Card represents a flashcard with content and FSRS algorithm data
type Card struct {
    ID        string     `json:"id"`
    Front     string     `json:"front"`
    Back      string     `json:"back"`
    CreatedAt time.Time  `json:"created_at"`
    Tags      []string   `json:"tags,omitempty"`
    // Algorithm data - from fsrs.Card which contains:
    // Due, Stability, Difficulty, ElapsedDays, ScheduledDays, Reps, Lapses, State, LastReview
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
```

#### Main Server Implementation Requirements
- Create a new MCP server instance using `server.NewMCPServer` with appropriate server name and version
  ```go
  // From mcp-go/server documentation:
  // func NewMCPServer(name, version string, opts ...ServerOption) *MCPServer
  // From mcp-go/server documentation:
  // func NewMCPServer(name, version string, opts ...ServerOption) *MCPServer
  s := server.NewMCPServer(
      "Flashcards MCP",
      "1.0.0",
      server.WithResourceCapabilities(true, true), // Resource capabilities for subscribe and listChanged
      server.WithLogging(), // Enable logging for the server
  )
  ```
- Define a `get_due_card` tool using an MCP tool definition
- Register a handler function with the correct function signature:
  
  ```go
  // From mcp-go/server documentation:
  // type ToolHandlerFunc func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
  func handleGetDueCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      // Implementation here
      return &mcp.CallToolResult{
          Result: response,
      }, nil
  }
  ```
  ```go
  // From mcp-go/mcp documentation:
  // func NewTool(name string, opts ...ToolOption) Tool
  getDueCardTool := mcp.NewTool("get_due_card",
      mcp.WithDescription("Get the next flashcard due for review"),
      // Additional options like parameters would go here
  )
  ```
- Register the tool with a handler function
  ```go
  // Using ServerTool to combine tool definition with handler
  s.RegisterTool(server.ServerTool{
      Tool:    getDueCardTool,
      Handler: handleGetDueCard,
  })
  ```
- Start the server using `server.ServeStdio`
  ```go
  // From mcp-go/server documentation:
  // func ServeStdio(server *MCPServer, opts ...StdioOption) error
  if err := server.ServeStdio(s); err != nil {
      log.Fatalf("Error serving MCP server: %v", err)
  }
  ```

#### Test Implementation Requirements
- Create a new MCP client using `client.NewStdioMCPClient` connecting to the server
- Initialize the client with appropriate protocol version and client info
- Call the `get_due_card` tool and verify the response structure
- Test that the response contains expected fields (card front/back, stats)

### Success Criteria
- [x] MCP server with a single `get_due_card` tool is implemented
  - Created in cmd/flashcards/main.go with proper server configuration
  - Implemented the get_due_card tool with hardcoded response
- [x] The tool returns a hardcoded response with proper structure
  - Response includes card data (ID, front, back, FSRS data) and statistics
  - Used proper JSON serialization
- [x] Integration test passes successfully
  - Test verifies response structure and content
  - Output: "Successfully got card: What is the capital of France? - Paris"
- [x] Code follows the patterns from the calculator example
  - Used the same server initialization and tool registration pattern
  - Followed similar structure for tests

### Step-by-Step Implementation
1. [x] Create directory `cmd/flashcards` if it doesn't exist
2. [x] Create `cmd/flashcards/main.go` with the MCP server implementation
3. [x] Create `cmd/flashcards/main_test.go` with the integration test
4. [x] Run the test with `go test ./cmd/flashcards -v`
5. [x] Fix any issues that arise during testing
   - Fixed issues with response format to ensure proper text content handling

## Task 2: Add Remaining MCP Tools with Static Responses

### Background and Context
Building on Task 1, we now need to extend our MCP server to include all the tools defined in our design document, still using hardcoded responses. This task completes the MCP interface layer with mock responses.

### My Task
Extend the MCP server to include all tools defined in the design document:
1. `submit_review` - Submit a review for a flashcard
2. `create_card` - Create a new flashcard
3. `update_card` - Update an existing flashcard
4. `delete_card` - Delete a flashcard
5. `list_cards` - List all flashcards

Each tool should return a hardcoded response that matches the expected structure of a real response.

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Add all MCP tools and handlers
2. `cmd/flashcards/main_test.go` - Add tests for all new tools

### Implementation Details

#### Additional Data Structures
Define these additional structures:

```go
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
```

#### Tool Definitions
Define each tool with appropriate parameters:

1. `submit_review` tool:
   - Required string parameter: `card_id`
   - Required number parameter: `rating` (1-4, corresponding to fsrs.Rating constants: Again=1, Hard=2, Good=3, Easy=4)
   - Optional string parameter: `answer`

2. `create_card` tool:
   - Required string parameter: `front`
   - Required string parameter: `back`
   - Optional array parameter: `tags`

3. `update_card` tool:
   - Required string parameter: `card_id`
   - Optional string parameter: `front`
   - Optional string parameter: `back`
   - Optional array parameter: `tags`

4. `delete_card` tool:
   - Required string parameter: `card_id`

5. `list_cards` tool:
   - Optional array parameter: `tags`
   - Optional boolean parameter: `include_stats`

#### Handler Implementation Requirements
Implement handlers for each tool that:
- Extract required and optional parameters from the request
- Create appropriate hardcoded responses
- Return responses using the correct MCP result format

#### Test Implementation Requirements
For each tool, implement test cases that:
- Create appropriate request parameters
- Call the tool
- Verify the response structure
- Check for expected values in the response

### Success Criteria
- [x] All MCP tools defined in the design document are implemented
  - Implemented all 5 required tools: submit_review, create_card, update_card, delete_card, list_cards
  - Each tool has appropriate parameter definitions with required/optional flags
- [x] Each tool handler returns a hardcoded response with the proper structure
  - Created appropriate response structures for each tool
  - Implemented handlers that properly extract parameters and return formatted JSON responses
- [x] Integration tests for all tools pass successfully
  - All tests pass with proper validation of input parameters and response structures
  - Added specific tests for tag filtering in list_cards
- [x] Code follows the patterns from the calculator example
  - Used consistent parameter handling with Arguments map
  - Followed the same tool definition pattern with WithString, WithNumber, etc.

### Step-by-Step Implementation
1. [x] Update `cmd/flashcards/main.go` with all additional MCP tools and handlers
   - Added response structures and handler functions for each tool
   - Implemented parameter validation and appropriate error responses
2. [x] Update `cmd/flashcards/main_test.go` with tests for all tools
   - Created a helper setup function to reduce code duplication
   - Added comprehensive tests for each tool with parameter validation
3. [x] Run the tests with `go test ./cmd/flashcards -v`
   - All tests pass successfully with no errors
4. [x] Fix any issues that arise during testing
   - Fixed parameter access using Arguments instead of Parameters
   - Updated tool definitions to use the correct MCP-go patterns

## Task 3: Implement File System / JSON Storage

### Background and Context
With the MCP interface layer completed, we now need to implement the storage layer for the flashcards data. We'll create a simple file-based JSON storage system that persists card data and review history.

### My Task
Implement a file-based JSON storage system for the flashcards MCP that:
1. Stores flashcards and review history in a JSON file
2. Provides methods to load, save, and manage card data
3. Is unit tested independently of the MCP layer

### Files to Create/Modify
1. `internal/storage/storage.go` - Storage implementation
2. `internal/storage/storage_test.go` - Unit tests for storage

### Implementation Details

#### Data Structures
Define these structures in the storage package:

```go
// Card represents a flashcard in storage
type Card struct {
    ID        string    `json:"id"`
    Front     string    `json:"front"`
    Back      string    `json:"back"`
    CreatedAt time.Time `json:"created_at"`
    Tags      []string  `json:"tags,omitempty"`
    // Using embedded fsrs.Card for algorithm data
    FSRS      fsrs.Card `json:"fsrs"`
}

// Review represents a review record in storage
// Structured to align with fsrs.ReviewLog
type Review struct {
    ID        string      `json:"id"`
    CardID    string      `json:"card_id"`
    Rating    fsrs.Rating `json:"rating"` // Using fsrs.Rating type (Again=1, Hard=2, Good=3, Easy=4)
    Timestamp time.Time   `json:"timestamp"`
    Answer    string      `json:"answer,omitempty"`
    // Additional fields from fsrs.ReviewLog that track scheduling information
    ScheduledDays uint64    `json:"scheduled_days"`
    ElapsedDays   uint64    `json:"elapsed_days"`
    State         fsrs.State `json:"state"`
}

// FlashcardStore represents the data structure stored in the JSON file
type FlashcardStore struct {
    Cards       map[string]Card `json:"cards"`
    Reviews     []Review        `json:"reviews"`
    LastUpdated time.Time       `json:"last_updated"`
}
```

#### Interface Definitions
Define a storage interface with these methods:

```go
// Storage represents the storage interface for flashcards
type Storage interface {
    // Card operations
    CreateCard(front, back string, tags []string) (Card, error)
    GetCard(id string) (Card, error)
    UpdateCard(card Card) error
    DeleteCard(id string) error
    ListCards(tags []string) ([]Card, error)
    
    // Review operations
    // Using fsrs.Rating type for proper integration with algorithm
    AddReview(cardID string, rating fsrs.Rating, answer string) (Review, error)
    GetCardReviews(cardID string) ([]Review, error)
    
    // File operations
    Load() error
    Save() error
}
```

#### Implementation Requirements
Implement a `FileStorage` struct that:
- Stores data in a JSON file
- Uses proper synchronization with a mutex to allow concurrent access
- Properly handles file I/O operations
- Performs atomic file writes (write to temp file, then rename)

The implementation should handle:
- Loading data from an existing file
- Creating a new file if it doesn't exist
- Saving data to the file
- Creating, retrieving, updating, and deleting cards
- Adding and retrieving reviews
- Filtering cards by tags

#### Test Implementation Requirements
Create tests that verify:
- Creating, retrieving, updating, and deleting cards
- Saving and loading data between sessions
- Proper filtering of cards by tags
- Adding and retrieving reviews
- Error handling for non-existent cards

### Success Criteria
- [x] Storage interface and implementation are complete
  - Created `Storage` interface with methods for card operations, review operations, and file operations
  - Implemented `FileStorage` struct that satisfies the interface
- [x] All methods of the Storage interface are properly implemented
  - Implemented all CRUD operations for cards with proper error handling
  - Implemented review management with proper integration with fsrs.Rating
  - Implemented file operations with atomic writes and proper error handling
- [x] File operations properly save and load data
  - Added JSON serialization/deserialization with proper file handling
  - Used atomic file writes (write to temp file, then rename)
  - Handled non-existent files by creating empty storage
- [x] CRUD operations for cards work correctly
  - Implemented CreateCard, GetCard, UpdateCard, DeleteCard, and ListCards
  - Added proper error handling and validation
  - Used UUIDs for card identifiers
- [x] Review operations work correctly
  - Implemented AddReview and GetCardReviews operations
  - Properly tracked review history with timestamps and ratings
  - Used UUIDs for review identifiers
- [x] All tests pass without errors
  - Created comprehensive unit tests for all operations
  - Added tests for edge cases (corrupted files, non-existent files)
  - All tests pass successfully

### Step-by-Step Implementation
1. [x] Create directory `internal/storage` if it doesn't exist
   - Created directory structure for storage implementation
2. [x] Create `internal/storage/storage.go` with the storage interface and implementation
   - Implemented Storage interface with all required methods
   - Created FileStorage struct with proper file operations and concurrency control
3. [x] Create `internal/storage/storage_test.go` with unit tests
   - Created comprehensive tests for all operations
   - Added tests for edge cases and error handling
4. [x] Run the tests with `go test ./internal/storage -v`
   - All tests passed successfully
5. [x] Fix any issues that arise during testing
   - Fixed minor issues with variable declarations in tests

## Task 4: Implement Working Version of Create Card

### Background and Context
Now that we have a storage system, we can connect the MCP server to it and implement a working version of the `create_card` tool that actually persists data to the storage.

### My Task
Update the MCP server to use the storage system for the `create_card` tool. This will:
1. Connect the MCP server to the storage system
2. Implement a working version of the `create_card` tool
3. Update the tests to verify actual card creation

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Update to use storage system
2. `cmd/flashcards/main_test.go` - Update tests for create_card

### Implementation Details

#### Updated Main Implementation Requirements
Modify the main function to:
- Initialize the storage system with a file path
- Pass the storage instance to the MCP tool handlers
- Update the `create_card` handler to use the storage system instead of returning hardcoded responses
- Keep other tools with hardcoded responses for now

```go
func main() {
    // Parse command-line flags
    filePath := flag.String("file", "./flashcards.json", "Path to flashcard data file")
    flag.Parse()
    
    // Initialize storage
    storage := storage.NewFileStorage(*filePath)
    if err := storage.Load(); err != nil {
        fmt.Printf("Error loading storage: %v\n", err)
        os.Exit(1)
    }
    
    // Create MCP server
    s := server.NewMCPServer(
        "Flashcards MCP",
        "1.0.0",
        server.WithResourceCapabilities(true, true),
        server.WithToolCapabilities(true), // Add tool capabilities with listChanged
        server.WithLogging(),
    )
    
    // Initialize and store the service in the server's UserData for context retrieval
    flashcardService := NewFlashcardService(storage, fsrsManager)
    s.UserData = flashcardService
    
    // Add tools
    // ...
}
```

#### Create Card Handler Requirements
Update the `create_card` handler to:
- Extract parameters from the request
- Call the storage.CreateCard method
- Return the actual created card
- Handle errors appropriately

#### Test Implementation Requirements
Update the create_card test to:
- Create a temporary file for testing
- Initialize the storage with the temporary file
- Verify that the created card is actually stored
- Clean up the temporary file after testing

### Success Criteria
- [x] MCP server initializes the storage system correctly
  - Implemented command-line flag for storage file path
  - Added proper error handling for storage initialization
- [x] `create_card` tool uses the storage system to create and persist cards
  - Updated handler to call storage.CreateCard with proper parameters
  - Added error handling for storage operations
- [x] Integration tests verify that cards are actually created and stored
  - Created tests that check if cards are properly persisted to file
  - Verified card data in the storage file after creation
- [x] The file is properly saved and can be loaded again
  - Implemented proper file handling with atomic writes
  - Added JSON parsing verification in tests

### Step-by-Step Implementation
1. [x] Update `cmd/flashcards/main.go` to initialize and use the storage system
   - Added command-line flag for specifying storage file path
   - Implemented proper storage initialization and error handling
2. [x] Update the `create_card` handler to use the storage system
   - Updated handler to extract parameters and call storage methods
   - Added proper error handling for storage operations
3. [x] Update `cmd/flashcards/main_test.go` to test actual card creation
   - Added verification that cards are persisted to the storage file
   - Implemented proper file cleanup after tests
4. [x] Run the tests with `go test ./cmd/flashcards -v`
   - All tests passed successfully with proper storage integration
5. [x] Create a test dataset of cards using the tool
   - Created test cards with various content and tags

## Task 5: Implement FSRS Manager

### Background and Context
The Free Spaced Repetition Scheduler (FSRS) algorithm is a key component of our flashcard system. It determines when cards should be reviewed again based on the user's performance.

### My Task
Implement an FSRS manager that:
1. Integrates with the go-fsrs library
2. Handles scheduling of card reviews
3. Is unit tested independently

### Files to Create/Modify
1. `internal/fsrs/fsrs.go` - FSRS manager implementation
2. `internal/fsrs/fsrs_test.go` - Unit tests for FSRS manager

### Implementation Details

#### Interface Definition
Define an FSRS manager interface:

```go
// FSRSManager defines the interface for scheduling flashcards using the FSRS algorithm
type FSRSManager interface {
    // ScheduleReview calculates the next review time based on the rating
    // From go-fsrs documentation:
    // type Rating int8
    // const Again Rating = iota + 1 // (1)
    // const Hard, Good, Easy Rating = 2, 3, 4
    //
    // type State int8
    // const New State = iota // (0)
    // const Learning, Review, Relearning State = 1, 2, 3
    ScheduleReview(state fsrs.State, rating fsrs.Rating, now time.Time) (fsrs.State, time.Time)
    
    // GetReviewPriority calculates a priority score for a card (for sorting)
    GetReviewPriority(state fsrs.State, due time.Time, now time.Time) float64
}
```

#### Implementation Subtasks

1. **Basic FSRS Integration**
   - Create a simple FSRS manager that uses default parameters
   - Implement the ScheduleReview method
   - Test basic scheduling functionality

2. **Priority Calculation**
   - Implement the GetReviewPriority method
   - Define a way to calculate priority based on due date and state
   - Test sorting cards by priority

3. **Algorithm Validation**
   - Test the algorithm against known test cases
   - Verify scheduling intervals follow expected patterns
   - Ensure priority calculation works correctly

#### FSRS Manager Implementation Requirements
Implement an FSRSManager that:
- Uses go-fsrs for scheduling
- Maintains FSRS parameters
- Properly converts between our data structures and fsrs library types
- Calculates card priority for optimal review order

```go
// From go-fsrs documentation:
// - type Card struct{...} - Contains algorithm state
// - func NewCard() Card - Creates a new card
// - func DefaultParam() Parameters - Returns default parameters
// - type Parameters struct{...} - Contains algorithm parameters
// - type Weights [17]float64 - Contains algorithm weights
// - type State int8 - States are New(0), Learning(1), Review(2), Relearning(3)
// - type Rating int8 - Ratings are Again(1), Hard(2), Good(3), Easy(4)

// FSRSManagerImpl implements the FSRSManager interface
type FSRSManagerImpl struct {
    parameters fsrs.Parameters // Using Parameters from go-fsrs
}

// NewFSRSManager creates a new FSRS manager with default parameters
func NewFSRSManager() FSRSManager {
    return &FSRSManagerImpl{
        parameters: fsrs.DefaultParam(), // Using DefaultParam() from go-fsrs
    }
}

// ScheduleReview implements the FSRSManager interface
func (f *FSRSManagerImpl) ScheduleReview(state fsrs.State, rating fsrs.Rating, now time.Time) (fsrs.State, time.Time) {
    // Create FSRS card with current state
    card := fsrs.NewCard() // Create a new card
    
    // Set the card's state based on the input state
    card.State = state
    
    // Use the Update method from the go-fsrs library to calculate next schedule
    // This properly implements the FSRS algorithm instead of manual calculations
    schedulingInfo := card.Update(f.parameters, now, rating)
    
    // Extract the updated state and due date from the scheduling information
    return schedulingInfo.State, schedulingInfo.Due
}
```

#### Test Implementation Requirements
Create tests that verify:
- Scheduling behavior for different ratings
- Priority calculation for due and overdue cards
- Consistency with FSRS algorithm expectations

### Success Criteria
- [x] FSRSManager interface and implementation are complete
  - Created FSRSManager interface with ScheduleReview and GetReviewPriority methods
  - Implemented FSRSManagerImpl with proper integration with go-fsrs library
- [x] Scheduling correctly uses the FSRS algorithm
  - Used go-fsrs Parameters.Repeat for accurate FSRS scheduling
  - Properly handled state transitions and due date calculations
- [x] Priority calculation correctly orders cards for review
  - Implemented priority calculation based on card state and due date
  - Added special handling for overdue cards and learning states
- [x] Tests verify correct scheduling behavior for all ratings
  - Created tests for all possible state and rating combinations
  - Verified state transitions and due date intervals
- [x] Tests verify consistency with expected FSRS behavior
  - Added test for priority sorting to ensure cards are ordered correctly
  - Verified behavior matches FSRS algorithm expectations

### Step-by-Step Implementation
1. [x] Create directory `internal/fsrs` if it doesn't exist
2. [x] Create `internal/fsrs/fsrs.go` with the FSRS manager implementation
   - Implemented FSRSManager interface and FSRSManagerImpl struct
   - Added methods for ScheduleReview and GetReviewPriority
   - Fixed API usage based on go-fsrs documentation
3. [x] Create `internal/fsrs/fsrs_test.go` with unit tests
   - Added tests for all constructor methods
   - Created test cases for different card states and ratings
   - Implemented priority calculation tests
4. [x] Run the tests with `go test ./internal/fsrs -v`
   - Fixed issues with expected intervals for Learning cards
   - All tests now pass successfully
5. [x] Verify the algorithm behaves as expected
   - Confirmed state transitions match FSRS algorithm
   - Verified priority sorting works correctly for different card states

## Task 6: Implement Working Version of Get Due Card

### Background and Context
Now that we have implemented the storage system and the FSRS manager, we can create a working version of the `get_due_card` tool that retrieves the next card due for review.

### My Task
Implement a working version of the `get_due_card` tool that:
1. Retrieves cards from storage
2. Uses the FSRS manager to prioritize cards
3. Returns the most appropriate card for review along with statistics

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Update get_due_card handler
2. `cmd/flashcards/main_test.go` - Update tests

### Implementation Details
#### Service Interface
Create a service layer that connects storage and FSRS:

```go
// FlashcardService provides methods for managing flashcards
type FlashcardService interface {
    // GetDueCard returns the next card due for review with statistics
    GetDueCard() (Card, CardStats, error)
    
    // Other methods will be implemented in future tasks
    // ...
}

// FlashcardServiceImpl implements the FlashcardService interface
type FlashcardServiceImpl struct {
    storage     storage.Storage
    fsrsManager FSRSManager
}
```
#### Handler Implementation Requirements
The `get_due_card` handler should:
- Call the FlashcardService.GetDueCard method
- Format the response in the expected MCP format
- Handle errors appropriately

```go
// Example handler implementation
func handleGetDueCard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract service from context using the standard API
    // From mcp-go/server documentation:
    // func ServerFromContext(ctx context.Context) *MCPServer
    s := server.ServerFromContext(ctx)
    service, ok := s.UserData.(FlashcardService)
    if !ok {
        return nil, fmt.Errorf("failed to get flashcard service from context")
    }
    
    // Call service method
    card, stats, err := service.GetDueCard()
    if err != nil {
        return nil, fmt.Errorf("error getting due card: %w", err)
    }
    
    // Format response
    response := CardResponse{
        Card:  card,
        Stats: stats,
    }
    
    // Return MCP result
    return &mcp.CallToolResult{
        Result: response,
    }, nil
}
```

#### Service Implementation
```go
// GetDueCard implementation that uses FSRS priority calculation
func (s *FlashcardServiceImpl) GetDueCard() (Card, CardStats, error) {
    // Get all cards from storage
    cards, err := s.storage.ListCards(nil)
    if err != nil {
        return Card{}, CardStats{}, fmt.Errorf("error listing cards: %w", err)
    }
    
    // Current time for priority calculation
    now := time.Now()
    
    // Find due cards and calculate priority
    var dueCards []struct {
        card     Card
        priority float64
    }
    
    for _, card := range cards {
        // Consider cards due now or in the past
        if !card.Due.After(now) {
            priority := s.fsrsManager.GetReviewPriority(card.FSRS, card.Due, now)
            dueCards = append(dueCards, struct {
                card     Card
                priority float64
            }{card, priority})
        }
    }
    
    // Sort by priority (highest first)
    sort.Slice(dueCards, func(i, j int) bool {
        return dueCards[i].priority > dueCards[j].priority
    })
    
    // Return highest priority card or error if none due
    if len(dueCards) == 0 {
        return Card{}, CardStats{}, fmt.Errorf("no cards due for review")
    }
    
    // Calculate statistics
    stats := s.calculateStats(cards)
    
    return dueCards[0].card, stats, nil
}
```

#### Statistics Calculation Requirements
Initially, implement a simple version with hardcoded statistics, then enhance to calculate statistics from the data:

```go
// Calculate real statistics from card and review data
func (s *FlashcardServiceImpl) calculateStats(cards []Card) CardStats {
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    
    // Count total and due cards
    totalCards := len(cards)
    dueCards := 0
    for _, card := range cards {
        if !card.Due.After(now) {
            dueCards++
        }
    }
    
    // Get today's reviews from storage
    reviews, _ := s.storage.GetReviewsSince(today)
    
    // Calculate retention rate (correct answers / total reviews)
    correctReviews := 0
    for _, review := range reviews {
        // From go-fsrs: Rating constants (Again=1, Hard=2, Good=3, Easy=4)
        // Rating 3 (Good) or 4 (Easy) is considered correct
        if review.Rating >= int(fsrs.Good) { // fsrs.Good = 3
            correctReviews++
        }
    }
    
    retentionRate := 0.0
    if len(reviews) > 0 {
        retentionRate = float64(correctReviews) / float64(len(reviews)) * 100.0
    }
    
    return CardStats{
        TotalCards:    totalCards,
        DueCards:      dueCards,
        ReviewsToday:  len(reviews),
        RetentionRate: retentionRate,
    }
}
```

#### Test Implementation Requirements
Update the get_due_card test to:
- Create test cards with different due dates
- Verify that the highest priority card is returned
- Verify that statistics are included in the response

### Success Criteria
- [x] The `get_due_card` tool correctly retrieves the highest priority due card
  - Implemented service layer connecting storage and FSRS
  - Added priority calculation for sorting due cards
  - Returns highest priority card based on FSRS algorithm
- [x] The response includes the card and statistics
  - Response format includes card data and comprehensive statistics
  - Statistics include total cards, due cards, reviews today, and retention rate
- [x] Tests verify the correct card selection
  - Test creates cards with different due dates
  - Confirms the correct (most overdue) card is selected
  - Verifies statistics are calculated correctly
- [x] The code follows the MCP pattern from the calculator example
  - Used proper context handling for service access
  - Implemented clean separation between handler and service logic
  - Provided appropriate error handling

### Step-by-Step Implementation
1. [x] Update `cmd/flashcards/main.go` to implement the FlashcardService
   - Created service layer with Storage and FSRSManager dependencies
   - Implemented GetDueCard method with priority-based card selection
2. [x] Update the `get_due_card` handler to use the service
   - Modified handler to extract service from context
   - Added proper error handling for service operations
3. [x] Update `cmd/flashcards/main_test.go` to test actual card retrieval
   - Created tests with cards having different due dates
   - Added verification for card selection priority
4. [x] Run the tests with `go test ./cmd/flashcards -v`
   - All tests pass successfully

## Task 7: Implement Working Version of Submit Review

### Background and Context
With the get_due_card tool working, we need to implement the submit_review tool to complete the review cycle. This tool will update the card's state using the FSRS algorithm.

### My Task
Implement a working version of the `submit_review` tool that:
1. Records the review in storage
2. Updates the card's state using the FSRS manager
3. Updates the card's due date

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Update submit_review handler
2. `cmd/flashcards/main_test.go` - Update tests

### Implementation Details

#### Service Interface Update
Add a method to the FlashcardService interface:

```go
// FlashcardService provides methods for managing flashcards
type FlashcardService interface {
    // Existing methods...
    
    // SubmitReview processes a review of a card
    // Returns updated card information after applying the FSRS algorithm
    // From go-fsrs docs:
    // type Rating int8
    // const Again Rating = iota + 1
    // const Hard, Good, Easy Rating = 2, 3, 4
    SubmitReview(cardID string, rating fsrs.Rating, answer string) (Card, error)
}
```

#### Handler Implementation Requirements
The `submit_review` handler should:
- Extract parameters from the request
- Call the FlashcardService.SubmitReview method
- Return a success response or error

```go
// Example handler implementation
func handleSubmitReview(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Extract parameters from request
    var params struct {
        CardID string      `json:"card_id"`
        Rating int         `json:"rating"` // 1-4 corresponding to fsrs.Again, Hard, Good, Easy
        Answer string      `json:"answer,omitempty"`
    }
    
    if err := request.Parameters.Decode(&params); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }
    
    // Validate rating (1-4)
    if params.Rating < 1 || params.Rating > 4 {
        return nil, fmt.Errorf("rating must be between 1 and 4")
    }
    
    // Convert rating to fsrs.Rating
    rating := fsrs.Rating(params.Rating)
    
    // Extract service from context using the standard API
    // Use server.ServerFromContext(ctx) to get the server instance
    s := server.ServerFromContext(ctx)
    // Then get the service from the server instance
    service := s.UserData.(FlashcardService)
    
    // Call service method
    updatedCard, err := service.SubmitReview(params.CardID, rating, params.Answer)
    if err != nil {
        return nil, fmt.Errorf("error submitting review: %w", err)
    }
    
    // Format response
    response := ReviewResponse{
        Success: true,
        Message: "Review submitted successfully",
        Card:    updatedCard,
    }
    
    // Return MCP result
    return &mcp.CallToolResult{
        Result: response,
    }, nil
}
```

#### Service Implementation
```go
// SubmitReview implementation
func (s *FlashcardServiceImpl) SubmitReview(cardID string, rating fsrs.Rating, answer string) (Card, error) {
    // Get the card from storage
    card, err := s.storage.GetCard(cardID)
    if err != nil {
        return Card{}, fmt.Errorf("error getting card: %w", err)
    }
    
    now := time.Now()
    
    // Use FSRS manager to schedule the review using the go-fsrs library
    // The FSRSManager.ScheduleReview method uses fsrs.Card.Update internally
    // which properly implements the complete FSRS algorithm
    updatedState, newDueDate := s.fsrsManager.ScheduleReview(card.State, rating, now)
    
    // Update the card with new state information
    card.State = updatedState
    card.Due = newDueDate
    
    // Extract other algorithm-calculated fields from the FSRS state
    // These fields could include stability, difficulty, and elapsed days
    // that are part of the Card state from the go-fsrs library
    
    // Create review record with comprehensive state information
    review := storage.Review{
        ID:           uuid.New().String(),
        CardID:       cardID,
        Rating:       int(rating),
        Timestamp:    now,
        Answer:       answer,
        State:        int(updatedState), // State from fsrs.State enum (New=0 through Relearning=3)
        // Include additional FSRS algorithm state from updatedState
    }
    
    // Add review to storage
    if _, err := s.storage.AddReview(cardID, int(rating), answer); err != nil {
        return Card{}, fmt.Errorf("error adding review: %w", err)
    }
    
    // Update card in storage
    if err := s.storage.UpdateCard(card); err != nil {
        return Card{}, fmt.Errorf("error updating card: %w", err)
    }
    
    return card, nil
}
```

#### Test Implementation Requirements
Update the submit_review test to:
- Create a test card
- Submit a review for the card
- Verify that the card's state and due date were updated
- Verify that a review record was created

### Success Criteria
- [ ] The `submit_review` tool correctly updates the card's state and due date
- [ ] The tool adds a review record to storage
- [ ] Tests verify the correct state updates and review creation
- [ ] The code follows the MCP pattern from the calculator example

### Step-by-Step Implementation
1. [ ] Update the FlashcardService interface to include SubmitReview
2. [ ] Implement the SubmitReview method in FlashcardServiceImpl
3. [ ] Update the submit_review handler to use the service
4. [ ] Update tests to verify correct behavior
5. [ ] Run the tests with `go test ./cmd/flashcards -v`

## Task 8: Complete Remaining Tool Implementations

### Background and Context
Now that we have implemented the core functionality (get_due_card and submit_review), we need to complete the implementation of the remaining tools.

### My Task
Implement working versions of the remaining tools:
1. `update_card` - Update an existing flashcard
2. `delete_card` - Delete a flashcard
3. `list_cards` - List all flashcards

### Files to Create/Modify
1. `cmd/flashcards/main.go` - Update handlers for remaining tools
2. `cmd/flashcards/main_test.go` - Update tests

### Implementation Details

#### Service Interface Update
Add methods to the FlashcardService interface:

```go
// FlashcardService provides methods for managing flashcards
type FlashcardService interface {
    // Existing methods...
    
    // UpdateCard updates an existing flashcard
    // Only updates content fields, preserves FSRS algorithm data
    UpdateCard(cardID string, front, back string, tags []string) (Card, error)
    
    // DeleteCard deletes a flashcard
    DeleteCard(cardID string) error
    
    // ListCards lists all flashcards, optionally filtered by tags
    ListCards(tags []string, includeStats bool) ([]Card, CardStats, error)
}
```

#### Implementation Requirements
For each tool:

1. `update_card` handler:
   - Retrieve the card from storage
   - Update the fields that were provided in the request
   - Save the updated card to storage

2. `delete_card` handler:
   - Delete the card from storage
   - Return a success response

3. `list_cards` handler:
   - Retrieve cards from storage, filtered by tags if provided
   - Calculate statistics if requested
   - Return the cards and statistics

#### Test Implementation Requirements
For each tool, create tests that:
- Verify the tool correctly performs its operation
- Verify the response structure
- Test edge cases and error handling

### Success Criteria
- [ ] All tools are fully implemented and connected to the storage system
- [ ] Each tool correctly handles its operation
- [ ] Tests verify the correct behavior for each tool
- [ ] The code follows the MCP pattern from the calculator example

### Step-by-Step Implementation
1. [ ] Update the FlashcardService interface to include all methods
2. [ ] Implement all methods in FlashcardServiceImpl
3. [ ] Update each handler to use the service
4. [ ] Update tests to verify correct behavior
5. [ ] Run the tests with `go test ./cmd/flashcards -v`

## Task 9: Implement Statistics Calculation

### Background and Context
Currently, we are using hardcoded statistics. We need to implement actual statistics calculation based on the flashcard and review data.

### My Task
Implement statistics calculation that:
1. Counts total cards, due cards, and reviews today
2. Calculates retention rate based on review history
3. Provides these statistics to the get_due_card and list_cards tools

### Files to Create/Modify
1. `internal/stats/stats.go` - Statistics calculation
2. `internal/stats/stats_test.go` - Unit tests
3. `cmd/flashcards/main.go` - Update to use real statistics

### Implementation Details

#### Statistics Interface
Define a statistics calculator interface:

```go
// StatsCalculator provides methods for calculating flashcard statistics
type StatsCalculator interface {
    // Calculate computes statistics for a set of cards and reviews
    Calculate(cards map[string]storage.Card, reviews []storage.Review) CardStats
}
```

#### Implementation Requirements
Implement a statistics calculator that:
- Counts total cards
- Counts cards due for review
- Counts reviews completed today
- Calculates retention rate (percentage of correct answers)

#### Integration Requirements
Update the FlashcardService to:
- Use the statistics calculator
- Provide real statistics in the get_due_card and list_cards methods

#### Test Implementation Requirements
Create tests that verify:
- Correct counting of cards and reviews
- Correct calculation of retention rate
- Integration with the FlashcardService

### Success Criteria
- [ ] Statistics calculator is implemented and tested
- [ ] get_due_card and list_cards tools provide real statistics
- [ ] Tests verify correct statistics calculation
- [ ] The code follows good design practices with proper separation of concerns

### Step-by-Step Implementation
1. [ ] Create directory `internal/stats` if it doesn't exist
2. [ ] Create `internal/stats/stats.go` with the statistics calculator
3. [ ] Create `internal/stats/stats_test.go` with unit tests
4. [ ] Update the FlashcardService to use the statistics calculator
5. [ ] Update the tools to provide real statistics
6. [ ] Run the tests to verify correct behavior

## Task 10: Implement End-to-End Integration Test

### Background and Context
With all the individual components implemented, we need to create an end-to-end integration test that verifies the complete flashcard review cycle.

### My Task
Create an end-to-end integration test that:
1. Creates flashcards
2. Gets a card due for review
3. Submits a review
4. Verifies that the card's state and due date were updated correctly

### Files to Create/Modify
1. `cmd/flashcards/integration_test.go` - End-to-end integration test

### Implementation Details

#### Test Implementation Requirements
Create an integration test that:
- Sets up a clean test environment with a temporary file
- Creates multiple flashcards with different properties
- Gets a card due for review
- Submits a review with different ratings
- Verifies the card's state changes appropriately
- Checks that due dates are scheduled according to the FSRS algorithm
- Verifies statistics are calculated correctly

### Success Criteria
- [ ] Integration test verifies the complete flashcard review cycle
- [ ] Test covers all key operations
- [ ] Test verifies correct behavior of the FSRS algorithm
- [ ] Test verifies correct statistics calculation

### Step-by-Step Implementation
1. [ ] Create `cmd/flashcards/integration_test.go` with the end-to-end test
2. [ ] Run the test with `go test ./cmd/flashcards -v -run=TestIntegration`
3. [ ] Fix any issues that arise during testing

## Task 11: Final Cleanup and Documentation

### Background and Context
With all functionality implemented and tested, we need to clean up the code and add documentation.

### My Task
Perform final cleanup and documentation:
1. Add godoc comments to all exported types and functions
2. Create a README with usage instructions
3. Ensure consistent code formatting and organization

### Files to Create/Modify
1. All source files - Add godoc comments
2. `README.md` - Create usage instructions
3. `cmd/flashcards/main.go` - Add usage help

### Implementation Details

#### Documentation Requirements
Add godoc comments to all exported types and functions, following Go's documentation conventions.

#### README Requirements
Create a README that includes:
- Overview of the project
- Installation instructions
- Usage examples
- API documentation
- Architecture overview

#### Help Output Requirements
Add usage help output to the main program:
- Command-line flags
- Examples of how to use the tools
- Information about the API

### Success Criteria
- [ ] All exported types and functions have godoc comments
- [ ] README provides comprehensive documentation
- [ ] Code is consistently formatted and organized
- [ ] Main program provides helpful usage information

### Step-by-Step Implementation
1. [ ] Add godoc comments to all source files
2. [ ] Create `README.md` with comprehensive documentation
3. [ ] Add usage help to the main program
4. [ ] Run `go fmt ./...` to ensure consistent formatting
5. [ ] Run `golint ./...` to catch any potential issues

## Implementation Sequence Summary

This implementation plan follows a steel-thread approach, building the system incrementally with end-to-end testing at each stage:

1. First, create a minimal MCP server with a single tool and hardcoded responses
2. Add the remaining MCP tools, still with hardcoded responses
3. Implement the storage system
4. Connect the create_card tool to storage
5. Implement the FSRS manager
6. Connect the get_due_card tool to storage and FSRS
7. Connect the submit_review tool to complete the review cycle
8. Implement the remaining tools
9. Add real statistics calculation
10. Create an end-to-end integration test
11. Finalize with cleanup and documentation

Each step builds on the previous ones, ensuring we have a working system at each stage that can be tested end-to-end.