// Package usage provides usage statistics tracking for tokrouter.
//
// The package records token usage (input/output) and costs per provider/model.
// Statistics are stored in a SQLite database and can be queried with time-based
// filters and grouping options.
//
// Basic usage:
//
//	storage, _ := usage.NewStorage("stats.db")
//	svc := usage.NewService(storage)
//
//	// Record usage (prices are per-million-token)
//	svc.RecordWithEndpoint(usage, endpoint, false, 0.01, 0.03)
//
//	// Query stats
//	stats, _ := svc.Query(usage.QueryFilter{
//	    Start:   start,
//	    End:     end,
//	    GroupBy: usage.GroupByProvider,
//	})
package usage
