package usage

import "time"

// Record represents a single usage record (Entity - immutable event)
type Record struct {
	ID           uint      `gorm:"primaryKey"`
	Timestamp    time.Time `gorm:"index:idx_query,priority:1"`          // Composite index for query
	Provider     string    `gorm:"index:idx_query,priority:2;size:256"` // BaseURL of the endpoint
	Model        string    `gorm:"index:idx_query,priority:3;size:64"`
	InputTokens  int64
	OutputTokens int64
	Success      bool
	Stream       bool
	LatencyMS    uint16 // Max 65535ms is sufficient
}

// TableName returns the GORM table name
func (Record) TableName() string {
	return "usage_records"
}
