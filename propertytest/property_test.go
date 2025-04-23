package propertytest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestMain is the main entry point for the property tests.
// It ensures the binary is built once before running any tests.
func TestMain(m *testing.M) {
	// Get current directory
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failed to get working directory: %w", err))
	}
	fmt.Printf("Current directory: %s\n", wd)

	// Get parent directory if we're in propertytest
	parentDir := wd
	if filepath.Base(wd) == "propertytest" {
		parentDir = filepath.Dir(wd)
	}
	fmt.Printf("Parent directory: %s\n", parentDir)

	// Build the binary once for all tests
	binPath := filepath.Join(parentDir, "flashcards")
	fmt.Printf("Binary path: %s\n", binPath)

	buildCmd := exec.Command("go", "build", "-o", binPath)
	buildCmd.Dir = filepath.Join(parentDir, "cmd", "flashcards")
	fmt.Printf("Build directory: %s\n", buildCmd.Dir)

	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("failed to build binary: %w\nOutput: %s", err, string(buildOutput)))
	}
	fmt.Printf("Successfully built binary: %s\n", binPath)

	// Wait a moment for the build to complete
	time.Sleep(100 * time.Millisecond)

	// Run all tests
	os.Exit(m.Run())
}
