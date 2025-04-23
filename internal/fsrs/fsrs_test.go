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
