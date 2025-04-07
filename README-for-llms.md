# Flashcards MCP - Technical Overview

This document provides a technical overview of the Flashcards MCP (Model Context Protocol) server, designed to be easily understood by LLMs for future extension and maintenance.

## Purpose

The Flashcards MCP is a Go-based implementation of a spaced repetition flashcard system that:

1. Provides a comprehensive API for managing flashcards through the Model Context Protocol
2. Implements the Free Spaced Repetition Scheduler (FSRS) algorithm to optimize review scheduling
3. Enables AI assistants and other systems to leverage spaced repetition for knowledge retention
4. Persists flashcard data and review history to disk for long-term learning

This implementation follows the MCP pattern to allow LLMs to directly interact with the spaced repetition system programmatically, creating a bridge between AI capabilities and proven learning techniques.

## Technical Implementation

### Code Organization

The project follows a modular architecture with a clean separation of concerns. The codebase is organized into the following files:

1. **models.go**: Contains all data structures
   - Card, CardStats, and all response type definitions
   - Provides clear data models with JSON serialization support
   - Establishes the foundation for the rest of the application

2. **service.go**: Implements business logic in FlashcardService
   - Manages operations for flashcards with storage and FSRS algorithm integration
   - Provides methods for creating, updating, deleting, and retrieving cards
   - Implements review submission and scheduling logic
   - Calculates statistics for flashcard review performance

3. **handlers.go**: Contains MCP tool handlers
   - Processes incoming MCP tool requests and extracts parameters
   - Maps requests to appropriate service layer methods
   - Formats responses as JSON for client consumption
   - Implements proper error handling and validation

4. **main.go**: Handles server setup and initialization
   - Configures and starts the MCP server
   - Registers tools and associates them with handlers
   - Initializes storage and service components
   - Provides command-line flag support for configuration

This modular structure improves maintainability, readability, and testability of the codebase while allowing for easier extension and future development.

### Core Architecture

The system is built around these primary components:

1. **MCP Server Layer**: Exposes tools for interacting with the flashcard system
2. **Flashcard Service**: Coordinates business logic between storage and FSRS algorithm
3. **FSRS Manager**: Implements spaced repetition scheduling logic
4. **Storage System**: Handles persistence of cards and review history

The implementation follows a clean separation of concerns with well-defined interfaces between components.

### Data Flow

1. External clients call MCP tools (e.g., `get_due_card`, `submit_review`)
2. Tool handlers extract parameters and delegate to the Flashcard Service
3. Flashcard Service orchestrates operations between storage and FSRS algorithm
4. Results are formatted as JSON and returned to the client

### Key Data Structures

- **Card**: Represents a flashcard with front/back content and FSRS algorithm state
- **Review**: Tracks review history with ratings and timestamps
- **FlashcardStore**: The root data structure persisted to disk

### Storage Implementation

- File-based JSON storage system
- Thread-safe operations using mutex locks
- Atomic file operations for data integrity (write to temp file, then rename)
- Unique UUIDs for card identification

### FSRS Integration

The system integrates with the [go-fsrs](https://github.com/open-spaced-repetition/go-fsrs) library to implement the Free Spaced Repetition Scheduler algorithm:

- Cards are scheduled based on difficulty and previous review performance
- Four rating levels: Again (1), Hard (2), Good (3), and Easy (4)
- Card states progress through: New → Learning → Review (or Relearning if forgotten)
- Custom priority calculator for determining review order of due cards

## Key APIs and Tools

### MCP Tools

The system exposes these tools through the Model Context Protocol:

1. **get_due_card**: Returns the next card due for review with statistics
   - Higher priority for overdue cards and those in learning/relearning states
   - Returns card content and overall statistics

2. **submit_review**: Records a review rating for a card
   - Rating scale: Again(1), Hard(2), Good(3), Easy(4)
   - Updates card state and schedules next review
   - Optional answer field for tracking user responses

3. **create_card**: Creates a new flashcard
   - Required front/back content
   - Optional tags for categorization
   - Initializes as "New" state in the FSRS algorithm

4. **update_card**: Updates an existing flashcard
   - Preserves algorithm data while allowing content changes
   - Supports partial updates (only fields that need changing)

5. **delete_card**: Deletes a flashcard by ID

6. **list_cards**: Lists flashcards with optional filtering
   - Filter by tags
   - Optional inclusion of statistics

### Storage Interface

```go
type Storage interface {
    // Card operations
    CreateCard(front, back string, tags []string) (Card, error)
    GetCard(id string) (Card, error)
    UpdateCard(card Card) error
    DeleteCard(id string) error
    ListCards(tags []string) ([]Card, error)
    
    // Review operations
    AddReview(cardID string, rating fsrs.Rating, answer string) (Review, error)
    GetCardReviews(cardID string) ([]Review, error)
    
    // File operations
    Load() error
    Save() error
}
```

### FSRS Manager Interface

```go
type FSRSManager interface {
    // Calculate next review date based on card state and rating
    ScheduleReview(state fsrs.State, rating fsrs.Rating, now time.Time) (fsrs.State, time.Time)
    
    // Calculate priority score for ordering cards
    GetReviewPriority(state fsrs.State, due time.Time, now time.Time) float64
}
```

## Design Philosophy

### End-to-End Integration Testing

The system employs a robust testing strategy that emphasizes real-world usage:

- Tests connect to an actual running MCP server over stdio
- Test suite systematically verifies each operation through the MCP interface
- Uses temporary file storage to ensure test isolation
- Tests cover the full lifecycle: create → review → update → delete

This approach guarantees that the system works correctly from an external client's perspective, not just at the unit level.

### Concurrent Operations

- Thread safety through careful mutex usage
- Supports multiple simultaneous clients without data corruption
- Read/write locks for optimized performance

### Extensibility

The system is designed for easy extension:

- Clean interface boundaries between components
- New tools can be added without changing existing functionality
- Storage implementation can be swapped (e.g., for a database backend)
- Modular code organization makes it easy to add new features

### Benefits of the Modular Architecture

The refactoring from a monolithic main.go to a modular architecture provides several benefits:

1. **Improved Readability**: Each file has a clear, single responsibility, making it easier to understand the codebase.
2. **Enhanced Maintainability**: Changes to one component (e.g., data models) don't require modifications to other parts.
3. **Better Testability**: Components can be tested in isolation, simplifying unit testing.
4. **Easier Onboarding**: New developers (including LLMs) can more quickly understand the codebase structure.
5. **Scalable Development**: Multiple developers can work on different components simultaneously with minimal conflicts.
6. **Future-Proof Design**: The modular structure accommodates growth and new features without major restructuring.

### Data Integrity

- Atomic file operations prevent data corruption during writes
- Comprehensive error handling with descriptive messages
- Validation of inputs to prevent invalid states

## Usage Example for LLMs

To use the Flashcards MCP in an LLM context:

1. Connect to the server using MCP client libraries
2. Use `create_card` for adding new knowledge to remember
3. Use `get_due_card` to retrieve the next card due for review
4. Present the card front to the user and ask for their response
5. Compare user's answer with the card back
6. Use `submit_review` with an appropriate rating based on answer quality
7. Continue the review session with more due cards

This enables LLMs to facilitate effective spaced repetition learning sessions, leveraging scientifically proven techniques for knowledge retention.