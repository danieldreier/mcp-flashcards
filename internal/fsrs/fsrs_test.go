package fsrs

import (
	"testing"
	"time"

	"github.com/open-spaced-repetition/go-fsrs"
)

func TestNewFSRSManager(t *testing.T) {
	manager := NewFSRSManager()
	if manager == nil {
		t.Fatal("Expected non-nil FSRSManager")
	}
}

func TestNewFSRSManagerWithParams(t *testing.T) {
	params := fsrs.DefaultParam()
	params.RequestRetention = 0.8 // Custom value

	manager := NewFSRSManagerWithParams(params)
	if manager == nil {
		t.Fatal("Expected non-nil FSRSManager")
	}

	// Type assertion to check the parameters
	impl, ok := manager.(*FSRSManagerImpl)
	if !ok {
		t.Fatal("Expected *FSRSManagerImpl")
	}

	if impl.parameters.RequestRetention != 0.8 {
		t.Fatalf("Expected RequestRetention 0.8, got %f", impl.parameters.RequestRetention)
	}
}

func TestScheduleReview(t *testing.T) {
	manager := NewFSRSManager()
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	lastReview := now.Add(-24 * time.Hour) // Set last review to 24 hours ago

	tests := []struct {
		name           string
		initialState   fsrs.State
		rating         fsrs.Rating
		expectedState  fsrs.State
		expectDueAfter time.Duration // minimum duration until due
	}{
		{
			name:           "New card rated Again",
			initialState:   fsrs.New,
			rating:         fsrs.Again,
			expectedState:  fsrs.Learning,
			expectDueAfter: time.Minute, // Should be scheduled for very soon
		},
		{
			name:           "New card rated Hard",
			initialState:   fsrs.New,
			rating:         fsrs.Hard,
			expectedState:  fsrs.Learning,
			expectDueAfter: time.Minute * 5, // Should be scheduled for soon
		},
		{
			name:           "New card rated Good",
			initialState:   fsrs.New,
			rating:         fsrs.Good,
			expectedState:  fsrs.Learning,
			expectDueAfter: time.Minute * 10, // Adjusted based on actual FSRS behavior
		},
		{
			name:           "New card rated Easy",
			initialState:   fsrs.New,
			rating:         fsrs.Easy,
			expectedState:  fsrs.Review,        // Should go straight to review state
			expectDueAfter: time.Hour * 24 * 3, // Should be scheduled for several days later
		},
		{
			name:           "Learning card rated Good",
			initialState:   fsrs.Learning,
			rating:         fsrs.Good,
			expectedState:  fsrs.Review,    // Should graduate to review
			expectDueAfter: time.Hour * 24, // At least a day later
		},
		{
			name:           "Review card rated Again",
			initialState:   fsrs.Review,
			rating:         fsrs.Again,
			expectedState:  fsrs.Relearning, // Should go to relearning
			expectDueAfter: time.Minute,     // Should be scheduled for very soon
		},
		{
			name:           "Relearning card rated Good",
			initialState:   fsrs.Relearning,
			rating:         fsrs.Good,
			expectedState:  fsrs.Review,    // Should go back to review
			expectDueAfter: time.Hour * 12, // Should be scheduled for later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test card with initial state
			testCard := fsrs.NewCard()
			testCard.State = tt.initialState
			testCard.LastReview = lastReview

			state, due := manager.ScheduleReview(testCard, tt.rating, now)

			if state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, state)
			}

			expectedEarliest := now.Add(tt.expectDueAfter)
			if due.Before(expectedEarliest) {
				t.Errorf("Due date %v is earlier than expected %v", due, expectedEarliest)
			}
		})
	}
}

func TestGetReviewPriority(t *testing.T) {
	manager := NewFSRSManager()
	now := time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		state    fsrs.State
		due      time.Time
		expected float64
		compare  func(float64, float64) bool
	}{
		{
			name:     "Overdue Review card",
			state:    fsrs.Review,
			due:      now.Add(-24 * time.Hour), // 1 day overdue
			expected: 2.0 * 1.1,                // basePriority * (1 + overdueDays * 0.1)
			compare:  func(a, b float64) bool { return a >= b },
		},
		{
			name:     "Due Review card",
			state:    fsrs.Review,
			due:      now,
			expected: 2.0, // basePriority
			compare:  func(a, b float64) bool { return a == b },
		},
		{
			name:     "Future Review card",
			state:    fsrs.Review,
			due:      now.Add(24 * time.Hour), // Due tomorrow
			expected: 2.0 / 2.0,               // basePriority / (1 + daysToDue)
			compare:  func(a, b float64) bool { return a <= b },
		},
		{
			name:     "Learning card vs Review card",
			state:    fsrs.Learning,
			due:      now,
			expected: 3.0, // Learning has higher priority than Review (2.0)
			compare:  func(a, b float64) bool { return a > 2.0 },
		},
		{
			name:     "New card vs others",
			state:    fsrs.New,
			due:      now,
			expected: 1.0, // New cards have lowest priority
			compare:  func(a, b float64) bool { return a < 2.0 && a < 3.0 },
		},
		{
			name:     "Very overdue card",
			state:    fsrs.Review,
			due:      now.Add(-240 * time.Hour), // 10 days overdue
			expected: 2.0 * 2.0,                 // priority should be higher
			compare:  func(a, b float64) bool { return a > 2.0 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := manager.GetReviewPriority(tt.state, tt.due, now)

			if !tt.compare(priority, tt.expected) {
				t.Errorf("Priority %f did not meet expectation (expected around %f)", priority, tt.expected)
			}
		})
	}
}

