package historymetrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeUsageJSONForTest(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write usage file: %v", err)
	}
	return path
}

func TestQueryStatsByDay(t *testing.T) {
	body := `{
		"schema_version": 3,
		"totals": {
			"provider_calls": 4,
			"input_tokens": 400,
			"output_tokens": 200,
			"cache_read_tokens": 100,
			"cache_write_tokens": 50,
			"total_tokens": 750
		},
		"daily_by_model": [
			{"date":"2026-08-19","model":"claude-opus","provider_calls":2,"input_tokens":200,"output_tokens":100,"cache_read_tokens":50,"cache_write_tokens":25,"total_tokens":375},
			{"date":"2026-08-20","model":"claude-opus","provider_calls":1,"input_tokens":100,"output_tokens":50,"cache_read_tokens":25,"cache_write_tokens":12,"total_tokens":187},
			{"date":"2026-08-20","model":"deepseek-v4","provider_calls":1,"input_tokens":100,"output_tokens":50,"cache_read_tokens":25,"cache_write_tokens":13,"total_tokens":188}
		]
	}`
	path := writeUsageJSONForTest(t, body)

	// 无过滤：全部记录。
	result, err := QueryStats(path, "", "", "", nil)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(result.ByDay) != 2 {
		t.Fatalf("byDay len = %d, want 2", len(result.ByDay))
	}
	if result.ByDay[0].Key != "2026-08-20" {
		t.Fatalf("byDay[0] key = %q, want 2026-08-20 (desc sorted)", result.ByDay[0].Key)
	}
	if result.ByDay[0].ProviderCalls != 2 {
		t.Fatalf("byDay[0].providerCalls = %d, want 2", result.ByDay[0].ProviderCalls)
	}
	if len(result.ByModel) != 2 {
		t.Fatalf("byModel len = %d, want 2", len(result.ByModel))
	}
	if result.Total.TotalTokens != 750 {
		t.Fatalf("total tokens = %d, want 750", result.Total.TotalTokens)
	}
}

func TestQueryStatsModelFilter(t *testing.T) {
	body := `{
		"schema_version": 3,
		"totals": {},
		"daily_by_model": [
			{"date":"2026-08-20","model":"claude-opus","provider_calls":1,"input_tokens":100,"output_tokens":50,"cache_read_tokens":25,"cache_write_tokens":12,"total_tokens":187},
			{"date":"2026-08-20","model":"deepseek-v4","provider_calls":1,"input_tokens":200,"output_tokens":100,"cache_read_tokens":50,"cache_write_tokens":25,"total_tokens":375}
		]
	}`
	path := writeUsageJSONForTest(t, body)

	result, err := QueryStats(path, "", "", "", []string{"deepseek-v4"})
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(result.ByModel) != 1 {
		t.Fatalf("byModel len = %d, want 1", len(result.ByModel))
	}
	if result.ByModel[0].Key != "deepseek-v4" {
		t.Fatalf("byModel[0] key = %q, want deepseek-v4", result.ByModel[0].Key)
	}
	if result.Total.ProviderCalls != 1 {
		t.Fatalf("total providerCalls = %d, want 1", result.Total.ProviderCalls)
	}
}

func TestResolveStatsTimeRange(t *testing.T) {
	now := time.Date(2026, 8, 20, 15, 30, 0, 0, time.Local)

	// 今天：本地当天 00:00 到当前时刻。
	today, ok := resolveStatsTimeRange("today", now)
	if !ok {
		t.Fatal("resolve today failed")
	}
	if today.Start.Hour() != 0 || today.Start.Minute() != 0 {
		t.Fatalf("today start = %v, want 00:00", today.Start)
	}
	if !today.End.Equal(now) {
		t.Fatalf("today end = %v, want %v", today.End, now)
	}

	// 昨天：前一天的完整 24 小时。
	yesterday, ok := resolveStatsTimeRange("yesterday", now)
	if !ok {
		t.Fatal("resolve yesterday failed")
	}
	if yesterday.Start.Format("2006-01-02") != "2026-08-19" {
		t.Fatalf("yesterday start = %v, want 2026-08-19", yesterday.Start)
	}
	if yesterday.End.Sub(yesterday.Start) != 24*time.Hour {
		t.Fatalf("yesterday duration = %v, want 24h", yesterday.End.Sub(yesterday.Start))
	}

	// 本月：当月 1 日 00:00。
	thisMonth, ok := resolveStatsTimeRange("thisMonth", now)
	if !ok {
		t.Fatal("resolve thisMonth failed")
	}
	if thisMonth.Start.Day() != 1 {
		t.Fatalf("thisMonth start day = %d, want 1", thisMonth.Start.Day())
	}

	// 未知预设：返回 false。
	if _, ok := resolveStatsTimeRange("bogus", now); ok {
		t.Fatal("resolve bogus should fail")
	}
}

func TestQueryStatsCustomDateRange(t *testing.T) {
	body := `{
		"schema_version": 3,
		"totals": {},
		"daily_by_model": [
			{"date":"2026-08-18","model":"claude-opus","provider_calls":1,"input_tokens":100,"output_tokens":50,"cache_read_tokens":25,"cache_write_tokens":12,"total_tokens":187},
			{"date":"2026-08-19","model":"claude-opus","provider_calls":1,"input_tokens":100,"output_tokens":50,"cache_read_tokens":25,"cache_write_tokens":12,"total_tokens":187},
			{"date":"2026-08-20","model":"claude-opus","provider_calls":1,"input_tokens":100,"output_tokens":50,"cache_read_tokens":25,"cache_write_tokens":12,"total_tokens":187}
		]
	}`
	path := writeUsageJSONForTest(t, body)

	// 自定义 YYYY-MM-DD 闭区间：只包含 08-19 与 08-20。
	result, err := QueryStats(path, "", "2026-08-19", "2026-08-20", nil)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(result.ByDay) != 2 {
		t.Fatalf("byDay len = %d, want 2", len(result.ByDay))
	}
	if result.Total.ProviderCalls != 2 {
		t.Fatalf("total providerCalls = %d, want 2", result.Total.ProviderCalls)
	}

	// 无时间范围：返回全部。
	allResult, err := QueryStats(path, "", "", "", nil)
	if err != nil {
		t.Fatalf("QueryStats all: %v", err)
	}
	if len(allResult.ByDay) != 3 {
		t.Fatalf("all byDay len = %d, want 3", len(allResult.ByDay))
	}
}

func TestParseRFC3339Optional(t *testing.T) {
	if _, err := parseRFC3339Optional(""); err != nil {
		t.Fatalf("empty parse: %v", err)
	}
	dateTime, err := parseRFC3339Optional("2026-08-19")
	if err != nil {
		t.Fatalf("date parse: %v", err)
	}
	if dateTime.Format("2006-01-02") != "2026-08-19" {
		t.Fatalf("date parsed = %v, want 2026-08-19", dateTime)
	}
	rfc3339, err := parseRFC3339Optional("2026-08-19T10:00:00Z")
	if err != nil {
		t.Fatalf("rfc3339 parse: %v", err)
	}
	if rfc3339.UTC().Format("2006-01-02T15:04:05") != "2026-08-19T10:00:00" {
		t.Fatalf("rfc3339 parsed = %v", rfc3339)
	}
	if _, err := parseRFC3339Optional("not-a-date"); err == nil {
		t.Fatal("invalid parse should fail")
	}
}