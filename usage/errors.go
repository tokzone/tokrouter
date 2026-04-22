package usage

import "errors"

// Errors for usage module
var (
	ErrDisabled       = errors.New("usage tracking is disabled")
	ErrInvalidFilter  = errors.New("invalid query filter")
	ErrRecordNotFound = errors.New("usage record not found")
)
