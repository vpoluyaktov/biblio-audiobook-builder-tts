package monitoring

import (
	"sync"
)

var metricsEnabled bool

type Metrics struct {
	mu       sync.RWMutex
	name     string
	counters map[string]int64
}

func NewMetrics(name string) *Metrics {
	return &Metrics{
		name:     name,
		counters: make(map[string]int64),
	}
}

// EnableMetrics enables metrics collection
func EnableMetrics() {
	metricsEnabled = true
}

// DisableMetrics disables metrics collection
func DisableMetrics() {
	metricsEnabled = false
}

// IsMetricsEnabled returns whether metrics collection is enabled
func IsMetricsEnabled() bool {
	return metricsEnabled
}

// IncrementCounter increments a counter by 1
func (m *Metrics) IncrementCounter(name string) {
	if !metricsEnabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

// GetCounter returns the value of a counter
func (m *Metrics) GetCounter(name string) int64 {
	if !metricsEnabled {
		return 0
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}
