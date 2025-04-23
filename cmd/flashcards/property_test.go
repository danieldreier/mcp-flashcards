package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// setupPropertyTestClient sets up an MCP client for a single property test run.
// It ensures a clean state by using a new temporary file for each run.
func setupPropertyTestClient(t *testing.T) (mcpClient *client.StdioMCPClient, ctx context.Context, cancel context.CancelFunc, cleanup func()) {
	t.Helper()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "flashcards-prop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Use a fixed filename in the temp dir
	tempFilePath := filepath.Join(tempDir, "flashcards-test.json")

	// Initialize with an empty JSON object
	err = os.WriteFile(tempFilePath, []byte("{}"), 0644)
	if err != nil {
		os.RemoveAll(tempDir) // Clean up if initialization fails
		t.Fatalf("Failed to initialize temp file: %v", err)
	}

	// Create cleanup function to remove the temp directory
	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	// First check if the main binary exists (might have already been built)
	wd, err := os.Getwd()
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Determine if we need to build the binary first
	binPath := filepath.Join(wd, "flashcards")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// Build the binary
		buildCmd := exec.Command("go", "build", "-o", binPath)
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			cleanup()
			t.Fatalf("Failed to build flashcards binary: %v\nOutput: %s", err, buildOutput)
		}
	}

	// Create the MCP client targeting the server binary with the specific temp file
	mcpClient, err = client.NewStdioMCPClient(
		binPath,    // Use the binary directly rather than "go run ."
		[]string{}, // Empty ENV
		"-file",
		tempFilePath,
	)
	if err != nil {
		cleanup() // Run cleanup if client creation fails
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create context with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second) // Adjust timeout as needed

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-property-test-client",
		Version: "0.1.0",
	}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		mcpClient.Close()
		cancel()
		cleanup()
		t.Fatalf("Failed to initialize MCP client: %v", err)
	}

	return mcpClient, ctx, cancel, cleanup
}

// --- Generators ---

// genNonEmptyString generates non-empty strings for card content.
func genNonEmptyString(maxLength int) gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= maxLength
	}).WithLabel("NonEmptyString")
}

// genTags generates a slice of unique, non-empty strings for tags.
func genTags(maxTags int, maxTagLength int) gopter.Gen {
	return gen.SliceOf(genNonEmptyString(maxTagLength)).
		Map(func(tags []string) []string {
			uniqueTags := make(map[string]struct{})
			result := []string{}
			for _, tag := range tags {
				if _, exists := uniqueTags[tag]; !exists && len(result) < maxTags {
					uniqueTags[tag] = struct{}{}
					result = append(result, tag)
				}
			}
			// Sort for consistent comparison later if needed
			sort.Strings(result)
			return result
		}).WithLabel("UniqueTags")
}

// --- Property Tests ---

