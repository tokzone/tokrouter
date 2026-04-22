package usage

import "time"

// Record represents a single usage record (Entity - immutable event)
// Optimized: unified int64 types, renamed KeyName→Provider, added composite index
type Record struct {
	ID           uint      `gorm:"primaryKey"`
	Timestamp    time.Time `gorm:"index:idx_query,priority:1"`          // Composite index for query
	Provider     string    `gorm:"index:idx_query,priority:2;size:256"` // Renamed from KeyName (stores BaseURL)
	Model        string    `gorm:"index:idx_query,priority:3;size:64"`
	InputTokens  int64     // Unified to int64 for consistency with StatRow
	OutputTokens int64     // Unified to int64
	Cost         int64     // Changed to int64 (cents * 10000 for precision)
	Success      bool
	Stream       bool
	LatencyMs    uint16 // Max 65535ms is sufficient
}

// TableName returns the GORM table name
func (Record) TableName() string {
	return "usage_records"
}

// TotalTokens returns total tokens (replaces TokenCount.Total())
func (r *Record) TotalTokens() int64 {
	return r.InputTokens + r.OutputTokens
}
