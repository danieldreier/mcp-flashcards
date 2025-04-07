# Flashcards MCP

A Go-based implementation of a spaced repetition flashcard system using the Model Context Protocol (MCP). This allows AI assistants and other systems to leverage spaced repetition for knowledge retention.

## Features

- Comprehensive API for managing flashcards through the Model Context Protocol
- Implementation of the Free Spaced Repetition Scheduler (FSRS) algorithm for optimized review scheduling
- Persistent storage of flashcard data and review history across sessions
- Support for tagging and filtering cards

## Installation

### Prerequisites

- Go 1.19 or later
- Access to an MCP-compatible client (e.g., Claude Desktop, Anthropic Claude in Console)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/danieldreier/mcp-flashcards.git
cd mcp-flashcards

# Build the flashcards executable
go build -o cmd/flashcards/flashcards ./cmd/flashcards
```

## Configuration

### Claude Desktop Configuration

To ensure your flashcards persist across sessions in Claude Desktop, you need to configure the MCP server with an **absolute path** for the storage file.

Create or edit your MCP configuration in Claude Desktop:

1. Open Claude Desktop
2. Go to Settings → MCP
3. Add the flashcards MCP server with an absolute file path:

```json
{
  "mcpServers": {
    "Flashcards": {
      "command": "/path/to/mcp-flashcards/cmd/flashcards/flashcards",
      "args": ["-file", "/absolute/path/to/store/flashcards-data.json"]
    }
  }
}
```

Example (using home directory):

```json
{
  "mcpServers": {
    "Flashcards": {
      "command": "/Users/yourusername/go/src/github.com/danieldreier/mcp-flashcards/cmd/flashcards/flashcards",
      "args": ["-file", "/Users/yourusername/flashcards-data.json"]
    }
  }
}
```

> **Important:** The default configuration uses a relative path (`./flashcards.json`), which depends on the current working directory when Claude Desktop launches the MCP. Using an absolute path ensures your flashcards are saved to and loaded from the same location regardless of the working directory.

## Usage

Once configured, you can use the flashcards MCP with Claude Desktop. Here are some example prompts:

### Creating Cards

```
Use the Flashcards MCP to create a new flashcard:
Front: What is the capital of France?
Back: Paris
Tags: geography, europe
```

### Reviewing Cards

```
Use the Flashcards MCP to get my next due card for review.
```

After seeing the card:

```
I recall the answer is Paris. Rate this card as "Good" (3).
```

### Listing Cards

```
Use the Flashcards MCP to list all my flashcards with the tag "geography".
```

## Available MCP Tools

The Flashcards MCP provides the following tools:

1. **get_due_card**: Returns the next card due for review
2. **submit_review**: Records a review with rating (1-4) for a card
3. **create_card**: Creates a new flashcard
4. **update_card**: Updates an existing flashcard
5. **delete_card**: Deletes a flashcard
6. **list_cards**: Lists flashcards, optionally filtered by tags

## Troubleshooting

### Cards Not Persisting Between Sessions

If your cards aren't being saved between sessions:

1. Verify you're using an absolute path for the storage file in your MCP configuration
2. Check that the directory for the storage file exists and is writable
3. Confirm your Claude Desktop configuration is correctly formatted
4. Check that the path to the executable is correct

### Running Tests

To verify persistence is working correctly:

```bash
cd cmd/flashcards
go test -v -run TestPersistenceAcrossSessions
```

## License

[MIT License](LICENSE)