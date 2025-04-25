package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestEmptyCardBug reproduces a bug where creating a card with empty
// front and back values causes the system to hang
func TestEmptyCardBug(t *testing.T) {
	// Setup: Create a temp file for the flashcards state
	tempDir, err := os.MkdirTemp("", "flashcards-bug-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFile := filepath.Join(tempDir, "flashcards-test.json")
	if err := os.WriteFile(stateFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create state file: %v", err)
	}

	// Locate and build the flashcards binary
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate to project root if we're in the tests directory
	if filepath.Base(workDir) == "tests" {
		workDir = filepath.Dir(workDir)
	}

	binPath := filepath.Join(workDir, "flashcards")
	t.Logf("Binary path: %s", binPath)

	// Build the binary if it doesn't exist
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Logf("Building flashcards binary...")
		makeCmd := exec.Command("make", "build")
		makeCmd.Dir = workDir
		if output, err := makeCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build flashcards binary: %v\nOutput: %s", err, output)
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the MCP client
	mcpClient, err := client.NewStdioMCPClient(
		binPath,
		[]string{"PYTHONUNBUFFERED=1", "GODEBUG=asyncpreemptoff=1"},
		"-file", stateFile,
	)
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}
	defer mcpClient.Close()

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-bug-test",
		Version: "0.1.0",
	}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize MCP client: %v", err)
	}

	// REPRODUCING THE BUG:
	// Create a card with empty front, back, and tags - this caused the hang in property testing
	createCardRequest := mcp.CallToolRequest{}
	createCardRequest.Params.Name = "create_card"
	createCardRequest.Params.Arguments = map[string]interface{}{
		"front": "",
		"back":  "",
		"tags":  []interface{}{},
	}

	t.Logf("Sending create_card request with empty values...")

	// Add a timeout for just this operation to detect hangs
	operationCtx, operationCancel := context.WithTimeout(ctx, 10*time.Second)
	defer operationCancel()

	// Create a channel for the result
	resultChan := make(chan struct {
		result interface{}
		err    error
	})

	// Run the operation in a goroutine so we can detect timeouts
	go func() {
		result, err := mcpClient.CallTool(ctx, createCardRequest)
		resultChan <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	// Wait for either the result or timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			t.Logf("Got error (expected): %v", res.err)
		} else {
			t.Logf("Successfully created card with empty values: %+v", res.result)
		}
	case <-operationCtx.Done():
		t.Fatalf("Operation timed out after 10 seconds, which confirms the hang bug")
	}

	t.Log("Test completed successfully - the operation did not hang")
}