func TestFlashcardProperties_EndToEnd(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5 // Start with very few tests during debugging
	// parameters.MinSuccessfulTests = 100 // Increase later
	properties := gopter.NewProperties(parameters)

	// First, build the binary once to avoid repeated builds during testing
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	binPath := filepath.Join(wd, "flashcards")
	buildCmd := exec.Command("go", "build", "-o", binPath)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build flashcards binary: %v\nOutput: %s", err, buildOutput)
	}
	t.Logf("Successfully built flashcards binary at %s", binPath)

	// Property: Create-Read Consistency
	properties.Property("CreateReadConsistency", prop.ForAll(
		func(front string, back string, tags []string) bool {
			t.Logf("Testing with front='%s', back='%s', tags=%v", front, back, tags)

			// Setup client for this specific test case
			mcpClient, ctx, cancel, cleanup := setupPropertyTestClient(t)
			defer func() {
				cancel()
				mcpClient.Close()
				cleanup()
			}()

			// --- Create Card ---
			createCardRequest := mcp.CallToolRequest{}
			createCardRequest.Params.Name = "create_card"
			createCardRequest.Params.Arguments = map[string]interface{}{
				"front": front,
				"back":  back,
				"tags":  interfaceSlice(tags), // Convert []string to []interface{} for MCP call
			}

			t.Logf("Calling create_card tool")
			createResult, err := mcpClient.CallTool(ctx, createCardRequest)
			if err != nil {
				t.Logf("Failed to call create_card: %v", err)
				return false
			}
			if len(createResult.Content) == 0 {
				t.Logf("No content returned from create_card")
				return false
			}
			createTextContent, ok := createResult.Content[0].(mcp.TextContent)
			if !ok {
				t.Logf("Expected TextContent from create_card, got %T", createResult.Content[0])
				return false
			}

			t.Logf("Parsing create_card response: %s", createTextContent.Text)
			var createResponse CreateCardResponse
			err = json.Unmarshal([]byte(createTextContent.Text), &createResponse)
			if err != nil {
				t.Logf("Failed to parse create_card response JSON: %v", err)
				return false
			}
			if createResponse.Card.ID == "" {
				t.Logf("create_card response did not contain a valid Card ID")
				return false
			}
			createdCardID := createResponse.Card.ID
			t.Logf("Created card with ID: %s", createdCardID)

			// --- Read Card (using list_cards for simplicity now) ---
			listCardsRequest := mcp.CallToolRequest{}
			listCardsRequest.Params.Name = "list_cards"
			// No arguments needed to list all cards

			t.Logf("Calling list_cards tool")
			listResult, err := mcpClient.CallTool(ctx, listCardsRequest)
			if err != nil {
				t.Logf("Failed to call list_cards: %v", err)
				return false
			}
			if len(listResult.Content) == 0 {
				t.Logf("No content returned from list_cards")
				return false
			}
			listTextContent, ok := listResult.Content[0].(mcp.TextContent)
			if !ok {
				t.Logf("Expected TextContent from list_cards, got %T", listResult.Content[0])
				return false
			}

			t.Logf("Parsing list_cards response: %s", listTextContent.Text)
			var listResponse ListCardsResponse
			err = json.Unmarshal([]byte(listTextContent.Text), &listResponse)
			if err != nil {
				t.Logf("Failed to parse list_cards response JSON: %v", err)
				return false
			}

			// Find the created card in the list
			var retrievedCard Card
			found := false
			for _, card := range listResponse.Cards {
				if card.ID == createdCardID {
					retrievedCard = card
					// Sort tags for comparison
					sort.Strings(retrievedCard.Tags)
					found = true
					break
				}
			}

			if !found {
				t.Logf("Card with ID %s not found in list_cards response", createdCardID)
				return false
			}
			t.Logf("Found card in list_cards response: %+v", retrievedCard)

			// --- Verify Consistency ---
			// Compare the originally provided data with the retrieved card data
			if retrievedCard.Front != front {
				t.Logf("Front mismatch: expected '%s', got '%s'", front, retrievedCard.Front)
				return false
			}
			if retrievedCard.Back != back {
				t.Logf("Back mismatch: expected '%s', got '%s'", back, retrievedCard.Back)
				return false
			}

			// Normalize tags for comparison - treat nil and empty slices as equivalent
			if !compareTags(tags, retrievedCard.Tags) {
				t.Logf("Tags mismatch: expected %v, got %v", tags, retrievedCard.Tags)
				return false
			}

			t.Logf("Card validation successful")
			return true // Test passed
		},
		genNonEmptyString(100), // Max length for front
		genNonEmptyString(200), // Max length for back
		genTags(5, 20),         // Max 5 tags, max 20 chars per tag
	))

	// Run all defined properties
	properties.TestingRun(t)
}

// compareTags handles the comparison of tag slices, treating nil and empty slices as equivalent.
func compareTags(expected, actual []string) bool {
	// If both are nil or empty, consider them equal
	if (expected == nil || len(expected) == 0) && (actual == nil || len(actual) == 0) {
		return true
	}

	// If only one is nil/empty but the other isn't, they're not equal
	if (expected == nil || len(expected) == 0) != (actual == nil || len(actual) == 0) {
		return false
	}

	// If we get here, both slices have elements, so compare them normally
	if len(expected) != len(actual) {
		return false
	}

	// We already sorted the slices, so we can compare them directly
	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}

	return true
}

// Helper function to convert []string to []interface{} for MCP tool calls
func interfaceSlice(strings []string) []interface{} {
	interfaces := make([]interface{}, len(strings))
	for i, s := range strings {
		interfaces[i] = s
	}
	return interfaces
}
