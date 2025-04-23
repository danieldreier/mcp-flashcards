package propertytest

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestCreateReadConsistency tests that after creating a card, we can read it back
// and the data matches what we originally provided.
func TestCreateReadConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 5 // Adjust based on test needs
	properties := gopter.NewProperties(parameters)

	// Build the binary once for all tests
	BuildBinary(t)

	properties.Property("CreateReadConsistency", prop.ForAll(
		func(front string, back string, tags []string) bool {
			t.Logf("Testing with front='%s', back='%s', tags=%v", front, back, tags)

			// Setup client for this specific test case
			mcpClient, ctx, cancel, cleanup := SetupPropertyTestClient(t)
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
				"tags":  InterfaceSlice(tags), // Convert []string to []interface{} for MCP call
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
			if !CompareTags(tags, retrievedCard.Tags) {
				t.Logf("Tags mismatch: expected %v, got %v", tags, retrievedCard.Tags)
				return false
			}

			t.Logf("Card validation successful")
			return true // Test passed
		},
		GenNonEmptyString(100), // Max length for front
		GenNonEmptyString(200), // Max length for back
		GenTags(5, 20),         // Max 5 tags, max 20 chars per tag
	))

	properties.TestingRun(t)
}
