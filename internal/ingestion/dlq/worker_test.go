package dlq

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	worker := NewRetryWorker(nil, nil)
	worker.baseDelay = 2 * time.Second
	worker.maxDelay = 60 * time.Second

	tests := []struct {
		retryCount int
		minBound   time.Duration
		maxBound   time.Duration
	}{
		{retryCount: 0, minBound: 2 * time.Second, maxBound: 3 * time.Second},
		{retryCount: 1, minBound: 4 * time.Second, maxBound: 6 * time.Second},
		{retryCount: 2, minBound: 8 * time.Second, maxBound: 11 * time.Second},
		{retryCount: 3, minBound: 16 * time.Second, maxBound: 21 * time.Second},
		{retryCount: 6, minBound: 60 * time.Second, maxBound: 73 * time.Second}, // Capped at maxDelay + jitter
	}

	for _, tt := range tests {
		delay := worker.CalculateBackoff(tt.retryCount)
		if delay < tt.minBound || delay > tt.maxBound {
			t.Errorf("retryCount %d: expected delay between %v and %v, got %v",
				tt.retryCount, tt.minBound, tt.maxBound, delay)
		}
	}
}
