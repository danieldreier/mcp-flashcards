package propertytest

import (
	"testing"

	"github.com/leanovate/gopter"
	gopterCmds "github.com/leanovate/gopter/commands" // Renamed import
	"github.com/leanovate/gopter/gen"
)

// TestCommandSequences verifies the consistency of the system through random command sequences.
func TestCommandSequences(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 30
	parameters.MaxSize = 15
	// parameters.Rng = gopter.NewStdRandSource(1234) // For deterministic runs

	properties := gopter.NewProperties(parameters)

	// Define the function to create the system under test
	newSystemUnderTest := func(initialState gopterCmds.State) gopterCmds.SystemUnderTest {
		// Ensure the state contains the testing context
		cmdState, ok := initialState.(*CommandState)
		if !ok || cmdState.T == nil {
			outerT := t
			if !ok {
				outerT.Logf("Warning: Initial state is not *CommandState, type is %T", initialState)
			} else {
				outerT.Logf("Warning: Initial state *CommandState is missing *testing.T")
			}
			outerT.Fatalf("Cannot create SUT without *testing.T in initial state")
			return nil
		}
		testingT := cmdState.T // Use the *testing.T from the state

		// Create a unique temporary state file and directory for this SUT instance
		_, stateFilePath, tempCleanup, err := CreateTempStateFile(testingT)
		if err != nil {
			testingT.Fatalf("Failed to create temp state file for SUT: %v", err)
			return nil
		}

		// Setup client with the unique state file
		mcpClient, ctx, cancel, err := SetupPropertyTestClient(testingT, stateFilePath)
		if err != nil {
			tempCleanup() // Clean up the temp directory if client setup fails
			testingT.Fatalf("Failed to setup property test client for SUT: %v", err)
			return nil
		}

		return &FlashcardSUT{
			Client:             mcpClient,
			Ctx:                ctx,
			Cancel:             cancel,
			tempDirCleanupFunc: tempCleanup,
			T:                  testingT,
		}
	}

	// Define the function to destroy the system under test
	destroySystemUnderTest := func(sut gopterCmds.SystemUnderTest) {
		if sut == nil {
			t.Logf("Warning: destroySystemUnderTest called with nil SUT")
			return
		}
		flashcardSUT, ok := sut.(*FlashcardSUT)
		if !ok {
			t.Logf("Warning: SUT provided to destroySystemUnderTest is not *FlashcardSUT, type is %T", sut)
			return
		}
		flashcardSUT.T.Logf("Destroying SUT...")

		// 1. Cancel the context
		if flashcardSUT.Cancel != nil {
			flashcardSUT.Cancel()
		}

		// 2. Close the client
		if flashcardSUT.Client != nil {
			if err := flashcardSUT.Client.Close(); err != nil {
				flashcardSUT.T.Logf("Warning: Error closing MCP client during SUT destruction: %v", err)
			}
		}

		// 3. Clean up the temporary state directory
		if flashcardSUT.tempDirCleanupFunc != nil {
			flashcardSUT.tempDirCleanupFunc()
		}
		flashcardSUT.T.Logf("SUT Destroyed.")
	}

	// Initial state generator
	initialStateGen := gen.Const(&CommandState{
		Cards:        make(map[string]Card),
		KnownRealIDs: []string{},
		T:            t, // Pass the test context here
	})

	// Command generator function
	commandGen := func(state gopterCmds.State) gopter.Gen {
		commandState, ok := state.(*CommandState)
		if !ok {
			t.Fatalf("Command generator received invalid state type: %T", state)
			return nil // Return nil generator on fatal error
		}

		var generators []gen.WeightedGen

		// CreateCardCmd
		generators = append(generators, gen.WeightedGen{
			Weight: 3, Gen: generateCreateCardCmd(),
		})

		// ID-based commands (only if cards exist)
		if len(commandState.KnownRealIDs) > 0 {
			generators = append(generators, gen.WeightedGen{
				Weight: 5, Gen: generateIdBasedCmd(commandState),
			})
			generators = append(generators, gen.WeightedGen{
				Weight: 2, Gen: generateUpdateCardCmd(commandState),
			})
		}

		// GetDueCardCmd
		generators = append(generators, gen.WeightedGen{
			Weight: 2, Gen: generateGetDueCardCmd(),
		})

		// Filter out commands whose preconditions fail
		return gen.Weighted(generators).SuchThat(func(cmd gopterCmds.Command) bool {
			return cmd.PreCondition(state)
		}).WithShrinker(gopter.NoShrinker)
	}

	// Create the ProtoCommands struct
	flashcardProtoCommands := &gopterCmds.ProtoCommands{
		NewSystemUnderTestFunc:     newSystemUnderTest,
		DestroySystemUnderTestFunc: destroySystemUnderTest,
		InitialStateGen:            initialStateGen,
		GenCommandFunc:             commandGen,
	}

	// Run the property test using commands.Prop
	properties.Property("command sequences preserve consistency", gopterCmds.Prop(flashcardProtoCommands))

	properties.TestingRun(t)
}
