package main

import (
	"os"
	"strings"
	"testing"

	// These imports are used indirectly via the setupMCPClient function
	_ "context"

	_ "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestServerInfoAndToolDescriptions(t *testing.T) {
	// Setup client
	c, ctx, cancel, tempFilePath := setupMCPClient(t)
	defer c.Close()
	defer cancel()
	defer os.Remove(tempFilePath)

	// 1. Verify the server info is provided in the initialize result
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-test-client",
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Check that the instructions field contains our server info
	if initResult.Instructions == "" {
		t.Error("Server instructions are empty")
	}

	// Verify key content from the instructions
	expectedContentSnippets := []string{
		"This is a spaced repetition flashcard system designed for middle school students",
		"PRESENTATION PHASE",
		"RESPONSE PHASE",
		"EVALUATION PHASE",
		"RATING PHASE",
		"TRANSITION PHASE",
		"COMPLETION PHASE",
	}

	for _, snippet := range expectedContentSnippets {
		if !strings.Contains(initResult.Instructions, snippet) {
			t.Errorf("Server instructions missing expected content: %s", snippet)
		}
	}

	// 2. Verify tool descriptions match the enhanced descriptions
	listToolsRequest := mcp.ListToolsRequest{}
	listToolsResult, err := c.ListTools(ctx, listToolsRequest)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Create a map of tool names to their expected description fragments
	expectedToolDescriptions := map[string][]string{
		"get_due_card": {
			"Show ONLY the front (question) side of the card to the student",
			"DO NOT reveal the back (answer) side at this stage",
			"Ask the student to attempt to recall and provide their answer",
		},
		"submit_review": {
			"Now that the student has provided their answer, show the correct answer",
			"Compare the student's answer with the correct one supportively and enthusiastically",
			"For incorrect answers, briefly explain the concept in a friendly way",
			"Rating 1: Answer was absent or completely wrong",
		},
		"create_card": {
			"Analyze what topics the student struggled with most in previous cards",
			"Focus on fundamental knowledge that applies to multiple missed questions",
			"Make questions clear, specific, and targeted",
		},
		"update_card": {
			"Preserve the educational intent of the card",
			"Improve clarity or accuracy to aid learning",
			"Consider making the card more engaging for middle school students",
		},
		"list_cards": {
			"When displaying cards to the student, prefer to show only the question side",
			"Use this data to identify patterns in what the student finds challenging",
			"Maintain an enthusiastic, encouraging tone when discussing the cards",
		},
		"help_analyze_learning": {
			"Review the student's performance across all cards",
			"Identify patterns in what concepts are challenging",
			"Suggest new cards that would help with prerequisite knowledge",
			"Look for fundamental concepts that apply across multiple difficult cards",
			"Use many emojis and exciting middle-school appropriate language",
		},
	}

	// Check each tool's description
	toolsChecked := make(map[string]bool)
	for _, tool := range listToolsResult.Tools {
		if snippets, exists := expectedToolDescriptions[tool.Name]; exists {
			toolsChecked[tool.Name] = true

			for _, snippet := range snippets {
				if !strings.Contains(tool.Description, snippet) {
					t.Errorf("Tool %s missing expected description content: %s", tool.Name, snippet)
				}
			}
		}
	}

	// Ensure all expected tools were found
	for toolName := range expectedToolDescriptions {
		if !toolsChecked[toolName] {
			t.Errorf("Tool %s not found in the list of tools", toolName)
		}
	}
}
