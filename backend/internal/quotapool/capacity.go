package quotapool

// Capacity is supplied by explicitly identified 604800-second weekly windows.
// The old recursive max-percent/fractional-utilization reader was removed:
// five-hour utilization and unrelated per-feature windows cannot enter a weekly ledger.
