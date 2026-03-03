package processor

import (
	"testing"
)

func TestNewDataProcessor(t *testing.T) {
	// This would require a Redis instance and config
	// For now, just a basic test structure
	t.Log("DataProcessor test structure created")
}

func TestProcessorMetrics(t *testing.T) {
	metrics := &ProcessorMetrics{
		ProcessedCount: 100,
		FailedCount:    5,
		AvgLatency:     2.5,
	}

	if metrics.ProcessedCount != 100 {
		t.Errorf("Expected ProcessedCount 100, got %d", metrics.ProcessedCount)
	}

	if metrics.FailedCount != 5 {
		t.Errorf("Expected FailedCount 5, got %d", metrics.FailedCount)
	}

	if metrics.AvgLatency != 2.5 {
		t.Errorf("Expected AvgLatency 2.5, got %f", metrics.AvgLatency)
	}
}
