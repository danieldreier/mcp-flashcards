// Package propertytest provides property-based tests for the flashcards MCP service.
package propertytest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter/commands"
	"github.com/mark3labs/mcp-go/client"
)

// TestCommon provides basic wrappers to other functions in the propertytest package
// that make individual test files compilable when run directly.
// This makes it possible to run individual test files with `go test -run TestFSRSNewCardTransitions`.

// SetupTestClient is a wrapper around SetupPropertyTestClient that should be used by
// individual test files to ensure they are individually compilable.
// It now returns only the client, context, and cancel func. Cleanup is managed by the caller via GetOrCreatePropertyTestFile.
func SetupTestClient(t *testing.T) (*client.Client, context.Context, context.CancelFunc, func(), error) {
	t.Helper()
	// Create a unique state file for this test instance
	_, stateFilePath, tempCleanup, err := CreateTempStateFile(t)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to create temp state file: %w", err)
	}

	// Setup client using the state file
	mcpClient, ctx, cancel, err := SetupPropertyTestClient(t, stateFilePath)
	if err != nil {
		tempCleanup() // Cleanup temp state if client setup fails
		return nil, nil, nil, nil, fmt.Errorf("failed to setup property test client: %w", err)
	}

	// Combine client/context cleanup with temp state cleanup
	fullCleanup := func() {
		t.Logf("Running combined cleanup for SetupTestClient")
		if cancel != nil {
			cancel()
		}
		if mcpClient != nil {
			mcpClient.Close()
		}
		tempCleanup() // Call the temp state cleanup
	}

	return mcpClient, ctx, cancel, fullCleanup, nil
}

// SetupTestClientWithLongTimeout creates a test client with a longer timeout.
// Cleanup is managed by the caller via GetOrCreatePropertyTestFile.
func SetupTestClientWithLongTimeout(t *testing.T, timeoutSeconds int) (*client.Client, context.Context, context.CancelFunc, func(), error) {
	t.Helper()
	// Create a unique state file for this test instance
	_, stateFilePath, tempCleanup, err := CreateTempStateFile(t)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to create temp state file: %w", err)
	}

	// Create the long-lived context first
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	t.Logf("Created context with %d second timeout", timeoutSeconds)

	// Setup client using the state file with the long-lived context
	mcpClient, _, _, err := SetupPropertyTestClientWithContext(t, stateFilePath, ctx)
	if err != nil {
		cancel()      // Cancel the context if client setup fails
		tempCleanup() // Clean up temp files
		return nil, nil, nil, nil, fmt.Errorf("failed to setup property test client: %w", err)
	}

	// Combine client/context cleanup with temp state cleanup
	fullCleanup := func() {
		t.Logf("Running combined cleanup for SetupTestClientWithLongTimeout")
		if cancel != nil {
			cancel()
		}
		if mcpClient != nil {
			mcpClient.Close()
		}
		tempCleanup() // Call the temp state cleanup
	}

	return mcpClient, ctx, cancel, fullCleanup, nil
}

// FlashcardSUTFactory creates a new FlashcardSUT instance for testing
func FlashcardSUTFactory(mcpClient *client.Client, ctx context.Context, cancel context.CancelFunc, tempCleanup func(), t *testing.T) *FlashcardSUT {
	t.Helper()
	// Note: The caller of this factory is responsible for ensuring tempCleanup is called.
	// This factory doesn't manage the lifecycle itself.
	return &FlashcardSUT{
		Client:             mcpClient,
		Ctx:                ctx,
		Cancel:             cancel,
		tempDirCleanupFunc: tempCleanup, // Assign the temp dir cleanup
		T:                  t,
	}
}

// Define empty state and system under test types for simple tests if needed
// (Usually not needed if using the main property test framework)
type NopState struct{}

func (s NopState) NextState(cmd commands.Command, result commands.Result) commands.State { return s }

type NopSUT struct{}

func (s NopSUT) Run(cmd commands.Command) commands.Result { return nil }
