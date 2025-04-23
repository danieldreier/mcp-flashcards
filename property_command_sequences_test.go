package propertytest

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/commands"
)

// TestCommandSequences verifies the consistency of the system through random command sequences.
func TestCommandSequences(t *testing.T) {
	// // Reset all command counts to get accurate statistics - Commented out for now
	// testutil.ResetCommandCounts()

	parameters := gopter.DefaultTestParameters()
	// Set minimum success count and max size for efficient testing
	parameters.MinSuccessfulTests = 10 // Adjust as needed
	parameters.MaxSize = 10            // Adjust as needed

	properties := gopter.NewProperties(parameters)

	// Use PropStateful to provide an initial state with the testing context
	initialStateFunc := func() commands.State {
		return &CommandState{
			Cards:        make(map[string]Card),
			KnownRealIDs: []string{},
			T:            t, // Inject the testing context here
		}
	}

	// FlashcardProtoCommands now provides NewSystemUnderTestFunc, DestroySystemUnderTestFunc, GenCommandFunc
	properties.Property("command sequences preserve consistency", commands.PropStateful(FlashcardProtoCommands, initialStateFunc))

	properties.TestingRun(t)

	/* // Commented out until replacement for testutil is found
	// Check if all commands were executed at least once and print statistics
	executedCmds := testutil.GetExecutedCommands()
	allCmds := []string{"CreateCardCmd", "GetCardCmd", "UpdateCardCmd", "DeleteCardCmd", "SubmitReviewCmd", "GetDueCardCmd"}

	// Log execution statistics
	fmt.Println("\nCommand Execution Statistics:")
	for cmd, count := range executedCmds {
		fmt.Printf("  %s: executed %d times\n", cmd, count)
	}

	// Check for commands that were never executed
	var notExecuted []string
	for _, cmd := range allCmds {
		if _, ok := executedCmds[cmd]; !ok {
			notExecuted = append(notExecuted, cmd)
		}
	}

	if len(notExecuted) > 0 {
		fmt.Printf("\nWARNING: The following commands were never executed: %v\n", notExecuted)
		t.Logf("Not all commands were executed in the property test. Consider increasing test count or adjusting generators.")
	}
	*/
}
