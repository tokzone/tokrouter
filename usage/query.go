package usage

import "time"

// GroupBy type for query filter
type GroupBy string

const (
	GroupByProvider GroupBy = "provider" // Renamed from "key" for semantic clarity
	GroupByModel    GroupBy = "model"
	GroupByDay      GroupBy = "day"
)

// QueryFilter defines query filter parameters
type QueryFilter struct {
	Start    time.Time
	End      time.Time
	Provider string // Provider filter (BaseURL)
	Model    string
	GroupBy  GroupBy
}

// StatRow represents aggregated statistics row (DTO)
type StatRow struct {
	GroupKey     string // group by value
	InputTokens  int64
	OutputTokens int64
	Cost         int64 // Unified to int64 (cents * 10000)
	RequestCount int64
	AvgLatency   int64
	SuccessRate  float64
}

// TotalTokens returns total tokens
func (s StatRow) TotalTokens() int64 {
	return s.InputTokens + s.OutputTokens
}

// MonthRange returns start and end of current month
func MonthRange() (start, end time.Time) {
	now := time.Now()
	start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end = start.AddDate(0, 1, 0)
	return start, end
}

// DayRange returns start and end of a day
func DayRange(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	end = start.Add(24 * time.Hour)
	return start, end
}

// WeekRange returns start and end of current week
func WeekRange() (start, end time.Time) {
	now := time.Now()
	daysSinceMonday := int(now.Weekday() - time.Monday)
	if daysSinceMonday < 0 {
		daysSinceMonday += 7
	}
	start = time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
	end = start.Add(7 * 24 * time.Hour)
	return start, end
}
