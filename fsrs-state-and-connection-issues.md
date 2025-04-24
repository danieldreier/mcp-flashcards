# Bug Report: FSRS State Transitions and Connection Issues in Property Tests

## Issue Summary
Two distinct classes of issues were identified while running property tests:

1. ~~**Connection/Transport Errors**: Several tests fail with "transport error: context canceled"~~ **FIXED**
2. **FSRS State Transition Discrepancies**: Tests expecting FSRS state 2 for "Good" ratings receive state 1 instead

## Affected Tests

### ~~Connection/Transport Issues~~ FIXED
~~These tests fail with "transport error: context canceled" errors:~~
- ~~`TestDeadlock`~~
- ~~`TestFSRSModelComparison`~~
- ~~`TestFSRSSequentialReviews`~~
- ~~`TestStateIsolation`~~
- ~~`TestMultipleDeletes`~~
- ~~`TestMultipleDeletes_Sequential`~~

**Note: Connection/Transport issues have been fixed. All tests now run without transport errors.**

### FSRS State Transition Issues
These tests fail with incorrect FSRS state transitions:
- `TestFSRSNewCardTransitions`: Rating 3 expected state 2, got state 1
- `TestFSRSNewCardGood`: Good rating expected state 2, got state 1
- `TestSubmitReviewCommand`: "New_Card_Rated_Good" expected state 2, got state 1

## Detailed Observations

### ~~Transport Error Issues~~ FIXED
```
// Example from TestFSRSSequentialReviews
commands.go:68: Run: CreateCard(Front: 'Sequential Review Front', Back: 'Sequential Review Back', Tags: [sequential-test])
fsrs_sequential_reviews_test.go:31: Failed to create card: create_card Run failed: transport error: context canceled
```

~~The transport error suggests that either:~~
1. ~~A timeout is occurring during test execution~~
2. ~~A context is being canceled prematurely~~
3. ~~The MCP client or server is closing connections unexpectedly~~

**FIX IMPLEMENTED:** The issue was in how contexts were being managed in the test setup. The `SetupTestClientWithLongTimeout` function had a race condition where it would cancel the original context before creating a new one, potentially leaving operations in a canceled state. Also, during the handoff between context creation and usage, there was a time when the context might be canceled.

The solution involved:
1. Creating a shared implementation (`setupPropertyTestClientImpl`) that separates context creation from client setup
2. Creating a new function (`SetupPropertyTestClientWithContext`) that accepts an external context
3. Modifying `SetupTestClientWithLongTimeout` to create the context first, then pass it to the implementation

This approach ensures that the same context is used consistently throughout the test, eliminating the race condition and premature cancellation.

### FSRS State Transition Issues
```
// Example from TestFSRSNewCardGood
fsrs_new_good_test.go:62: Rating: Good -> State: 1
fsrs_new_good_test.go:64: Rating Good: Expected state 2, got 1
fsrs_new_good_test.go:70: Due date for New->Good rating should be at least ~1 day, got 2025-04-24 10:21:19.483272
```

The tests expect that a "Good" rating (3) for a new card should transition it to state 2 (Review), but it's transitioning to state 1 (Learning) instead. The due date is also much sooner than expected (10 minutes instead of ~1 day).

Interestingly, in the raw FSRS library behavior check, the expected behavior is correctly demonstrated:
```
property_submit_review_test.go:47: Raw go-fsrs for new card, rating 3: State=1, Due=2025-04-24T10:21:19-04:00 (Interval: 10m0s)
property_submit_review_test.go:47: Raw go-fsrs for new card, rating 4: State=2, Due=2025-05-05T10:11:19-04:00 (Interval: 264h0m0s)
```

However, when the test performs the actual transition through the MCP service, the transition behavior differs from expectations.

## Impact
- ~~**Property Tests Reliability**: The connection issues prevent reliable execution of property tests~~ **FIXED**
- **FSRS Algorithm Accuracy**: The state transition discrepancies could lead to incorrect scheduling of cards in the actual application

## Reproduction Steps
1. Run property tests with `go test ./propertytest -v`
2. ~~Observe connection errors~~ **No longer occurs**
3. Observe FSRS state discrepancies in test output

## ~~Possible Causes~~ Solutions

### ~~For Transport Errors~~ FIXED
1. ~~Context timeouts may be too short for the operations~~
2. ~~The MCP service may be terminating unexpectedly~~
3. ~~There may be a deadlock in the client/server communication~~

The issue was fixed by properly handling contexts in the test setup, eliminating race conditions and ensuring that operations have a consistent context throughout their lifecycle.

### For FSRS State Discrepancies
1. The FSRS implementation in the actual service may differ from expectations
2. Test expectations may be outdated compared to current FSRS algorithm behavior
3. The service might be using different FSRS parameters than the test expects

## Next Steps
1. ~~**Transport Issues**: Investigate context lifetimes and cancellation patterns in tests~~ **COMPLETED**
2. **FSRS State Issues**: Compare raw FSRS library behavior with service implementation
3. Confirm whether test expectations are aligned with the current FSRS algorithm specification 