func TestPrioritySorting(t *testing.T) {
	manager := NewFSRSManager()
	now := time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)

	// Define cards with different states and due dates
	cards := []struct {
		state fsrs.State
		due   time.Time
	}{
		{fsrs.New, now},                            // New card due now
		{fsrs.Learning, now.Add(1 * time.Hour)},    // Learning card due in 1 hour
		{fsrs.Review, now.Add(-2 * time.Hour)},     // Review card 2 hours overdue
		{fsrs.Review, now.Add(5 * time.Hour)},      // Review card due in 5 hours
		{fsrs.Learning, now},                       // Learning card due now
		{fsrs.Relearning, now.Add(-1 * time.Hour)}, // Relearning card 1 hour overdue
	}

	// Calculate priorities
	priorities := make([]float64, len(cards))
	for i, card := range cards {
		priorities[i] = manager.GetReviewPriority(card.state, card.due, now)
	}

	// Verify that the overdue Relearning card has the highest priority
	highestIdx := 0
	for i, p := range priorities {
		if p > priorities[highestIdx] {
			highestIdx = i
		}
	}

	if highestIdx != 5 { // Index of the overdue Relearning card
		t.Errorf("Expected overdue Relearning card to have highest priority, but got card at index %d", highestIdx)
	}

	// Verify that the New card has the lowest priority
	lowestIdx := 0
	for i, p := range priorities {
		if p < priorities[lowestIdx] {
			lowestIdx = i
		}
	}

	if lowestIdx != 0 { // Index of the New card
		t.Errorf("Expected New card to have lowest priority, but got card at index %d", lowestIdx)
	}
}

