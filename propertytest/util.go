// Package propertytest provides property-based tests for the flashcards MCP service.
package propertytest

import (
	"context"
	"fmt"
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

// CreateTempStateFile creates a new unique temporary directory and an empty state file within it.
// It returns the path to the temporary directory, the path to the state file,
// a cleanup function to remove the directory, and an error if creation fails.
// Each call creates a completely isolated environment.
func CreateTempStateFile(t *testing.T) (tempDir string, stateFilePath string, cleanup func(), err error) {
	t.Helper()

	// Create a new unique temporary directory for this specific SUT instance.
	// The pattern includes the test name for easier identification, but uniqueness
	// is guaranteed by os.MkdirTemp.
	// safeTestName := strings.ReplaceAll(t.Name(), "/", "_") // Keep for logging if needed, but not strictly necessary for MkdirTemp uniqueness
	tempDir, err = os.MkdirTemp("", "flashcards-sut-state-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Define the state file path within the unique temp directory.
	stateFilePath = filepath.Join(tempDir, "flashcards-test.json")

	// Initialize the state file with an empty JSON object.
	err = os.WriteFile(stateFilePath, []byte("{}"), 0644)
	if err != nil {
		os.RemoveAll(tempDir) // Clean up the directory if file creation fails
		return "", "", nil, fmt.Errorf("failed to initialize state file %s: %w", stateFilePath, err)
	}

	t.Logf("Created unique state file for SUT instance in test %s at %s (within %s)", t.Name(), stateFilePath, tempDir)

	// Define the cleanup function specific to this temporary directory.
	cleanup = func() {
		t.Logf("Cleaning up temp directory for SUT instance: %s", tempDir)
		err := os.RemoveAll(tempDir)
		if err != nil {
			// Log the error but don't fail the test, as cleanup failure
			// might happen after the test itself has passed/failed.
			t.Logf("Warning: failed to remove temp directory %s: %v", tempDir, err)
		}
	}

	return tempDir, stateFilePath, cleanup, nil
}

// SetupPropertyTestClient sets up an MCP client using a specific data file path.
// The caller is responsible for managing the lifecycle of the temp file via the cleanup function.
func SetupPropertyTestClient(t *testing.T, stateFilePath string) (mcpClient *client.Client, ctx context.Context, cancel context.CancelFunc, err error) {
	t.Helper()

	// Ensure the state file exists (it should have been created by CreateTempStateFile)
	// Declare statErr here
	var statErr error
	_, statErr = os.Stat(stateFilePath)
	if os.IsNotExist(statErr) {
		// This indicates a programming error - CreateTempStateFile should always be called first.
		return nil, nil, nil, fmt.Errorf("SetupPropertyTestClient called with non-existent file: %s", stateFilePath)
	} else if statErr != nil {
		return nil, nil, nil, fmt.Errorf("error checking state file %s: %w", stateFilePath, statErr)
	}

	// Get working directory and binary path
	wd, err := os.Getwd()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Move up to the project root directory if we're in the propertytest directory
	if filepath.Base(wd) == "propertytest" {
		wd = filepath.Dir(wd)
	}

	// Determine if we need to build the binary first
	binPath := filepath.Join(wd, "flashcards") // Assumes binary is in the root
	_, statErr = os.Stat(binPath)
	shouldBuild := os.IsNotExist(statErr)

	// If the binary doesn't exist OR if the Makefile exists and is newer than the binary
	makefileInfo, makefileStatErr := os.Stat(filepath.Join(wd, "Makefile"))
	if !shouldBuild && makefileStatErr == nil {
		binaryInfo, binStatErr := os.Stat(binPath)
		if binStatErr == nil && makefileInfo.ModTime().After(binaryInfo.ModTime()) {
			t.Logf("Makefile is newer than binary, rebuilding...")
			shouldBuild = true
		}
	}

	if shouldBuild {
		// Build the binary if it doesn't exist or needs rebuilding
		t.Logf("Building flashcards binary at %s", binPath)
		// Use 'make build' if Makefile exists, otherwise 'go build'
		var buildCmd *exec.Cmd
		if makefileStatErr == nil {
			buildCmd = exec.Command("make", "build")
			buildCmd.Dir = wd // Run make from the project root
		} else {
			buildCmd = exec.Command("go", "build", "-o", binPath, filepath.Join("cmd", "flashcards"))
			buildCmd.Dir = wd // Run go build from the project root
		}

		buildOutput, buildErr := buildCmd.CombinedOutput()
		if buildErr != nil {
			// Return error instead of calling t.Fatalf
			return nil, nil, nil, fmt.Errorf("failed to build flashcards binary: %v\nOutput: %s", buildErr, buildOutput)
		}
		t.Logf("Successfully built flashcards binary.")
	}

	// Create the MCP client targeting the server binary with the specific temp file
	mcpClient, err = client.NewStdioMCPClient(
		binPath, // Use the binary directly
		[]string{"PYTHONUNBUFFERED=1", "GODEBUG=asyncpreemptoff=1"}, // Force unbuffered IO
		"-file",
		stateFilePath, // Use the provided state file path
	)
	if err != nil {
		// Return error instead of calling t.Fatalf
		return nil, nil, nil, fmt.Errorf("failed to create client with file %s: %w", stateFilePath, err)
	}

	// Create context with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)

	// Initialize the client
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "flashcards-property-test-client",
		Version: "0.1.0",
	}

	_, initErr := mcpClient.Initialize(ctx, initRequest)
	if initErr != nil {
		mcpClient.Close() // Close the client if initialization fails
		cancel()          // Cancel the context
		// Return error instead of calling t.Fatalf
		return nil, nil, nil, fmt.Errorf("failed to initialize MCP client with file %s: %w", stateFilePath, initErr)
	}

	return mcpClient, ctx, cancel, nil // Return nil error on success
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
// This function is DEPRECATED, building is now handled within SetupPropertyTestClient.
// Keeping it temporarily for reference or potential reuse if needed outside Setup.
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
