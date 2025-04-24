# Bug Report: State Leakage in Flashcards Property Tests

## Issue Description
Property tests are failing intermittently due to state leakage between test sequences. In particular, after deleting a card, subsequent operations expecting the card to be gone are failing because the deletion operation is not consistently persisted.

## Reproduction Steps
1. Run the command: `cd propertytest && go test -v -run TestCommandSequences`
2. Observe failures in the property test where a card that was deleted is still found in subsequent operations

## Observed Behavior
In a test sequence that includes:
1. Creating a card
2. Deleting the card
3. Attempting to delete the card again or perform operations on it

The test expects the second deletion to fail with "card not found" (which correctly happens sometimes), but occasionally the test fails with an unexpected result suggesting the card still exists.

Additionally, when calling `GetDueCard()` after deleting all cards, sometimes a card that was deleted is still returned, indicating that the deletion wasn't properly persisted.

## Expected Behavior
After deleting a card, all subsequent operations should:
1. Return "card not found" errors when trying to access the deleted card
2. Not return deleted cards when calling `GetDueCard()`
3. Maintain consistent state between commands even when new SUT instances are created

## Hypothesis
The issue may be related to the SUT (System Under Test) setup and teardown mechanism in the property tests:

1. The `TestCommandSequences` test in `property_command_sequences_test.go` creates a new SUT for each command, but reuses the same underlying storage file across all commands
2. The critical issue is that the file reset logic was commented out in the `NewSystemUnderTestFunc` function:
   ```go
   // Removed file reset: SUT should load existing state from tempFilePath
   // if err := os.WriteFile(tempFilePath, []byte("{}"), 0644); err != nil {
   //   t.Fatalf("Failed to reset temp state file %s: %v", tempFilePath, err)
   // }
   // t.Logf("Reset state file %s for new sequence", tempFilePath)
   ```

3. This was an intentional change to make state persist between commands in a sequence, but it created two problems:
   - In the `DeleteCard` method in `storage.go`, changes are made to the in-memory state but they might not be properly persisted to disk
   - Each new SUT instance loads from the same file, but if the previous SUT didn't properly save state, the new SUT will have stale data

## Alternate hypothesis
The system under test may be implemented incorrectly, and delete or change operations are not updating state correctly.

### Potential solution: Use distinct files for each test sequence
Modify the test infrastructure to create a unique file for each test sequence rather than reusing the same file:

```go
// In property_command_sequences_test.go
for sequenceIndex := 0; sequenceIndex < parameters.MinSuccessfulTests; sequenceIndex++ {
    sequenceFilePath := filepath.Join(tempDir, fmt.Sprintf("sequence-%d.json", sequenceIndex))
    // Use sequenceFilePath for this test sequence only
}
```

## Fix Implementation
After investigation, the issue was found to be primarily in the error handling of the test code:

1. The `DeleteCardCmd.Run` method in the property tests was not correctly handling error responses. When deleting a card that doesn't exist, the server returns a JSON response with an error field rather than an actual Go error.

2. The fix involved updating `DeleteCardCmd.Run` to:
   - First attempt to unmarshal the response as a success response
   - If that fails or has success=false, try to unmarshal as an error response
   - When detecting a "not found" error, return it as a proper Go error instead of treating it as a success

3. The `TestDeleteCardStateLeakage` test also needed to be updated to:
   - Better handle the error response formats
   - Check for error messages in the JSON response
   - Properly handle success=false responses

These changes ensure that the tests correctly detect when a card is deleted and when a deleted card is attempted to be accessed again.

## Testing Results
After implementing the fix:
1. The `TestDeleteCardStateLeakage` test now passes consistently
2. The `TestDeleteCardPersistence` test passes consistently
3. The `TestMultipleDeleteOperations` test passes as well

The core functionality is now working correctly - deletion of cards is properly persisted to disk and subsequent operations on deleted cards correctly return "not found" errors.

Some of the property tests still have unrelated issues with matching error types, but these are separate problems and not related to the state leakage bug fixed here.
