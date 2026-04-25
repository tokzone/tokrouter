package usage

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tokzone/fluxcore/message"
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

// enqueue sends a record to the buffer (non-blocking)
func (s *Service) enqueue(record *Record) bool {
	select {
	case s.recordCh <- record:
		return true
	default:
		count := s.droppedCount.Add(1)
		if count%100 == 0 {
			s.logger.Warn("usage records dropped", "count", count)
		}
		return false
	}
}

// Record records usage (non-blocking)
func (s *Service) Record(usage *message.Usage, isStream bool) bool {
	if s.store == nil || usage == nil {
		return false
	}
	return s.enqueue(&Record{
		Timestamp:    time.Now(),
		InputTokens:  int64(usage.InputTokens),
		OutputTokens: int64(usage.OutputTokens),
		Success:      usage.InputTokens > 0 || usage.OutputTokens > 0,
		Stream:       isStream,
		LatencyMS:    uint16(usage.LatencyMs),
	})
}

// RecordWithModel records usage with model info (non-blocking)
func (s *Service) RecordWithModel(usage *message.Usage, model string, isStream bool) bool {
	if s.store == nil || usage == nil {
		return false
	}
	return s.enqueue(&Record{
		Timestamp:    time.Now(),
		Model:        model,
		InputTokens:  int64(usage.InputTokens),
		OutputTokens: int64(usage.OutputTokens),
		Success:      usage.InputTokens > 0 || usage.OutputTokens > 0,
		Stream:       isStream,
		LatencyMS:    uint16(usage.LatencyMs),
	})
}

// RecordWithModelAndProvider records usage with model and provider info (non-blocking)
func (s *Service) RecordWithModelAndProvider(usage *message.Usage, model string, provider string, isStream bool) bool {
	if s.store == nil || usage == nil {
		return false
	}
	return s.enqueue(&Record{
		Timestamp:    time.Now(),
		Provider:     provider,
		Model:        model,
		InputTokens:  int64(usage.InputTokens),
		OutputTokens: int64(usage.OutputTokens),
		Success:      usage.InputTokens > 0 || usage.OutputTokens > 0,
		Stream:       isStream,
		LatencyMS:    uint16(usage.LatencyMs),
	})
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
