package usage

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Store defines the storage interface for usage records
type Store interface {
	Record(record *Record) error
	Query(filter QueryFilter) ([]StatRow, error)
	Close() error
}

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db *gorm.DB
}

// NewStore creates a new SQLite store
func NewStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// Use WAL mode for better concurrency
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&Record{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Record records a usage record
func (s *SQLiteStore) Record(record *Record) error {
	return s.db.Create(record).Error
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Query queries aggregated usage statistics
func (s *SQLiteStore) Query(filter QueryFilter) ([]StatRow, error) {
	db := s.db.Model(&Record{})

	// Apply filters (uses composite index: timestamp, provider, model)
	if !filter.Start.IsZero() {
		db = db.Where("timestamp >= ?", filter.Start)
	}
	if !filter.End.IsZero() {
		db = db.Where("timestamp < ?", filter.End)
	}
	if filter.Provider != "" {
		db = db.Where("provider = ?", filter.Provider)
	}
	if filter.Model != "" {
		db = db.Where("model = ?", filter.Model)
	}

	// Determine group by field
	groupField := "provider"
	switch filter.GroupBy {
	case GroupByModel:
		groupField = "model"
	case GroupByProvider:
		groupField = "provider"
	case GroupByDay:
		groupField = "DATE(timestamp)"
	}

	// Query aggregated stats
	var results []StatRow
	err := db.Select(
		groupField+" as group_key",
		"SUM(input_tokens) as input_tokens",
		"SUM(output_tokens) as output_tokens",
		"COUNT(*) as request_count",
		"AVG(latency_ms) as avg_latency",
		"SUM(CASE WHEN success THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as success_rate",
	).
		Group(groupField).
		Find(&results).Error

	return results, err
}
