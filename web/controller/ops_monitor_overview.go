package controller

import (
	"fmt"
	"math"
	"sort"
	"time"
)

var opsRanges = map[string]time.Duration{
	"1h":  time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

// OpsOverview returns a cached SLA view. It never samples, scans sockets, reads logs,
// or performs network I/O on the request path.
func OpsOverview(rangeName string) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	duration, ok := opsRanges[rangeName]
	if !ok {
		responseBody.Msg = "range 仅支持 1h、24h、7d、30d"
		return &responseBody
	}

	opsOverviewBuild.Lock()
	defer opsOverviewBuild.Unlock()
	now := time.Now().UTC()
	opsMonitor.RLock()
	if cached, found := opsMonitor.caches[rangeName]; found && now.Before(cached.expiresAt) {
		responseBody.Data = cached.data
		opsMonitor.RUnlock()
		return &responseBody
	}
	storageAvailable := !opsMonitor.storageLoadError && !opsMonitor.storageWriteError
	storageMessage := ""
	if opsMonitor.storageLoadError {
		storageMessage = fmt.Sprintf("历史数据读取不完整，已丢弃 %d 条损坏或不可读记录", opsMonitor.storageDropped)
	} else if opsMonitor.storageWriteError {
		storageMessage = "历史数据写入失败；实时采样仍保留在内存，重启后可能丢失"
	}
	data := buildOpsOverview(opsMonitor.samples, rangeName, duration, now, opsMonitor.trackingSince, storageAvailable, storageMessage)
	opsMonitor.RUnlock()
	opsMonitor.Lock()
	opsMonitor.caches[rangeName] = opsOverviewCacheEntry{data: data, expiresAt: now.Add(opsOverviewCacheTTL)}
	opsMonitor.Unlock()
	responseBody.Data = data
	return &responseBody
}

func buildOpsOverview(allSamples []opsSample, rangeName string, duration time.Duration, now, tracking time.Time, storageAvailable bool, storageMessage string) opsOverviewData {
	windowStart, _, includeStart := opsSchedule(duration, now, tracking)
	first := sort.Search(len(allSamples), func(i int) bool {
		if includeStart {
			return !allSamples[i].At.Before(windowStart)
		}
		return allSamples[i].At.After(windowStart)
	})
	samples := allSamples[first:]
	data := opsOverviewData{
		SchemaVersion: opsSchemaVersion,
		UpdatedAt:     now.Format(time.RFC3339),
		Range:         rangeName,
		Scope:         "server-side-synthetic",
		Status:        "unknown",
		Overview:      summarizeOpsSamples(samples, duration, now, tracking),
		Targets:       defaultOpsTargets(),
		LatencyBuckets: []opsLatencyBucket{
			{LEMS: opsIntPointer(200)}, {LEMS: opsIntPointer(500)}, {LEMS: opsIntPointer(1000)},
			{LEMS: opsIntPointer(3000)}, {LEMS: nil},
		},
		Events:  []opsEvent{},
		History: []opsHistoryPoint{},
	}
	if !tracking.IsZero() {
		data.TrackingSince = tracking.Format(time.RFC3339)
	}
	if len(samples) > 0 {
		latest := samples[len(samples)-1]
		data.Status = latest.Status
		data.Resources = latest.Resources
		data.Targets = append([]opsTargetResult(nil), latest.Targets...)
		lastSample := latest.At.Format(time.RFC3339)
		nextSample := latest.At.Add(opsSampleInterval).Format(time.RFC3339)
		data.Freshness.LastSampleAt = &lastSample
		data.Freshness.NextExpectedAt = &nextSample
		data.Freshness.Stale = now.Sub(latest.At) > 5*opsSampleInterval/2
		if data.Freshness.Stale {
			data.Status = "unknown"
		}
	}
	data.Freshness.SampleIntervalSeconds = int(opsSampleInterval.Seconds())
	data.History = buildOpsHistory(samples, duration, now, tracking, 240)
	data.Events = buildOpsEvents(samples, 50)
	fillLatencyBuckets(data.LatencyBuckets, samples)
	validSynthetic := data.Overview.Success+data.Overview.Failed > 0
	systemAvailable := data.Resources.SampledAt != nil
	data.Sources = []opsSource{
		{ID: "synthetic", Label: "Trojan 服务端合成探测", Scope: "服务端经真实 Trojan 路径访问固定小对象；非外部端到端、非真实用户请求", Available: validSynthetic, Message: syntheticSourceMessage(validSynthetic, data.Targets)},
		{ID: "system", Label: "VPS 系统资源", Scope: "本机 CPU、内存、磁盘、负载、网络速率与 trojan.service 状态", Available: systemAvailable, Message: data.Resources.Error},
		{ID: "persistence", Label: "SLA 历史持久化", Scope: "本机 30 天、64MB 上限的按日 JSONL", Available: storageAvailable, Message: storageMessage},
	}
	if !storageAvailable && data.Status == "healthy" {
		data.Status = "degraded"
	}
	return data
}

