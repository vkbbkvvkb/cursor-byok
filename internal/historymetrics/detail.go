package historymetrics

import (
	"encoding/json"
	"os"
	"sort"
	"time"
)

// StatsTimeRange 定义统计时间窗口（使用本地时区解析“今天/昨天/本月”等语义）。
type StatsTimeRange struct {
	Start time.Time
	End   time.Time
}

// StatsTokenBucket 定义一条统计明细（按天或按模型维度）。
type StatsTokenBucket struct {
	Key              string `json:"key"`
	Label            string `json:"label"`
	ProviderCalls    int64  `json:"providerCalls"`
	InputTokens      int64  `json:"inputTokens"`
	OutputTokens     int64  `json:"outputTokens"`
	CacheReadTokens  int64  `json:"cacheReadTokens"`
	CacheWriteTokens int64  `json:"cacheWriteTokens"`
	TotalTokens      int64  `json:"totalTokens"`
}

// StatsResult 定义统计详情的聚合结果。
type StatsResult struct {
	ByDay   []StatsTokenBucket `json:"byDay"`
	ByModel []StatsTokenBucket `json:"byModel"`
	// Total 为窗口内汇总。
	Total StatsTokenBucket `json:"total"`
}

func (bucket StatsTokenBucket) withCountedTotals() StatsTokenBucket {
	bucket.TotalTokens = bucket.InputTokens + bucket.OutputTokens + bucket.CacheReadTokens + bucket.CacheWriteTokens
	return bucket
}

// localDayStart 返回本地时区某天 00:00。
func localDayStart(t time.Time) time.Time {
	year, month, day := t.In(time.Local).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

// resolveStatsTimeRange 解析时间段预设（today/yesterday/last24h/last7d/last14d/last30d/thisMonth/lastMonth）为本地时区闭区间。
func resolveStatsTimeRange(period string, now time.Time) (StatsTimeRange, bool) {
	now = now.In(time.Local)
	todayStart := localDayStart(now)
	switch period {
	case "today":
		return StatsTimeRange{Start: todayStart, End: now}, true
	case "yesterday":
		yesterdayStart := todayStart.AddDate(0, 0, -1)
		return StatsTimeRange{Start: yesterdayStart, End: yesterdayStart.Add(24 * time.Hour)}, true
	case "last24h":
		return StatsTimeRange{Start: now.Add(-24 * time.Hour), End: now}, true
	case "last7d":
		return StatsTimeRange{Start: todayStart.AddDate(0, 0, -6), End: now}, true
	case "last14d":
		return StatsTimeRange{Start: todayStart.AddDate(0, 0, -13), End: now}, true
	case "last30d":
		return StatsTimeRange{Start: todayStart.AddDate(0, 0, -29), End: now}, true
	case "thisMonth":
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		return StatsTimeRange{Start: monthStart, End: now}, true
	case "lastMonth":
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		lastMonthStart := firstOfThisMonth.AddDate(0, -1, 0)
		return StatsTimeRange{Start: lastMonthStart, End: firstOfThisMonth}, true
	default:
		return StatsTimeRange{}, false
	}
}

// DailyUsage 表示从 usage.json 解析出的一条按天+模型聚合记录。
type DailyUsage struct {
	Date            string
	Model           string
	ProviderCalls   int64
	InputTokens     int64
	OutputTokens    int64
	CacheReadTokens int64
	CacheWriteTokens int64
	TotalTokens     int64
}

// LoadDailyUsage 解析 usage.json 的 daily_by_model 聚合。
func LoadDailyUsage(path string) ([]DailyUsage, error) {
	doc, err := readUsageDocument(path)
	if err != nil {
		return nil, err
	}
	items := make([]DailyUsage, 0, len(doc.DailyByModel))
	for _, item := range doc.DailyByModel {
		items = append(items, DailyUsage{
			Date:             item.Date,
			Model:            item.Model,
			ProviderCalls:    item.ProviderCalls,
			InputTokens:      item.InputTokens,
			OutputTokens:     item.OutputTokens,
			CacheReadTokens:  item.CacheReadTokens,
			CacheWriteTokens: item.CacheWriteTokens,
			TotalTokens:      item.TotalTokens,
		})
	}
	return items, nil
}

func readUsageDocument(path string) (usageFileDocument, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return usageFileDocument{}, err
	}
	var doc usageFileDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return usageFileDocument{}, err
	}
	return doc, nil
}

