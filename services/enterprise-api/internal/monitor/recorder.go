package monitor

import (
	"math"
	"regexp"
	"sort"
	"sync"
	"time"
)

var uuidPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

const maxSamples = 4000

// Sample is one finished HTTP request.
type Sample struct {
	Time      time.Time
	Method    string
	Path      string
	Status    int
	Duration  time.Duration
	TimedOut  bool
	RequestID string
}

// Recorder keeps a bounded window of request samples.
type Recorder struct {
	mu      sync.Mutex
	samples []Sample
}

// Add stores one sample and drops the oldest past the cap.
func (r *Recorder) Add(sample Sample) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.samples = append(r.samples, sample)
	if len(r.samples) > maxSamples {
		r.samples = append([]Sample(nil), r.samples[len(r.samples)-maxSamples:]...)
	}
}

// Snapshot aggregates samples newer than the window.
func (r *Recorder) Snapshot(window time.Duration, now time.Time) (Traffic, []ErrorStat, []RecentError) {
	if r == nil {
		return Traffic{}, []ErrorStat{}, []RecentError{}
	}
	cutoff := now.Add(-window)
	r.mu.Lock()
	kept := r.samples[:0]
	for _, sample := range r.samples {
		if !sample.Time.Before(cutoff) {
			kept = append(kept, sample)
		}
	}
	r.samples = kept
	current := append([]Sample(nil), r.samples...)
	r.mu.Unlock()

	var traffic Traffic
	durations := make([]int, 0, len(current))
	groups := map[string]*ErrorStat{}
	recent := make([]RecentError, 0)
	var totalMS int64
	for _, sample := range current {
		traffic.Requests++
		ms := int(sample.Duration / time.Millisecond)
		durations = append(durations, ms)
		totalMS += int64(ms)
		kind := classify(sample)
		switch kind {
		case "timeout":
			traffic.Timeouts++
		case "rejected":
			traffic.Rejected++
		case "server_error":
			traffic.ServerErrors++
		case "client_error":
			traffic.ClientErrors++
		}
		if kind == "ok" {
			continue
		}
		label := sampleLabel(sample)
		stat := groups[label]
		if stat == nil {
			stat = &ErrorStat{Type: label}
			groups[label] = stat
		}
		stat.Count++
		if sample.Time.After(stat.LastTime) {
			stat.LastTime = sample.Time
		}
		recent = append(recent, RecentError{
			Time:       sample.Time,
			Method:     sample.Method,
			Path:       sample.Path,
			Status:     sample.Status,
			DurationMS: ms,
			Type:       label,
			RequestID:  sample.RequestID,
		})
	}
	if traffic.Requests > 0 {
		traffic.AvgMS = int(totalMS / int64(traffic.Requests))
		traffic.P95MS = percentile(durations, 0.95)
	}
	errors := make([]ErrorStat, 0, len(groups))
	for _, stat := range groups {
		errors = append(errors, *stat)
	}
	sort.Slice(errors, func(i, j int) bool {
		if errors[i].Count == errors[j].Count {
			return errors[i].Type < errors[j].Type
		}
		return errors[i].Count > errors[j].Count
	})
	sort.Slice(recent, func(i, j int) bool { return recent[i].Time.After(recent[j].Time) })
	if len(recent) > 30 {
		recent = recent[:30]
	}
	return traffic, errors, recent
}

func classify(sample Sample) string {
	if sample.TimedOut || sample.Status == 504 {
		return "timeout"
	}
	if sample.Status == 413 || sample.Status == 429 {
		return "rejected"
	}
	if sample.Status >= 500 {
		return "server_error"
	}
	if sample.Status >= 400 {
		return "client_error"
	}
	return "ok"
}

func sampleLabel(sample Sample) string {
	if sample.TimedOut || sample.Status == 504 {
		return "timeout"
	}
	return itoa(sample.Status)
}

func percentile(values []int, p float64) int {
	if len(values) == 0 {
		return 0
	}
	sort.Ints(values)
	index := int(math.Ceil(p*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func redactPath(path string) string {
	path = uuidPattern.ReplaceAllString(path, "{id}")
	if len(path) > 160 {
		return path[:160]
	}
	return path
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buf [16]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
