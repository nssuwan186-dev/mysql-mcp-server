package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/askdba/mysql-mcp-server/internal/api"
)

// ===== TokenMetrics unit tests =====

func TestTokenMetricsEmpty(t *testing.T) {
	m := newTokenMetrics(5)
	snap := m.Snapshot()

	if snap.TotalInputTokens != 0 {
		t.Errorf("expected TotalInputTokens=0, got %d", snap.TotalInputTokens)
	}
	if snap.TotalOutputTokens != 0 {
		t.Errorf("expected TotalOutputTokens=0, got %d", snap.TotalOutputTokens)
	}
	if snap.TotalTokens != 0 {
		t.Errorf("expected TotalTokens=0, got %d", snap.TotalTokens)
	}
	if snap.TotalCostUSD != 0 {
		t.Errorf("expected TotalCostUSD=0, got %f", snap.TotalCostUSD)
	}
	if snap.QueryCount != 0 {
		t.Errorf("expected QueryCount=0, got %d", snap.QueryCount)
	}
	if len(snap.RecentQueries) != 0 {
		t.Errorf("expected no recent queries, got %d", len(snap.RecentQueries))
	}
}

func TestTokenMetricsRecord(t *testing.T) {
	m := newTokenMetrics(5)

	m.Record("run_query", 100, 200)

	snap := m.Snapshot()
	if snap.TotalInputTokens != 100 {
		t.Errorf("expected TotalInputTokens=100, got %d", snap.TotalInputTokens)
	}
	if snap.TotalOutputTokens != 200 {
		t.Errorf("expected TotalOutputTokens=200, got %d", snap.TotalOutputTokens)
	}
	if snap.TotalTokens != 300 {
		t.Errorf("expected TotalTokens=300, got %d", snap.TotalTokens)
	}
	if snap.QueryCount != 1 {
		t.Errorf("expected QueryCount=1, got %d", snap.QueryCount)
	}
	if len(snap.RecentQueries) != 1 {
		t.Errorf("expected 1 recent query, got %d", len(snap.RecentQueries))
	}
	if snap.RecentQueries[0].Tool != "run_query" {
		t.Errorf("expected tool=run_query, got %s", snap.RecentQueries[0].Tool)
	}
}

func TestTokenMetricsCumulativeAccumulation(t *testing.T) {
	m := newTokenMetrics(5)

	m.Record("list_databases", 10, 50)
	m.Record("run_query", 100, 200)

	snap := m.Snapshot()
	if snap.TotalInputTokens != 110 {
		t.Errorf("expected TotalInputTokens=110, got %d", snap.TotalInputTokens)
	}
	if snap.TotalOutputTokens != 250 {
		t.Errorf("expected TotalOutputTokens=250, got %d", snap.TotalOutputTokens)
	}
	if snap.TotalTokens != 360 {
		t.Errorf("expected TotalTokens=360, got %d", snap.TotalTokens)
	}
	if snap.QueryCount != 2 {
		t.Errorf("expected QueryCount=2, got %d", snap.QueryCount)
	}
}

func TestTokenMetricsRingBuffer(t *testing.T) {
	m := newTokenMetrics(3) // max 3 recent

	m.Record("t1", 1, 1)
	m.Record("t2", 2, 2)
	m.Record("t3", 3, 3)
	m.Record("t4", 4, 4) // should evict t1

	snap := m.Snapshot()
	if len(snap.RecentQueries) != 3 {
		t.Fatalf("expected 3 recent queries, got %d", len(snap.RecentQueries))
	}
	if snap.RecentQueries[0].Tool != "t2" {
		t.Errorf("expected oldest tool=t2, got %s", snap.RecentQueries[0].Tool)
	}
	if snap.RecentQueries[2].Tool != "t4" {
		t.Errorf("expected newest tool=t4, got %s", snap.RecentQueries[2].Tool)
	}
}

func TestTokenMetricsIOEfficiency(t *testing.T) {
	m := newTokenMetrics(5)
	m.Record("run_query", 100, 400)

	snap := m.Snapshot()
	// 400/100 = 4.0
	if snap.IOEfficiency != 4.0 {
		t.Errorf("expected IOEfficiency=4.0, got %f", snap.IOEfficiency)
	}
}

func TestTokenMetricsIOEfficiencyZeroInput(t *testing.T) {
	m := newTokenMetrics(5)
	m.Record("run_query", 0, 0)

	snap := m.Snapshot()
	if snap.IOEfficiency != 0.0 {
		t.Errorf("expected IOEfficiency=0.0 for zero input, got %f", snap.IOEfficiency)
	}
}

func TestTokenMetricsCostUSD(t *testing.T) {
	m := newTokenMetrics(5)
	// 1M input tokens costs $2.50, 1M output costs $10.00
	// 1000 input = $0.0025, 1000 output = $0.01
	m.Record("run_query", 1_000_000, 1_000_000)

	snap := m.Snapshot()
	expectedCost := 2.50 + 10.00 // $12.50 for 1M+1M
	if snap.TotalCostUSD != expectedCost {
		t.Errorf("expected TotalCostUSD=%f, got %f", expectedCost, snap.TotalCostUSD)
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		input    float64
		decimals int
		want     float64
	}{
		{1.23456, 2, 1.23},
		{1.235, 2, 1.24},
		{0.0, 4, 0.0},
		{-1.235, 2, -1.24},
	}
	for _, tt := range tests {
		got := roundFloat(tt.input, tt.decimals)
		if got != tt.want {
			t.Errorf("roundFloat(%f, %d) = %f, want %f", tt.input, tt.decimals, got, tt.want)
		}
	}
}

// ===== HTTP endpoint tests =====

func TestHTTPMetricsTokens(t *testing.T) {
	// Reset the global metrics for a clean snapshot
	oldMetrics := globalTokenMetrics
	globalTokenMetrics = newTokenMetrics(5)
	defer func() { globalTokenMetrics = oldMetrics }()

	// Seed some data
	globalTokenMetrics.Record("run_query", 50, 150)

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/tokens", nil)
	w := httptest.NewRecorder()

	httpMetricsTokens(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result api.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if v, _ := data["total_input_tokens"].(float64); v != 50 {
		t.Errorf("expected total_input_tokens=50, got %v", v)
	}
	if v, _ := data["total_output_tokens"].(float64); v != 150 {
		t.Errorf("expected total_output_tokens=150, got %v", v)
	}
	if v, _ := data["total_tokens"].(float64); v != 200 {
		t.Errorf("expected total_tokens=200, got %v", v)
	}
	if v, _ := data["query_count"].(float64); v != 1 {
		t.Errorf("expected query_count=1, got %v", v)
	}
}

func TestHTTPMetricsTokensEmpty(t *testing.T) {
	oldMetrics := globalTokenMetrics
	globalTokenMetrics = newTokenMetrics(5)
	defer func() { globalTokenMetrics = oldMetrics }()

	req := httptest.NewRequest(http.MethodGet, "/api/metrics/tokens", nil)
	w := httptest.NewRecorder()

	httpMetricsTokens(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result api.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestHTTPStatusPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	httpStatusPage(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type=text/html; charset=utf-8, got %s", ct)
	}
}
