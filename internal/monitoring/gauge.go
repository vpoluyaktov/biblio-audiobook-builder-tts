package monitoring

import (
	"sync"
)

// Labels represents a set of key-value pairs for metric labels
type Labels map[string]string

var (
	gaugesMu sync.RWMutex
	gauges   = make(map[string]map[string]float64) // map[name]map[labelsKey]value
)

// SetGauge sets a gauge value with labels
func SetGauge(name string, value float64, labels Labels) {
	if !metricsEnabled {
		return
	}

	labelsKey := labelsToString(labels)
	
	gaugesMu.Lock()
	defer gaugesMu.Unlock()
	
	if _, exists := gauges[name]; !exists {
		gauges[name] = make(map[string]float64)
	}
	
	gauges[name][labelsKey] = value
}

// GetGauge gets a gauge value with labels
func GetGauge(name string, labels Labels) float64 {
	if !metricsEnabled {
		return 0
	}

	labelsKey := labelsToString(labels)
	
	gaugesMu.RLock()
	defer gaugesMu.RUnlock()
	
	if gaugeMap, exists := gauges[name]; exists {
		if value, ok := gaugeMap[labelsKey]; ok {
			return value
		}
	}
	
	return 0
}

// labelsToString converts a Labels map to a string key
func labelsToString(labels Labels) string {
	if len(labels) == 0 {
		return ""
	}
	
	// Simple implementation for now - in a real system this would be more robust
	result := ""
	for k, v := range labels {
		result += k + "=" + v + ";"
	}
	return result
}
