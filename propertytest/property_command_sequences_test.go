package propertytest

import (
	"reflect"
	"sort"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/commands"
	"github.com/leanovate/gopter/gen"

	// Corrected import path for go-fsrs
	gofsrs "github.com/open-spaced-repetition/go-fsrs"
)

// TestCommandSequences verifies the consistency of the system through random command sequences.
func TestCommandSequences(t *testing.T) {
	// // Reset command counts - Removed, logic needs review/replacement
	// testutil.ResetCommandCounts()

	parameters := gopter.DefaultTestParameters()
	// Set minimum success count and max size for efficient testing
	parameters.MinSuccessfulTests = 10 // Adjust as needed
	parameters.MaxSize = 10            // Adjust as needed

	// Define ProtoCommands locally to capture 't'
	protoCmds := &commands.ProtoCommands{
		// Initial state generator now includes the testing context
		InitialStateGen: gen.Const(&CommandState{
			Cards:        make(map[string]Card),
			KnownRealIDs: []string{},
			T:            t, // Inject t here
		}),

		// SUT creation now uses the captured 't' directly
		NewSystemUnderTestFunc: func(initialState commands.State) commands.SystemUnderTest {
			t.Logf("Creating New System Under Test...")
			// Use captured 't' instead of initialState.T
			mcpClient, ctx, cancel, baseCleanup := SetupPropertyTestClient(t)

			combinedCleanup := func() {
				t.Logf("Running SUT cleanup...")
				cancel()
				mcpClient.Close()
				baseCleanup()
			}
			cleanupMapMutex.Lock()
			cleanupMap[mcpClient] = combinedCleanup
			cleanupMapMutex.Unlock()

			// Cast initialState to access model data if needed by SUT, but T comes from captured t
			_ = initialState.(*CommandState)

			return &FlashcardSUT{
				Client:      mcpClient,
				Ctx:         ctx,
				Cancel:      cancel,
				CleanupFunc: combinedCleanup,
				T:           t, // Assign captured t to SUT
			}
		},

		// Copy DestroySystemUnderTestFunc logic
		DestroySystemUnderTestFunc: func(sut commands.SystemUnderTest) {
			fSUT := sut.(*FlashcardSUT)
			fSUT.T.Logf("Destroying System Under Test...")
			cleanupMapMutex.Lock()
			if cleanupFunc, ok := cleanupMap[fSUT.Client]; ok {
				cleanupFunc()
				delete(cleanupMap, fSUT.Client)
			} else {
				fSUT.T.Logf("Warning: Cleanup function not found for SUT Client %p", fSUT.Client)
			}
			cleanupMapMutex.Unlock()
		},

		// Copy GenCommandFunc logic
		GenCommandFunc: func(state commands.State) gopter.Gen {
			cmdState := state.(*CommandState)
			weightedGens := []gen.WeightedGen{
				// Weight: 5 - CreateCardCmd
				{Weight: 5, Gen: gopter.CombineGens(GenNonEmptyString(100), GenNonEmptyString(200), GenTags(5, 20)).
					Map(func(v []interface{}) commands.Command {
						return &CreateCardCmd{Front: v[0].(string), Back: v[1].(string), Tags: v[2].([]string)}
					})},
				// Weight: 2 - GetDueCardCmd
				{Weight: 2, Gen: gopter.CombineGens(
					gen.SliceOf(GenNonEmptyString(20)).Map(func(tags []string) []string {
						uniqueTags := make(map[string]struct{})
						result := []string{}
						for _, tag := range tags {
							if _, exists := uniqueTags[tag]; !exists {
								uniqueTags[tag] = struct{}{}
								result = append(result, tag)
							}
						}
						sort.Strings(result)
						return result
					}),
				).Map(func(values []interface{}) commands.Command {
					filterTags := values[0].([]string)
					sample, ok := gen.Bool().Sample()
					if ok && len(filterTags) > 0 && sample.(bool) {
						filterTags = []string{}
					}
					return &GetDueCardCmd{FilterTags: filterTags}
				})},
			}

			if len(cmdState.KnownRealIDs) > 0 {
				// Convert known IDs to []interface{} for OneConstOf
				knownIDsInterfaces := make([]interface{}, len(cmdState.KnownRealIDs))
				for i, id := range cmdState.KnownRealIDs {
					knownIDsInterfaces[i] = id
				}
				// Generate a random known ID (as interface{})
				randomIDGen := gen.OneConstOf(knownIDsInterfaces...)

				// Use FlatMap to ensure ID is generated and asserted before combining

				// Weight: 2 - GetCardCmd
				getCardGen := randomIDGen.FlatMap(func(idVal interface{}) gopter.Gen {
					cardID, ok := idVal.(string)
					if !ok {
						return gen.Fail(reflect.TypeOf((*GetCardCmd)(nil)))
					} // Fail if type assertion fails
					return gen.Const(&GetCardCmd{CardID: cardID})
				}, reflect.TypeOf((*GetCardCmd)(nil)))
				weightedGens = append(weightedGens, gen.WeightedGen{Weight: 2, Gen: getCardGen})

				// Weight: 1 - DeleteCardCmd
				deleteCardGen := randomIDGen.FlatMap(func(idVal interface{}) gopter.Gen {
					cardID, ok := idVal.(string)
					if !ok {
						return gen.Fail(reflect.TypeOf((*DeleteCardCmd)(nil)))
					}
					return gen.Const(&DeleteCardCmd{CardID: cardID})
				}, reflect.TypeOf((*DeleteCardCmd)(nil)))
				weightedGens = append(weightedGens, gen.WeightedGen{Weight: 1, Gen: deleteCardGen})

				// Weight: 3 - UpdateCardCmd
				updateCardGen := randomIDGen.FlatMap(func(idVal interface{}) gopter.Gen {
					cardID, ok := idVal.(string)
					if !ok {
						return gen.Fail(reflect.TypeOf((*UpdateCardCmd)(nil)))
					}
					// Now combine the known string ID with other generators
					return gopter.CombineGens(GenMaybeString(100), GenMaybeString(200), GenMaybeTags(5, 20)).
						SuchThat(func(values []interface{}) bool { return values[0] != nil || values[1] != nil || values[2] != nil }).
						Map(func(values []interface{}) commands.Command {
							return &UpdateCardCmd{
								CardID:   cardID, // Use the cardID from FlatMap scope
								NewFront: values[0].(*string),
								NewBack:  values[1].(*string),
								NewTags:  values[2].(*[]string),
							}
						})
				}, reflect.TypeOf((*UpdateCardCmd)(nil)))
				weightedGens = append(weightedGens, gen.WeightedGen{Weight: 3, Gen: updateCardGen})

				// Weight: 10 - SubmitReviewCmd
				submitReviewGen := randomIDGen.FlatMap(func(idVal interface{}) gopter.Gen {
					cardID, ok := idVal.(string)
					if !ok {
						return gen.Fail(reflect.TypeOf((*SubmitReviewCmd)(nil)))
					}
					// Now combine the known string ID with other generators
					return gopter.CombineGens(GenRating(), GenNonEmptyString(100)).
						Map(func(values []interface{}) commands.Command {
							rating := values[0].(gofsrs.Rating)
							answer := values[1].(string)
							return &SubmitReviewCmd{
								CardID: cardID, // Use the cardID from FlatMap scope
								Rating: rating,
								Answer: answer,
							}
						})
				}, reflect.TypeOf((*SubmitReviewCmd)(nil)))
				weightedGens = append(weightedGens, gen.WeightedGen{Weight: 10, Gen: submitReviewGen})
			}

			return gen.Weighted(weightedGens)
		},
	}

	properties := gopter.NewProperties(parameters)

	// Pass the locally defined protoCmds to commands.Prop
	properties.Property("command sequences preserve consistency", commands.Prop(protoCmds))

	properties.TestingRun(t)

	// Statistics section commented out until replacement for testutil is found
	/*
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