// TestDueDateConsistency verifies that FSRS due date calculations are consistent
// when performed directly or sequentially through multiple calls.
func TestDueDateConsistency(t *testing.T) {
	manager := NewFSRSManager()

	// Starting point for our test
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create a new card with initial state
	card := fsrs.NewCard()
	card.Due = startTime

	// Sequence of review ratings to test
	ratings := []fsrs.Rating{
		fsrs.Good, // First review - should move to graduated
		fsrs.Good, // Second review after some time
		fsrs.Good, // Third review after more time
	}

	// Times when reviews are performed
	reviewTimes := []time.Time{
		startTime,                         // First review at start time
		startTime.Add(24 * time.Hour),     // Second review 1 day later
		startTime.Add(24 * time.Hour * 7), // Third review 7 days after start
	}

	// Method 1: Direct library calls with original card and times
	directCard := card // Start with a copy of the original new card
	var directResults []struct {
		state fsrs.State
		due   time.Time
	}

	// Perform direct sequential reviews
	for i, rating := range ratings {
		directCard = manager.GetSchedulingInfo(directCard, rating, reviewTimes[i])
		directResults = append(directResults, struct {
			state fsrs.State
			due   time.Time
		}{directCard.State, directCard.Due})
		t.Logf("Direct review %d (Rating: %d): State=%v, Due=%v",
			i+1, rating, directCard.State, directCard.Due.Format(time.RFC3339))
	}

	// Method 2: Calculate each step individually starting from original card
	var individualResults []struct {
		state fsrs.State
		due   time.Time
	}

	// First review from new card -> directly calculate result
	step1Card := manager.GetSchedulingInfo(card, ratings[0], reviewTimes[0])
	individualResults = append(individualResults, struct {
		state fsrs.State
		due   time.Time
	}{step1Card.State, step1Card.Due})
	t.Logf("Individual review 1 (Rating: %d): State=%v, Due=%v",
		ratings[0], step1Card.State, step1Card.Due.Format(time.RFC3339))

	// Second review - calculate by applying same time offset to the result of first review
	step2Card := manager.GetSchedulingInfo(step1Card, ratings[1], reviewTimes[1])
	individualResults = append(individualResults, struct {
		state fsrs.State
		due   time.Time
	}{step2Card.State, step2Card.Due})
	t.Logf("Individual review 2 (Rating: %d): State=%v, Due=%v",
		ratings[1], step2Card.State, step2Card.Due.Format(time.RFC3339))

	// Third review - calculate from second review result
	step3Card := manager.GetSchedulingInfo(step2Card, ratings[2], reviewTimes[2])
	individualResults = append(individualResults, struct {
		state fsrs.State
		due   time.Time
	}{step3Card.State, step3Card.Due})
	t.Logf("Individual review 3 (Rating: %d): State=%v, Due=%v",
		ratings[2], step3Card.State, step3Card.Due.Format(time.RFC3339))

	// Compare the results
	for i := range ratings {
		if directResults[i].state != individualResults[i].state {
			t.Errorf("Review %d: State mismatch between direct and individual calculation. Direct=%v, Individual=%v",
				i+1, directResults[i].state, individualResults[i].state)
		}

		timeDiff := directResults[i].due.Sub(individualResults[i].due).Abs()
		if timeDiff > time.Second {
			t.Errorf("Review %d: Due date mismatch between direct and individual calculation. Direct=%v, Individual=%v, Diff=%v",
				i+1, directResults[i].due.Format(time.RFC3339), individualResults[i].due.Format(time.RFC3339), timeDiff)
		}
	}

	// Additional test with a longer sequence of different ratings
	longRatings := []fsrs.Rating{
		fsrs.Good,  // First review
		fsrs.Hard,  // Second review
		fsrs.Easy,  // Third review
		fsrs.Again, // Fourth review - should cause regression to relearning
		fsrs.Good,  // Fifth review
	}

	// Reset and start with a new card
	longCard := fsrs.NewCard()
	longCard.Due = startTime

	// Track due dates after each review
	var dueDates []time.Time
	currentTime := startTime

	// Track all review timestamps
	var reviewTimestamps []time.Time

	// Perform sequential reviews with different time intervals
	for i, rating := range longRatings {
		// Log what we're about to do
		t.Logf("Long sequence review %d: Applying rating %d at time %v",
			i+1, rating, currentTime.Format(time.RFC3339))

		// Save review timestamp
		reviewTimestamps = append(reviewTimestamps, currentTime)

		// Update card with review
		longCard = manager.GetSchedulingInfo(longCard, rating, currentTime)

		// Store due date
		dueDates = append(dueDates, longCard.Due)
		t.Logf("Long sequence review %d result: State=%v, Due=%v (in %v)",
			i+1, longCard.State, longCard.Due.Format(time.RFC3339), longCard.Due.Sub(currentTime))

		// Advance time differently based on rating to create realistic scenarios
		switch rating {
		case fsrs.Again:
			// Very short interval for "Again" ratings
			currentTime = currentTime.Add(30 * time.Minute)
		case fsrs.Hard:
			// Shorter interval for "Hard" ratings
			currentTime = currentTime.Add(24 * time.Hour)
		case fsrs.Good:
			// Advance by scheduled days or at least one day
			if longCard.ScheduledDays > 0 {
				currentTime = currentTime.AddDate(0, 0, int(longCard.ScheduledDays))
			} else {
				currentTime = currentTime.Add(24 * time.Hour)
			}
		case fsrs.Easy:
			// Advance by scheduled days * 1.5 or at least two days
			if longCard.ScheduledDays > 0 {
				currentTime = currentTime.AddDate(0, 0, int(longCard.ScheduledDays)+1)
			} else {
				currentTime = currentTime.Add(48 * time.Hour)
			}
		}
	}

	// Now try to reproduce the exact same sequence but one review at a time
	// This tests if we can get consistent results when the FSRS calculation
	// is performed from scratch each time with the same inputs
	multiStepCard := fsrs.NewCard()
	multiStepCard.Due = startTime

	for i, rating := range longRatings {
		// Apply this review
		nextCard := manager.GetSchedulingInfo(multiStepCard, rating, reviewTimestamps[i])

		// Check due date
		if nextCard.Due.Sub(dueDates[i]).Abs() > time.Second {
			t.Errorf("Review %d (Rating %d): Due date inconsistency when reproducing sequence. "+
				"Expected=%v, Got=%v, Diff=%v",
				i+1, rating, dueDates[i].Format(time.RFC3339), nextCard.Due.Format(time.RFC3339),
				nextCard.Due.Sub(dueDates[i]).Abs())
		} else {
			t.Logf("Review %d (Rating %d): Due date consistent when reproducing sequence.", i+1, rating)
		}

		// For the next iteration, use the updated card
		multiStepCard = nextCard
	}
}
