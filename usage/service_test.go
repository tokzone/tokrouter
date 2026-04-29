package usage

import (
	"sync"
	"testing"
	"time"

	"github.com/tokzone/fluxcore/message"
)

// mockStore implements Store for testing
type mockStore struct {
	mu       sync.Mutex
	records  []*Record
	recorded chan struct{} // Signal when a record is written
}

func newMockStore() *mockStore {
	return &mockStore{
		recorded: make(chan struct{}, 100),
	}
}

func (m *mockStore) Record(record *Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, record)
	// Signal that record was written (non-blocking)
	select {
	case m.recorded <- struct{}{}:
	default:
	}
	return nil
}

func (m *mockStore) Query(filter QueryFilter) ([]StatRow, error) {
	return nil, nil
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) getRecordCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

// waitForRecord waits for a record to be written with timeout
func (m *mockStore) waitForRecord(timeout time.Duration) bool {
	select {
	case <-m.recorded:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestNewService(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.store == nil {
		t.Error("store not set")
	}
	if svc.recordCh == nil {
		t.Error("recordCh not initialized")
	}
	svc.Close()
}

func TestRecord(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)
	defer svc.Close()

	usage := &message.Usage{
		InputTokens:  100,
		OutputTokens: 50,
		LatencyMs:    200,
		IsAccurate:   true,
	}

	if !svc.RecordWithModelAndProvider(usage, "gpt-4", "", false) {
		t.Error("expected record to be queued")
	}

	// Wait for background writer with timeout
	if !store.waitForRecord(100 * time.Millisecond) {
		t.Error("timeout waiting for record to be written")
	}

	if count := store.getRecordCount(); count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestRecordWithModel(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)
	defer svc.Close()

	usage := &message.Usage{
		InputTokens:  100,
		OutputTokens: 50,
		LatencyMs:    200,
		IsAccurate:   true,
	}

	if !svc.RecordWithModelAndProvider(usage, "gpt-4", "openai", false) {
		t.Error("expected record to be queued")
	}

	// Wait for background writer with timeout
	if !store.waitForRecord(100 * time.Millisecond) {
		t.Error("timeout waiting for record to be written")
	}

	if count := store.getRecordCount(); count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestRecordWithModelAndProvider(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)
	defer svc.Close()

	usage := &message.Usage{
		InputTokens:  100,
		OutputTokens: 50,
		LatencyMs:    200,
		IsAccurate:   true,
	}

	if !svc.RecordWithModelAndProvider(usage, "gpt-4", "https://api.openai.com", false) {
		t.Error("expected record to be queued")
	}

	// Wait for background writer with timeout
	if !store.waitForRecord(100 * time.Millisecond) {
		t.Error("timeout waiting for record to be written")
	}

	if count := store.getRecordCount(); count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

func TestRecordNilInputs(t *testing.T) {
	store := newMockStore()
	svc := NewService(store)
	defer svc.Close()

	// nil usage
	svc.RecordWithModelAndProvider(nil, "", "", false)
	svc.RecordWithModelAndProvider(nil, "gpt-4", "", false)
	svc.RecordWithModelAndProvider(nil, "gpt-4", "https://api.openai.com", false)

	// Give a brief moment for any potential async processing
	// No records should be written, so we just check count after brief delay
	time.Sleep(5 * time.Millisecond)

	if count := store.getRecordCount(); count != 0 {
		t.Errorf("expected 0 records, got %d", count)
	}
}

// blockingStore implements Store for testing buffer overflow
type blockingStore struct {
	mu      sync.Mutex
	records []*Record
	block   chan struct{}
}

func newBlockingStore() *blockingStore {
	return &blockingStore{
		block: make(chan struct{}),
	}
}

func (m *blockingStore) unblock() {
	close(m.block)
}

func (m *blockingStore) Record(record *Record) error {
	// Block until unblocked with timeout to prevent goroutine leak
	select {
	case <-m.block:
	case <-time.After(5 * time.Second):
		return nil // timeout, just return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, record)
	return nil
}

func (m *blockingStore) Query(filter QueryFilter) ([]StatRow, error) {
	return nil, nil
}

func (m *blockingStore) Close() error {
	return nil
}

func (m *blockingStore) getRecordCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

func TestRecordBufferFull(t *testing.T) {
	store := newBlockingStore()
	svc := NewService(store)
	defer svc.Close()

	usage := &message.Usage{
		InputTokens:  100,
		OutputTokens: 50,
	}

	// Fill buffer beyond capacity (worker is blocked, so buffer will fill)
	dropped := 0
	for i := 0; i < recordBufferSize+100; i++ {
		if !svc.RecordWithModelAndProvider(usage, "", "", false) {
			dropped++
		}
	}

	// Should have some dropped records
	if dropped == 0 {
		t.Error("expected some records to be dropped")
	}

	// Unblock the worker so it can drain and Close() can complete
	store.unblock()
}

func TestQueryDisabled(t *testing.T) {
	svc := NewService(nil)
	defer svc.Close()

	_, err := svc.Query(QueryFilter{})
	if err != ErrDisabled {
		t.Errorf("expected ErrDisabled, got %v", err)
	}
}

func TestCloseNilStore(t *testing.T) {
	svc := NewService(nil)
	if err := svc.Close(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}