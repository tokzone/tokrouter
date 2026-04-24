package usage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("store is nil")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Clean up
	if err := store.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestStoreRecord(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	record := &Record{
		Timestamp:    time.Now(),
		Provider:     "https://api.openai.com",
		Model:        "gpt-4",
		InputTokens:  100,
		OutputTokens: 50,
		Success:      true,
		Stream:       false,
		LatencyMS:    200,
	}

	if err := store.Record(record); err != nil {
		t.Errorf("Record failed: %v", err)
	}
}

func TestStoreQuery(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert test records
	now := time.Now()
	records := []*Record{
		{
			Timestamp:    now.Add(-2 * time.Hour),
			Provider:     "https://api.openai.com",
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 50,
			Success:      true,
			Stream:       false,
			LatencyMS:    200,
		},
		{
			Timestamp:    now.Add(-1 * time.Hour),
			Provider:     "https://api.anthropic.com",
			Model:        "claude-3",
			InputTokens:  200,
			OutputTokens: 100,
			Success:      true,
			Stream:       true,
			LatencyMS:    300,
		},
		{
			Timestamp:    now,
			Provider:     "https://api.openai.com",
			Model:        "gpt-3.5",
			InputTokens:  50,
			OutputTokens: 25,
			Success:      false,
			Stream:       false,
			LatencyMS:    100,
		},
	}

	for _, r := range records {
		if err := store.Record(r); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	tests := []struct {
		name    string
		filter  QueryFilter
		wantLen int
	}{
		{
			name: "query all by provider",
			filter: QueryFilter{
				GroupBy: GroupByProvider,
			},
			wantLen: 2, // openai + anthropic
		},
		{
			name: "query all by model",
			filter: QueryFilter{
				GroupBy: GroupByModel,
			},
			wantLen: 3, // gpt-4, claude-3, gpt-3.5
		},
		{
			name: "filter by provider",
			filter: QueryFilter{
				Provider: "https://api.openai.com",
				GroupBy:  GroupByModel,
			},
			wantLen: 2, // gpt-4 + gpt-3.5
		},
		{
			name: "filter by model",
			filter: QueryFilter{
				Model:   "gpt-4",
				GroupBy: GroupByProvider,
			},
			wantLen: 1,
		},
		{
			name: "time range filter",
			filter: QueryFilter{
				Start:   now.Add(-90 * time.Minute),
				End:     now.Add(1 * time.Minute),
				GroupBy: GroupByProvider,
			},
			wantLen: 2, // anthropic (1h ago) + openai (now)
		},
		{
			name: "empty result",
			filter: QueryFilter{
				Provider: "https://nonexistent.com",
				GroupBy:  GroupByProvider,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Query(tt.filter)
			if err != nil {
				t.Errorf("Query failed: %v", err)
				return
			}
			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestStoreQueryAggregation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert multiple records for same provider
	now := time.Now()
	for i := 0; i < 5; i++ {
		record := &Record{
			Timestamp:    now.Add(-time.Duration(i) * time.Hour),
			Provider:     "https://api.openai.com",
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 50,
			Success:      true,
			Stream:       false,
			LatencyMS:    uint16(100 + i*10),
		}
		if err := store.Record(record); err != nil {
			t.Fatalf("Record failed: %v", err)
		}
	}

	results, err := store.Query(QueryFilter{GroupBy: GroupByProvider})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	row := results[0]
	if row.GroupKey != "https://api.openai.com" {
		t.Errorf("GroupKey = %s, want https://api.openai.com", row.GroupKey)
	}
	if row.InputTokens != 500 {
		t.Errorf("InputTokens = %d, want 500", row.InputTokens)
	}
	if row.OutputTokens != 250 {
		t.Errorf("OutputTokens = %d, want 250", row.OutputTokens)
	}
	if row.RequestCount != 5 {
		t.Errorf("RequestCount = %d, want 5", row.RequestCount)
	}
}

func TestStoreClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Close should succeed
	if err := store.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Second close should not error (or handle gracefully)
	if err := store.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}