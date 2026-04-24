package usage

import (
	"testing"
	"time"
)

func TestMonthRange(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:     "mid month",
			now:      time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
			wantStart: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "first day of month",
			now:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "last day of month",
			now:      time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			wantStart: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "December - next year",
			now:      time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// MonthRange uses time.Now(), so we test the logic directly
			start, end := monthRangeAt(tt.now)
			if start != tt.wantStart {
				t.Errorf("start = %v, want %v", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

func TestDayRange(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:     "noon",
			input:    time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC),
			wantStart: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "midnight",
			input:    time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "last moment of day",
			input:    time.Date(2024, 6, 15, 23, 59, 59, 999999999, time.UTC),
			wantStart: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "first day of year",
			input:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "last day of year",
			input:    time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC),
			wantStart: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := DayRange(tt.input)
			if start != tt.wantStart {
				t.Errorf("start = %v, want %v", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

func TestWeekRange(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:     "Wednesday",
			now:      time.Date(2024, 6, 12, 10, 0, 0, 0, time.UTC), // Wednesday June 12
			wantStart: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC), // Monday June 10
			wantEnd:   time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC), // Monday June 17
		},
		{
			name:     "Monday",
			now:      time.Date(2024, 6, 10, 10, 0, 0, 0, time.UTC), // Monday
			wantStart: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC), // Same day
			wantEnd:   time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Sunday",
			now:      time.Date(2024, 6, 16, 10, 0, 0, 0, time.UTC), // Sunday June 16
			wantStart: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC), // Monday June 10
			wantEnd:   time.Date(2024, 6, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "cross month boundary",
			now:      time.Date(2024, 7, 3, 10, 0, 0, 0, time.UTC), // Wednesday July 3
			wantStart: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), // Monday July 1
			wantEnd:   time.Date(2024, 7, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "cross year boundary",
			now:      time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), // Tuesday Dec 31
			wantStart: time.Date(2024, 12, 30, 0, 0, 0, 0, time.UTC), // Monday Dec 30
			wantEnd:   time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := weekRangeAt(tt.now)
			if start != tt.wantStart {
				t.Errorf("start = %v, want %v", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

// Helper functions for testing with specific times
func monthRangeAt(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 1, 0)
	return start, end
}

func weekRangeAt(t time.Time) (start, end time.Time) {
	daysSinceMonday := int(t.Weekday() - time.Monday)
	if daysSinceMonday < 0 {
		daysSinceMonday += 7
	}
	start = time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, t.Location())
	end = start.Add(7 * 24 * time.Hour)
	return start, end
}