// filterDailyByTime 按本地日期闭区间过滤按天记录。
// startDate/endDate 为 "2006-01-02" 字符串，空值表示不限制。
func filterDailyByTime(items []DailyUsage, startDate string, endDate string) []DailyUsage {
	filtered := make([]DailyUsage, 0, len(items))
	for _, item := range items {
		if startDate != "" && item.Date < startDate {
			continue
		}
		if endDate != "" && item.Date > endDate {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterDailyByModels(items []DailyUsage, models []string) []DailyUsage {
	if len(models) == 0 {
		return items
	}
	allow := make(map[string]struct{}, len(models))
	for _, model := range models {
		if model != "" {
			allow[model] = struct{}{}
		}
	}
	filtered := make([]DailyUsage, 0, len(items))
	for _, item := range items {
		if _, ok := allow[item.Model]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func emptyStatsBucket(key string, label string) StatsTokenBucket {
	return StatsTokenBucket{Key: key, Label: label}
}

func aggregateBuckets(buckets []StatsTokenBucket) StatsTokenBucket {
	var total StatsTokenBucket
	for _, bucket := range buckets {
		total.ProviderCalls += bucket.ProviderCalls
		total.InputTokens += bucket.InputTokens
		total.OutputTokens += bucket.OutputTokens
		total.CacheReadTokens += bucket.CacheReadTokens
		total.CacheWriteTokens += bucket.CacheWriteTokens
		total.TotalTokens += bucket.TotalTokens
	}
	return total
}

// QueryStats 按天+模型聚合 usage.json 记录，并汇总窗口内统计。
// 时间过滤基于本地时区，模型过滤基于白名单。
func QueryStats(path string, period string, startAt string, endAt string, models []string) (StatsResult, error) {
	items, err := LoadDailyUsage(path)
	if err != nil {
		return StatsResult{}, err
	}

	// 时间段预设优先于显式起止时间。
	var timeRange StatsTimeRange
	hasTimeRange := false
	if period != "" {
		if resolved, ok := resolveStatsTimeRange(period, time.Now()); ok {
			timeRange = resolved
			hasTimeRange = true
		}
	}
	if !hasTimeRange {
		if startAt != "" || endAt != "" {
			startTime, startErr := parseRFC3339Optional(startAt)
			endTime, endErr := parseRFC3339Optional(endAt)
			if startErr != nil {
				return StatsResult{}, startErr
			}
			if endErr != nil {
				return StatsResult{}, endErr
			}
			timeRange = StatsTimeRange{Start: startTime, End: endTime}
			hasTimeRange = true
		}
	}

	startDate := ""
	endDate := ""
	if hasTimeRange {
		if !timeRange.Start.IsZero() {
			startDate = timeRange.Start.In(time.Local).Format("2006-01-02")
		}
		if !timeRange.End.IsZero() {
			endDate = timeRange.End.In(time.Local).Format("2006-01-02")
		}
	}

	filtered := filterDailyByModels(filterDailyByTime(items, startDate, endDate), models)

	// 按天聚合。
	byDayIndex := map[string]StatsTokenBucket{}
	for _, item := range filtered {
		bucket := byDayIndex[item.Date]
		if bucket.Key == "" {
			bucket = emptyStatsBucket(item.Date, item.Date)
		}
		bucket.ProviderCalls += item.ProviderCalls
		bucket.InputTokens += item.InputTokens
		bucket.OutputTokens += item.OutputTokens
		bucket.CacheReadTokens += item.CacheReadTokens
		bucket.CacheWriteTokens += item.CacheWriteTokens
		bucket.TotalTokens += item.TotalTokens
		byDayIndex[item.Date] = bucket
	}

	// 按模型聚合。
	byModelIndex := map[string]StatsTokenBucket{}
	for _, item := range filtered {
		modelKey := item.Model
		if modelKey == "" {
			modelKey = "unknown"
		}
		bucket := byModelIndex[modelKey]
		if bucket.Key == "" {
			bucket = emptyStatsBucket(modelKey, modelKey)
		}
		bucket.ProviderCalls += item.ProviderCalls
		bucket.InputTokens += item.InputTokens
		bucket.OutputTokens += item.OutputTokens
		bucket.CacheReadTokens += item.CacheReadTokens
		bucket.CacheWriteTokens += item.CacheWriteTokens
		bucket.TotalTokens += item.TotalTokens
		byModelIndex[modelKey] = bucket
	}

	byDay := make([]StatsTokenBucket, 0, len(byDayIndex))
	for _, bucket := range byDayIndex {
		byDay = append(byDay, bucket.withCountedTotals())
	}
	sort.Slice(byDay, func(left int, right int) bool {
		return byDay[left].Key > byDay[right].Key
	})

	byModel := make([]StatsTokenBucket, 0, len(byModelIndex))
	for _, bucket := range byModelIndex {
		byModel = append(byModel, bucket.withCountedTotals())
	}
	sort.Slice(byModel, func(left int, right int) bool {
		return byModel[left].TotalTokens > byModel[right].TotalTokens
	})

	total := aggregateBuckets(byDay).withCountedTotals()
	return StatsResult{
		ByDay:   byDay,
		ByModel: byModel,
		Total:   total,
	}, nil
}

func parseRFC3339Optional(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	// 兼容 YYYY-MM-DD 本地日期：解析为本地当天 00:00（结束日期按当天 23:59:59 由日期闭区间语义覆盖）。
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return parsed, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}