func defaultOpsTargets() []opsTargetResult {
	targets := make([]opsTargetResult, 0, len(opsTargets))
	for _, target := range opsTargets {
		targets = append(targets, unknownOpsTarget(target.id, target.name, target.host, "尚无采样数据"))
	}
	return targets
}

func summarizeOpsSamples(samples []opsSample, duration time.Duration, now, tracking time.Time) opsOverviewSummary {
	summary := opsOverviewSummary{SLOTarget: opsSLOTarget}
	var latencies []float64
	for _, sample := range samples {
		for _, target := range sample.Targets {
			switch target.Status {
			case "healthy":
				summary.Success++
				if target.LatencyMS != nil {
					latencies = append(latencies, *target.LatencyMS)
				}
			case "failed":
				summary.Failed++
			}
		}
	}
	if !tracking.IsZero() {
		_, slots, _ := opsSchedule(duration, now, tracking)
		summary.Expected = slots * len(opsTargets)
	}
	observed := summary.Success + summary.Failed
	if summary.Expected < observed {
		summary.Expected = observed
	}
	summary.Unknown = summary.Expected - observed
	if summary.Expected > 0 {
		summary.Coverage = float64(observed) / float64(summary.Expected)
	}
	if observed > 0 {
		availability := float64(summary.Success) / float64(observed)
		summary.Availability = opsFloatPointer(availability)
		budgetFraction := 1 - (1-availability)/(1-opsSLOTarget)
		if budgetFraction < 0 {
			budgetFraction = 0
		}
		if budgetFraction > 1 {
			budgetFraction = 1
		}
		summary.ErrorBudgetRemaining = opsFloatPointer(budgetFraction)
	}
	sort.Float64s(latencies)
	summary.P50MS = percentile(latencies, 0.50)
	summary.P95MS = percentile(latencies, 0.95)
	summary.P99MS = percentile(latencies, 0.99)
	return summary
}

func opsSchedule(duration time.Duration, now, tracking time.Time) (time.Time, int, bool) {
	windowStart := now.Add(-duration)
	if tracking.IsZero() {
		return windowStart, 0, false
	}
	if tracking.After(windowStart) {
		slots := int(math.Floor(now.Sub(tracking).Seconds()/opsSampleInterval.Seconds())) + 1
		if slots < 1 {
			slots = 1
		}
		return tracking, slots, true
	}
	slots := int(math.Ceil(duration.Seconds() / opsSampleInterval.Seconds()))
	if slots < 1 {
		slots = 1
	}
	return windowStart, slots, false
}

