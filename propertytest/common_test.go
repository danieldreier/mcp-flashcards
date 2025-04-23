// Package propertytest provides property-based tests for the flashcards MCP service.
package propertytest

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
)

// TestCommon provides basic wrappers to other functions in the propertytest package
// that make individual test files compilable when run directly.
// This makes it possible to run individual test files with `go test -run TestFSRSNewCardTransitions`.

// SetupTestClient is a wrapper around SetupPropertyTestClient that should be used by
// individual test files to ensure they are individually compilable.
func SetupTestClient(t *testing.T) (mcpClient *client.Client, ctx context.Context, cancel context.CancelFunc, cleanup func()) {
	return SetupPropertyTestClient(t)
}

// SetupTestClientWithLongTimeout creates a test client with a longer timeout for more complex test scenarios.
// This is useful for tests that perform multiple operations like sequential reviews or complex transitions.
func SetupTestClientWithLongTimeout(t *testing.T, timeoutSeconds int) (mcpClient *client.Client, ctx context.Context, cancel context.CancelFunc, cleanup func()) {
	// Use the base setup from SetupPropertyTestClient but override the context timeout
	mcpClient, _, cancel, cleanup = SetupPropertyTestClient(t)

	// Create a new context with the longer timeout
	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)

	return mcpClient, ctx, cancel, cleanup
}

// FlashcardSUTFactory creates a new FlashcardSUT instance for testing
func FlashcardSUTFactory(client *client.Client, ctx context.Context, cancel context.CancelFunc, cleanup func(), t *testing.T) *FlashcardSUT {
	return &FlashcardSUT{
		Client:      client,
		Ctx:         ctx,
		Cancel:      cancel,
		CleanupFunc: cleanup,
		T:           t,
	}
}
