// Package propertytest provides property-based tests for the flashcards MCP service.
package propertytest

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/leanovate/gopter/commands"
	"github.com/mark3labs/mcp-go/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	// Initialize Zap logger
	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Colorful levels
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder        // Standard time format
	logConfig.EncoderConfig.CallerKey = ""                                 // Don't log caller

	// Set log level based on environment variable
	if os.Getenv("MCP_DEBUG") == "true" {
		t.Log("MCP_DEBUG is true, setting log level to Debug")
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		t.Log("MCP_DEBUG is not set or false, setting log level to Info")
		logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := logConfig.Build(zap.AddStacktrace(zapcore.ErrorLevel)) // Add stacktrace for errors
	if err != nil {
		t.Fatalf("Failed to create zap logger: %v", err)
	}

	// Ensure logger is synced on cleanup
	originalCleanup := tempCleanup
	syncedCleanup := func() {
		if originalCleanup != nil {
			originalCleanup()
		}
		_ = logger.Sync() // Ignore sync errors
	}

	// Note: The caller of this factory is responsible for ensuring tempCleanup is called.
	// This factory doesn't manage the lifecycle itself.
	return &FlashcardSUT{
		Client:             mcpClient,
		Ctx:                ctx,
		Cancel:             cancel,
		tempDirCleanupFunc: syncedCleanup, // Use cleanup func that also syncs logger
		T:                  t,
		Logger:             logger, // Assign the created logger
	}
}

// Define empty state and system under test types for simple tests if needed
// (Usually not needed if using the main property test framework)
type NopState struct{}

func (s NopState) NextState(cmd commands.Command, result commands.Result) commands.State { return s }

type NopSUT struct{}

func (s NopSUT) Run(cmd commands.Command) commands.Result { return nil }
