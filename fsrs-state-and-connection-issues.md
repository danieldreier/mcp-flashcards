# Bug Report: FSRS State Transitions and Connection Issues in Property Tests

## Issue Summary
Two distinct classes of issues were identified while running property tests:

1. ~~**Connection/Transport Errors**: Several tests fail with "transport error: context canceled"~~ **FIXED**
2. **FSRS State Transition Discrepancies**: Tests expecting FSRS state 2 for "Good" ratings receive state 1 instead **FIXED**

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
- `TestFSRSNewCardTransitions`: Rating 3 expected state 2, got state 1 **FIXED**
- `TestFSRSNewCardGood`: Good rating expected state 2, got state 1 **FIXED**
- `TestSubmitReviewCommand`: "New_Card_Rated_Good" expected state 2, got state 1 **FIXED**

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

**FIX IMPLEMENTED:** After investigation, it was determined that the tests were expecting behavior that didn't match the actual behavior of the go-fsrs library. The issue wasn't with the implementation but with incorrect test expectations:

1. According to the raw go-fsrs library output, a new card with rating 3 (Good) should transition to state 1 (Learning) with a 10-minute interval
2. Only a new card with rating 4 (Easy) should transition directly to state 2 (Review)

The solution was to update the test expectations to match the actual behavior of the go-fsrs library:

1. Updated `TestFSRSNewCardGood` to expect state 1 (Learning) for a Good rating on a new card
2. Updated the due date check to expect ~10 minutes instead of ~1 day
3. Updated `TestFSRSNewCardTransitions` to expect state 1 (Learning) for a Good rating on a new card
4. Updated `TestSubmitReviewCommand` to expect state 1 (Learning) for a Good rating on a new card

After these changes, the basic FSRS state transition tests now pass. There are still failures in other tests like `TestFSRSModelComparison` and `TestFSRSSequentialReviews` which are related to additional FSRS metadata fields like Stability, Difficulty, ElapsedDays, etc. This suggests that the implementation in the service may not be maintaining all the FSRS metadata fields correctly when processing reviews.

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
1. The test expectations were misaligned with the actual go-fsrs library behavior
2. The service implementation correctly follows the go-fsrs library behavior
3. The tests needed to be updated to match the actual FSRS algorithm behavior

## Next Steps
1. ~~**Transport Issues**: Investigate context lifetimes and cancellation patterns in tests~~ **COMPLETED**
2. ~~**FSRS State Issues**: Compare raw FSRS library behavior with service implementation~~ **COMPLETED**
3. ~~**Test Expectations**: Update test expectations to match the actual FSRS behavior~~ **COMPLETED**
4. **Metadata Fields**: Further investigation is needed for the FSRS metadata fields (Stability, Difficulty, etc.) as they are not being properly maintained between reviews in the service implementation 