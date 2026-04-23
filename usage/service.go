package usage

import (
	"time"

	"github.com/tokzone/fluxcore/message"
	"github.com/tokzone/fluxcore/routing"
)

// Service records and queries usage statistics
type Service struct {
	storage Storage
}

// NewService creates a usage service
func NewService(storage Storage) *Service {
	return &Service{
		storage: storage,
	}
}

// RecordWithEndpoint records usage with endpoint info
// inputPrice and outputPrice are per-million-token prices for cost calculation.
func (s *Service) RecordWithEndpoint(usage *message.Usage, ep *routing.Endpoint, isStream bool, inputPrice, outputPrice float64) {
	if s.storage == nil || usage == nil || ep == nil {
		return
	}

	record := &Record{
		Timestamp:    time.Now(),
		Provider:     ep.Key.BaseURL, // Use BaseURL as provider identifier
		Model:        ep.Model,
		InputTokens:  int64(usage.InputTokens),
		OutputTokens: int64(usage.OutputTokens),
		Success:      usage.InputTokens > 0 || usage.OutputTokens > 0,
		Stream:       isStream,
		LatencyMs:    uint16(usage.LatencyMs),
	}

	// Calculate cost from pricing (store as cents * 10000 for precision)
	// Price is per million tokens
	cost := float64(usage.InputTokens)*inputPrice + float64(usage.OutputTokens)*outputPrice
	record.Cost = int64(cost * 10000)

	s.storage.Record(record)
}

// Query queries usage statistics
func (s *Service) Query(filter QueryFilter) ([]StatRow, error) {
	if s.storage == nil {
		return nil, ErrDisabled
	}
	return s.storage.Query(filter)
}

// Close closes the storage
func (s *Service) Close() error {
	if s.storage == nil {
		return nil
	}
	return s.storage.Close()
}
