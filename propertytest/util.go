// Package propertytest provides property-based tests for the flashcards MCP service.
package propertytest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// setupPropertyTestClient sets up an MCP client for a single property test run.
// It ensures a clean state by using a new temporary file for each run.
// Returns *client.Client instead of *client.StdioMCPClient
func SetupPropertyTestClient(t *testing.T) (mcpClient *client.Client, ctx context.Context, cancel context.CancelFunc, cleanup func()) {
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

	// Get working directory and binary path
	wd, err := os.Getwd()
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Move up to the flashcards directory if we're in the propertytest directory
	if filepath.Base(wd) == "propertytest" {
		wd = filepath.Dir(wd)
	}

	// Determine if we need to build the binary first
	binPath := filepath.Join(wd, "flashcards")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// Build the binary
		buildCmd := exec.Command("go", "build", "-o", binPath)
		buildCmd.Dir = wd // Set working directory for the build command
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
	ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second) // Increased timeout from 30s to 120s

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

// GenNonEmptyString generates non-empty strings for card content.
func GenNonEmptyString(maxLength int) gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= maxLength
	}).WithLabel("NonEmptyString")
}

// GenTags generates a slice of unique, non-empty strings for tags.
func GenTags(maxTags int, maxTagLength int) gopter.Gen {
	return gen.SliceOf(GenNonEmptyString(maxTagLength)).
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

// --- Helper functions ---

// CompareTags handles the comparison of tag slices, treating nil and empty slices as equivalent.
func CompareTags(expected, actual []string) bool {
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

// InterfaceSlice converts a string slice to an interface slice for MCP calls
func InterfaceSlice(strings []string) []interface{} {
	interfaces := make([]interface{}, len(strings))
	for i, s := range strings {
		interfaces[i] = s
	}
	return interfaces
}

// BuildBinary ensures the flashcards binary is built and returns its path
func BuildBinary(t *testing.T) string {
	t.Helper()

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	t.Logf("Current directory: %s", wd)

	// Move up to the flashcards directory if we're in the propertytest directory
	parentDir := wd
	if filepath.Base(wd) == "propertytest" {
		parentDir = filepath.Dir(wd)
	}
	t.Logf("Parent directory: %s", parentDir)

	// Set path to binary and build directory
	binPath := filepath.Join(parentDir, "flashcards")
	buildDir := filepath.Join(parentDir, "cmd", "flashcards")
	t.Logf("Binary path: %s", binPath)
	t.Logf("Build directory: %s", buildDir)

	// Build the binary using the cmd/flashcards directory
	buildCmd := exec.Command("go", "build", "-o", binPath)
	buildCmd.Dir = buildDir // Set working directory for the build command
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build flashcards binary: %v\nOutput: %s", err, string(buildOutput))
	}
	t.Logf("Successfully built flashcards binary at %s", binPath)

	return binPath
}

// --- More Generators ---

// GenRating generates a random rating from the valid FSRS ratings (Again, Hard, Good, Easy)
func GenRating() gopter.Gen {
	return gen.OneConstOf(gofsrs.Again, gofsrs.Hard, gofsrs.Good, gofsrs.Easy).
		WithLabel("FSRSRating")
}

// GenMaybeString generates a string pointer that is sometimes nil.
func GenMaybeString(maxLength int) gopter.Gen {
	return gopter.CombineGens(
		gen.Bool(),                   // Decide if the string should be generated
		GenNonEmptyString(maxLength), // Generate the actual string
	).Map(func(values []interface{}) *string {
		shouldGenerate := values[0].(bool)
		if !shouldGenerate {
			return nil
		}
		str := values[1].(string)
		return &str
	})
}

// GenMaybeTags generates a tag slice pointer that is sometimes nil.
func GenMaybeTags(maxTags int, maxTagLength int) gopter.Gen {
	return gopter.CombineGens(
		gen.Bool(),                     // Decide if tags should be generated
		GenTags(maxTags, maxTagLength), // Generate the actual tags
	).Map(func(values []interface{}) *[]string {
		shouldGenerate := values[0].(bool)
		if !shouldGenerate {
			return nil
		}
		tags := values[1].([]string)
		return &tags
	})
}
