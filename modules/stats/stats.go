package stats

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/isa-programmer/goscan/modules/utils"
)

// RequestStats holds statistics for individual requests
type RequestStats struct {
	URL          string
	StatusCode   int
	ResponseTime time.Duration
	ResponseSize int64
	Timestamp    time.Time
	Error        string
}

// ScanStats holds overall scan statistics
type ScanStats struct {
	mu                sync.RWMutex
	StartTime         time.Time
	EndTime           time.Time
	TotalRequests     int64
	SuccessfulReqs    int64
	FailedReqs        int64
	TotalBytes        int64
	StatusCodeCounts  map[int]int64
	ResponseTimes     []time.Duration
	RequestHistory    []RequestStats
	ErrorCounts       map[string]int64
	MaxResponseTime   time.Duration
	MinResponseTime   time.Duration
	AvgResponseTime   time.Duration
	RequestsPerSecond float64
}

// StatsManager manages scan statistics
type StatsManager struct {
	Stats *ScanStats
}

// New creates a new StatsManager
func New() *StatsManager {
	return &StatsManager{
		Stats: &ScanStats{
			StartTime:        time.Now(),
			StatusCodeCounts: make(map[int]int64),
			ResponseTimes:    make([]time.Duration, 0),
			RequestHistory:   make([]RequestStats, 0),
			ErrorCounts:      make(map[string]int64),
			MinResponseTime:  time.Duration(math.MaxInt64),
			MaxResponseTime:  time.Duration(0),
		},
	}
}

// RecordRequest records statistics for a single request
func (sm *StatsManager) RecordRequest(url string, statusCode int, responseTime time.Duration, responseSize int64, err error) {
	sm.Stats.mu.Lock()
	defer sm.Stats.mu.Unlock()

	sm.Stats.TotalRequests++
	sm.Stats.TotalBytes += responseSize
	sm.Stats.StatusCodeCounts[statusCode]++
	sm.Stats.ResponseTimes = append(sm.Stats.ResponseTimes, responseTime)

	// Update min/max response times
	if responseTime < sm.Stats.MinResponseTime {
		sm.Stats.MinResponseTime = responseTime
	}
	if responseTime > sm.Stats.MaxResponseTime {
		sm.Stats.MaxResponseTime = responseTime
	}

	// Record request details
	reqStats := RequestStats{
		URL:          url,
		StatusCode:   statusCode,
		ResponseTime: responseTime,
		ResponseSize: responseSize,
		Timestamp:    time.Now(),
	}

	if err != nil {
		sm.Stats.FailedReqs++
		reqStats.Error = err.Error()
		sm.Stats.ErrorCounts[err.Error()]++
	} else {
		sm.Stats.SuccessfulReqs++
	}

	sm.Stats.RequestHistory = append(sm.Stats.RequestHistory, reqStats)

	// Calculate running average response time
	n := time.Duration(len(sm.Stats.ResponseTimes))
	if n == 1 {
		sm.Stats.AvgResponseTime = responseTime
	} else {
		// Running average: new_avg = old_avg + (new_value - old_avg) / n
		sm.Stats.AvgResponseTime = sm.Stats.AvgResponseTime + (responseTime-sm.Stats.AvgResponseTime)/n
	}

	// Calculate requests per second
	elapsed := time.Since(sm.Stats.StartTime)
	if elapsed > 0 {
		sm.Stats.RequestsPerSecond = float64(sm.Stats.TotalRequests) / elapsed.Seconds()
	}
}

// FinalizeScan finalizes the scan statistics
func (sm *StatsManager) FinalizeScan() {
	sm.Stats.mu.Lock()
	defer sm.Stats.mu.Unlock()

	sm.Stats.EndTime = time.Now()

	// Final calculations
	elapsed := sm.Stats.EndTime.Sub(sm.Stats.StartTime)
	if elapsed > 0 {
		sm.Stats.RequestsPerSecond = float64(sm.Stats.TotalRequests) / elapsed.Seconds()
	}
}

// GetElapsedTime returns the elapsed time since scan start
func (sm *StatsManager) GetElapsedTime() time.Duration {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	if sm.Stats.EndTime.IsZero() {
		return time.Since(sm.Stats.StartTime)
	}
	return sm.Stats.EndTime.Sub(sm.Stats.StartTime)
}

// GetRequestsPerSecond returns the current requests per second rate
func (sm *StatsManager) GetRequestsPerSecond() float64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.RequestsPerSecond
}

// GetTotalRequests returns the total number of requests made
func (sm *StatsManager) GetTotalRequests() int64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.TotalRequests
}

// GetSuccessfulRequests returns the number of successful requests
func (sm *StatsManager) GetSuccessfulRequests() int64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.SuccessfulReqs
}

// GetFailedRequests returns the number of failed requests
func (sm *StatsManager) GetFailedRequests() int64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.FailedReqs
}