func percentile(sortedValues []float64, quantile float64) *float64 {
	if len(sortedValues) == 0 {
		return nil
	}
	if len(sortedValues) == 1 {
		return opsFloatPointer(sortedValues[0])
	}
	position := quantile * float64(len(sortedValues)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	value := sortedValues[lower]
	if upper != lower {
		value += (sortedValues[upper] - sortedValues[lower]) * (position - float64(lower))
	}
	return opsFloatPointer(value)
}

func fillLatencyBuckets(buckets []opsLatencyBucket, samples []opsSample) {
	for _, sample := range samples {
		for _, target := range sample.Targets {
			if target.Status != "healthy" || target.LatencyMS == nil {
				continue
			}
			placed := false
			for index := 0; index < len(buckets)-1; index++ {
				if *target.LatencyMS <= float64(*buckets[index].LEMS) {
					buckets[index].Count++
					placed = true
					break
				}
			}
			if !placed {
				buckets[len(buckets)-1].Count++
			}
		}
	}
}

func buildOpsHistory(samples []opsSample, duration time.Duration, now, tracking time.Time, limit int) []opsHistoryPoint {
	if tracking.IsZero() || limit <= 0 {
		return []opsHistoryPoint{}
	}
	windowStart, slots, includeStart := opsSchedule(duration, now, tracking)
	slotsPerPoint := int(math.Ceil(float64(slots) / float64(limit)))
	pointCount := int(math.Ceil(float64(slots) / float64(slotsPerPoint)))
	points := make([]opsHistoryPoint, pointCount)
	bucketLatencies := make([][]float64, pointCount)
	latestSamples := make([]*opsSample, pointCount)
	for index := range points {
		bucketSlots := min(slotsPerPoint, slots-index*slotsPerPoint)
		points[index].Unknown = bucketSlots * len(opsTargets)
		pointTime := windowStart.Add(time.Duration((index+1)*slotsPerPoint) * opsSampleInterval)
		if includeStart {
			pointTime = windowStart.Add(time.Duration(index*slotsPerPoint) * opsSampleInterval)
		}
		if pointTime.After(now) {
			pointTime = now
		}
		points[index].At = pointTime.Format(time.RFC3339)
	}
	for sampleIndex := range samples {
		sample := &samples[sampleIndex]
		if sample.At.Before(windowStart) || (!includeStart && sample.At.Equal(windowStart)) || sample.At.After(now) {
			continue
		}
		delta := sample.At.Sub(windowStart)
		slot := int(delta / opsSampleInterval)
		if !includeStart && delta%opsSampleInterval == 0 && slot > 0 {
			slot--
		}
		pointIndex := min(slot/slotsPerPoint, pointCount-1)
		latestSamples[pointIndex] = sample
		var sampleLatencies []float64
		for _, target := range sample.Targets {
			switch target.Status {
			case "healthy":
				points[pointIndex].Success++
				points[pointIndex].Unknown--
				if target.LatencyMS != nil {
					sampleLatencies = append(sampleLatencies, *target.LatencyMS)
				}
			case "failed":
				points[pointIndex].Failed++
				points[pointIndex].Unknown--
			}
		}
		bucketLatencies[pointIndex] = append(bucketLatencies[pointIndex], sampleLatencies...)
	}
	for index := range points {
		point := &points[index]
		if point.Unknown < 0 {
			point.Unknown = 0
		}
		if latest := latestSamples[index]; latest != nil {
			point.CPUPercent = latest.Resources.CPUPercent
			point.MemoryPercent = latest.Resources.MemoryPercent
			point.DiskPercent = latest.Resources.DiskPercent
			point.NetworkUpBPS = latest.Resources.NetworkUpBPS
			point.NetworkDownBPS = latest.Resources.NetworkDownBPS
		}
		valid := point.Success + point.Failed
		if valid > 0 {
			point.Availability = opsFloatPointer(float64(point.Success) / float64(valid))
		}
		sort.Float64s(bucketLatencies[index])
		point.P95MS = percentile(bucketLatencies[index], .95)
		switch {
		case point.Failed > 0 && point.Success == 0:
			point.Status = "critical"
		case point.Failed > 0 || point.Unknown > 0:
			point.Status = "degraded"
		case point.Success > 0:
			point.Status = "healthy"
		default:
			point.Status = "unknown"
		}
	}
	return points
}

func buildOpsEvents(samples []opsSample, limit int) []opsEvent {
	if limit <= 0 {
		return []opsEvent{}
	}
	type targetEventState struct {
		status       string
		name         string
		failureStart time.Time
	}
	states := make(map[string]targetEventState, len(opsTargets))
	events := make([]opsEvent, 0, limit)
	appendEvent := func(event opsEvent) {
		if len(events) < limit {
			events = append(events, event)
			return
		}
		copy(events, events[1:])
		events[len(events)-1] = event
	}
	var previousSampleAt time.Time
	for _, sample := range samples {
		if !previousSampleAt.IsZero() && sample.At.Sub(previousSampleAt) > 3*opsSampleInterval/2 {
			unknownAt := previousSampleAt.Add(opsSampleInterval)
			for id, state := range states {
				if state.status != "unknown" {
					appendEvent(opsEvent{At: unknownAt.Format(time.RFC3339), Severity: "warning", Target: state.name, Message: "采样中断，探测状态未知"})
					state.status = "unknown"
					state.failureStart = time.Time{}
					states[id] = state
				}
			}
		}
		seen := make(map[string]bool, len(sample.Targets))
		for _, target := range sample.Targets {
			id := target.ID
			if id == "" {
				id = target.Host
			}
			if id == "" {
				id = target.Name
			}
			seen[id] = true
			state, exists := states[id]
			if state.name == "" {
				state.name = target.Name
			}
			nextStatus := canonicalOpsTargetStatus(target.Status)
			if !exists {
				state.status = nextStatus
				if nextStatus == "failed" {
					state.failureStart = sample.At
					appendEvent(opsEvent{At: sample.At.Format(time.RFC3339), Severity: "critical", Target: state.name, Message: opsTargetFailureMessage(target)})
				} else if nextStatus == "unknown" {
					appendEvent(opsEvent{At: sample.At.Format(time.RFC3339), Severity: "warning", Target: state.name, Message: opsTargetUnknownMessage(target)})
				}
				states[id] = state
				continue
			}
			if nextStatus == state.status {
				continue
			}
			switch nextStatus {
			case "failed":
				state.failureStart = sample.At
				appendEvent(opsEvent{At: sample.At.Format(time.RFC3339), Severity: "critical", Target: state.name, Message: opsTargetFailureMessage(target)})
			case "unknown":
				state.failureStart = time.Time{}
				appendEvent(opsEvent{At: sample.At.Format(time.RFC3339), Severity: "warning", Target: state.name, Message: opsTargetUnknownMessage(target)})
			case "healthy":
				message := "探测恢复"
				if state.status == "failed" && !state.failureStart.IsZero() {
					message += "，故障持续 " + sample.At.Sub(state.failureStart).Round(time.Second).String()
				}
				state.failureStart = time.Time{}
				appendEvent(opsEvent{At: sample.At.Format(time.RFC3339), Severity: "info", Target: state.name, Message: message})
			}
			state.status = nextStatus
			states[id] = state
		}
		for id, state := range states {
			if !seen[id] && state.status != "unknown" {
				appendEvent(opsEvent{At: sample.At.Format(time.RFC3339), Severity: "warning", Target: state.name, Message: "该目标本轮无探测结果"})
				state.status = "unknown"
				state.failureStart = time.Time{}
				states[id] = state
			}
		}
		previousSampleAt = sample.At
	}
	return events
}

func opsTargetFailureMessage(target opsTargetResult) string {
	if target.Error != "" {
		return target.Error
	}
	return "合成探测失败"
}

func opsTargetUnknownMessage(target opsTargetResult) string {
	if target.Error != "" {
		return target.Error
	}
	return "探测状态未知"
}

func syntheticSourceMessage(available bool, targets []opsTargetResult) string {
	if available {
		return "成功率仅代表服务端合成探测，不代表真实用户请求或外部客户端端到端体验"
	}
	for _, target := range targets {
		if target.Error != "" {
			return target.Error
		}
	}
	return "尚无有效合成探测结果"
}
