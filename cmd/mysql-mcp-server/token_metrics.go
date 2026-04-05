// cmd/mysql-mcp-server/token_metrics.go
package main

import (
	"sync"
	"time"
)

// QueryTokenRecord stores token usage information for a single tool/query call.
type QueryTokenRecord struct {
	Tool         string    `json:"tool"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	Timestamp    time.Time `json:"timestamp"`
}

// TokenMetrics holds the cumulative token usage since server startup.
type TokenMetrics struct {
	mu sync.Mutex

	// Cumulative totals since startup
	TotalInputTokens  int
	TotalOutputTokens int
	TotalTokens       int
	TotalCostUSD      float64
	QueryCount        int

	// Rolling window: last N query records
	recentQueries []QueryTokenRecord
	maxRecent     int

	// Server start time (for "uptime" context)
	StartTime time.Time
}

// globalTokenMetrics is the process-wide singleton aggregator.
var globalTokenMetrics = newTokenMetrics(5)

func newTokenMetrics(maxRecent int) *TokenMetrics {
	return &TokenMetrics{
		maxRecent: maxRecent,
		StartTime: time.Now(),
	}
}

// Record adds a new query token record to the aggregator.
// It is safe to call concurrently from multiple goroutines.
func (m *TokenMetrics) Record(tool string, inputTokens, outputTokens int) {
	cost := calculateCostUSD(inputTokens, outputTokens)
	rec := QueryTokenRecord{
		Tool:         tool,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		CostUSD:      cost,
		Timestamp:    time.Now(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalInputTokens += inputTokens
	m.TotalOutputTokens += outputTokens
	m.TotalTokens += inputTokens + outputTokens
	m.TotalCostUSD += cost
	m.QueryCount++

	// Append to ring buffer (keep only the most recent maxRecent entries)
	m.recentQueries = append(m.recentQueries, rec)
	if len(m.recentQueries) > m.maxRecent {
		m.recentQueries = m.recentQueries[len(m.recentQueries)-m.maxRecent:]
	}
}

// Snapshot returns a point-in-time copy of the current metrics.
func (m *TokenMetrics) Snapshot() TokenMetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	recent := make([]QueryTokenRecord, len(m.recentQueries))
	copy(recent, m.recentQueries)

	ioEff := 0.0
	if m.TotalInputTokens > 0 {
		ioEff = roundFloat(float64(m.TotalOutputTokens)/float64(m.TotalInputTokens), 4)
	}

	return TokenMetricsSnapshot{
		TotalInputTokens:  m.TotalInputTokens,
		TotalOutputTokens: m.TotalOutputTokens,
		TotalTokens:       m.TotalTokens,
		TotalCostUSD:      roundFloat(m.TotalCostUSD, 6),
		QueryCount:        m.QueryCount,
		IOEfficiency:      ioEff,
		UptimeSeconds:     int(time.Since(m.StartTime).Seconds()),
		RecentQueries:     recent,
		TokenTrackingOn:   tokenTracking,
	}
}

// TokenMetricsSnapshot is the immutable view returned by the API endpoint.
type TokenMetricsSnapshot struct {
	TotalInputTokens  int                `json:"total_input_tokens"`
	TotalOutputTokens int                `json:"total_output_tokens"`
	TotalTokens       int                `json:"total_tokens"`
	TotalCostUSD      float64            `json:"total_cost_usd"`
	QueryCount        int                `json:"query_count"`
	IOEfficiency      float64            `json:"io_efficiency"`
	UptimeSeconds     int                `json:"uptime_seconds"`
	RecentQueries     []QueryTokenRecord `json:"recent_queries"`
	TokenTrackingOn   bool               `json:"token_tracking_on"`
}

// calculateCostUSD computes the USD cost for a given input/output token pair using
// the same pricing constants defined in token_estimator.go.
func calculateCostUSD(inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) / 1_000_000 * costPerMillionInputTokens
	outputCost := float64(outputTokens) / 1_000_000 * costPerMillionOutputTokens
	return inputCost + outputCost
}

// roundFloat rounds f to the given number of decimal places.
func roundFloat(f float64, decimals int) float64 {
	p := 1.0
	for i := 0; i < decimals; i++ {
		p *= 10
	}
	// Use integer arithmetic to avoid importing math just for Round.
	// This is equivalent to math.Round(f*p)/p.
	if f >= 0 {
		return float64(int64(f*p+0.5)) / p
	}
	return float64(int64(f*p-0.5)) / p
}