// GetStatusCodeCounts returns a map of status codes and their counts
func (sm *StatsManager) GetStatusCodeCounts() map[int]int64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	// Return a copy to avoid race conditions
	counts := make(map[int]int64)
	for code, count := range sm.Stats.StatusCodeCounts {
		counts[code] = count
	}
	return counts
}

// GetAverageResponseTime returns the average response time
func (sm *StatsManager) GetAverageResponseTime() time.Duration {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.AvgResponseTime
}

// GetMinResponseTime returns the minimum response time
func (sm *StatsManager) GetMinResponseTime() time.Duration {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.MinResponseTime
}

// GetMaxResponseTime returns the maximum response time
func (sm *StatsManager) GetMaxResponseTime() time.Duration {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.MaxResponseTime
}

// GetTotalBytes returns the total bytes received
func (sm *StatsManager) GetTotalBytes() int64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	return sm.Stats.TotalBytes
}

// GetErrorCounts returns a map of errors and their counts
func (sm *StatsManager) GetErrorCounts() map[string]int64 {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	// Return a copy to avoid race conditions
	errorCounts := make(map[string]int64)
	for err, count := range sm.Stats.ErrorCounts {
		errorCounts[err] = count
	}
	return errorCounts
}

// GetPercentileResponseTime calculates the percentile response time
func (sm *StatsManager) GetPercentileResponseTime(percentile float64) time.Duration {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	if len(sm.Stats.ResponseTimes) == 0 {
		return 0
	}

	// Create a copy and sort it
	times := make([]time.Duration, len(sm.Stats.ResponseTimes))
	copy(times, sm.Stats.ResponseTimes)
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	index := int(float64(len(times)) * percentile / 100.0)
	if index >= len(times) {
		index = len(times) - 1
	}

	return times[index]
}

// PrintSummary prints a summary of the scan statistics
func (sm *StatsManager) PrintSummary() {
	sm.Stats.mu.RLock()
	defer sm.Stats.mu.RUnlock()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("\x1b[38;5;6mSCAN STATISTICS\x1b[0m")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("\x1b[38;5;2mTotal Requests:\x1b[0m %d\n", sm.Stats.TotalRequests)
	fmt.Printf("\x1b[38;5;2mSuccessful:\x1b[0m %d\n", sm.Stats.SuccessfulReqs)
	fmt.Printf("\x1b[38;5;1mFailed:\x1b[0m %d\n", sm.Stats.FailedReqs)
	fmt.Printf("\x1b[38;5;3mTotal Bytes:\x1b[0m %s\n", utils.FormatBytes(sm.Stats.TotalBytes))
	fmt.Printf("\x1b[38;5;4mElapsed Time:\x1b[0m %v\n", sm.GetElapsedTime())
	fmt.Printf("\x1b[38;5;5mRequests/sec:\x1b[0m %.2f\n", sm.Stats.RequestsPerSecond)

	if len(sm.Stats.ResponseTimes) > 0 {
		fmt.Printf("\x1b[38;5;6mAvg Response Time:\x1b[0m %v\n", sm.Stats.AvgResponseTime)
		fmt.Printf("\x1b[38;5;6mMin Response Time:\x1b[0m %v\n", sm.Stats.MinResponseTime)
		fmt.Printf("\x1b[38;5;6mMax Response Time:\x1b[0m %v\n", sm.Stats.MaxResponseTime)
		fmt.Printf("\x1b[38;5;6m95th Percentile:\x1b[0m %v\n", sm.GetPercentileResponseTime(95))
	}

	if len(sm.Stats.StatusCodeCounts) > 0 {
		fmt.Println("\n\x1b[38;5;6mStatus Code Distribution:\x1b[0m")
		for code, count := range sm.Stats.StatusCodeCounts {
			color := getStatusCodeColor(code)
			fmt.Printf("  %s%d:\x1b[0m %d\n", color, code, count)
		}
	}

	if len(sm.Stats.ErrorCounts) > 0 {
		fmt.Println("\n\x1b[38;5;1mTop Errors:\x1b[0m")
		count := 0
		for err, errCount := range sm.Stats.ErrorCounts {
			if count >= 5 { // Show only top 5 errors
				break
			}
			fmt.Printf("  %s: %d\n", err, errCount)
			count++
		}
	}

	fmt.Println(strings.Repeat("=", 50))
}



// getStatusCodeColor returns the appropriate color for a status code
func getStatusCodeColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "\x1b[38;5;2m" // Green
	case code >= 300 && code < 400:
		return "\x1b[38;5;3m" // Yellow
	case code >= 400 && code < 500:
		return "\x1b[38;5;1m" // Red
	case code >= 500:
		return "\x1b[38;5;5m" // Magenta
	default:
		return "\x1b[0m" // Default
	}
}