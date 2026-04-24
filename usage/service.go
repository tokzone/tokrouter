package usage

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/routing"
)

const recordBufferSize = 1000

// Service records and queries usage statistics
type Service struct {
	store        Store
	recordCh     chan *Record
	wg           sync.WaitGroup
	droppedCount atomic.Int64 // Count of dropped records
	logger       *slog.Logger
}

// NewService creates a usage service
func NewService(store Store) *Service {
	svc := &Service{
		store:    store,
		recordCh: make(chan *Record, recordBufferSize),
		logger:   slog.Default().With("component", "usage"),
	}

	// Start background writer
	svc.wg.Add(1)
	go svc.recordWorker()

	return svc
}

// recordWorker processes records in background
func (s *Service) recordWorker() {
	defer s.wg.Done()
	for record := range s.recordCh {
		s.store.Record(record)
	}
}

// RecordWithEndpoint records usage with endpoint info (non-blocking)
// Returns true if record was queued, false if dropped due to buffer full
func (s *Service) RecordWithEndpoint(usage *message.Usage, ep *routing.Endpoint, isStream bool) bool {
	if s.store == nil || usage == nil || ep == nil {
		return false
	}

	record := &Record{
		Timestamp:    time.Now(),
		Provider:     ep.Key.BaseURL, // Use BaseURL as provider identifier
		Model:        ep.Model,
		InputTokens:  int64(usage.InputTokens),
		OutputTokens: int64(usage.OutputTokens),
		Success:      usage.InputTokens > 0 || usage.OutputTokens > 0,
		Stream:       isStream,
		LatencyMS:    uint16(usage.LatencyMs),
	}

	// Non-blocking send, drop if buffer full
	select {
	case s.recordCh <- record:
		return true
	default:
		count := s.droppedCount.Add(1)
		if count%100 == 0 { // Log every 100 drops
			s.logger.Warn("usage records dropped", "count", count)
		}
		return false
	}
}

// Query queries usage statistics
func (s *Service) Query(filter QueryFilter) ([]StatRow, error) {
	if s.store == nil {
		return nil, ErrDisabled
	}
	return s.store.Query(filter)
}

// Close closes the store and waits for pending records
func (s *Service) Close() error {
	if s.store == nil {
		return nil
	}

	// Close channel and wait for worker to finish
	close(s.recordCh)
	s.wg.Wait()

	return s.store.Close()
}
