// Package usage provides usage statistics tracking for tokrouter.
//
// The package records token usage (input/output) and costs per provider/model.
// Statistics are stored in a SQLite database and can be queried with time-based
// filters and grouping options.
//
// Basic usage:
//
//	store, _ := usage.NewStore("stats.db")
//	svc := usage.NewService(store)
//
//	// Record usage
//	svc.RecordWithEndpoint(usage, endpoint, false)
//
//	// Query stats
//	stats, _ := svc.Query(usage.QueryFilter{
//	    Start:   start,
//	    End:     end,
//	    GroupBy: usage.GroupByProvider,
//	})
package usage
