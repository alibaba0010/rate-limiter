package limiter

import (
	"context"
	"math"
	"sync"
	"time"
)

// SlidingWindow implements the Strategy interface using the sliding window counter algorithm.
// It approximates the request rate by combining the count of the current window and the previous window.
type SlidingWindow struct {
	mu      sync.Mutex
	windows map[string]*windowState
}

type windowState struct {
	currWindowStart time.Time
	currCount       int
	prevCount       int
}

// NewSlidingWindow creates a new instance of SlidingWindow strategy.
func NewSlidingWindow() *SlidingWindow {
	return &SlidingWindow{
		windows: make(map[string]*windowState),
	}
}

// Allow checks if the request is allowed based on the sliding window algorithm.
func (sw *SlidingWindow) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	w, exists := sw.windows[key]
	if !exists {
		w = &windowState{
			currWindowStart: now,
			currCount:       0,
			prevCount:       0,
		}
		sw.windows[key] = w
	}

	// Calculate how many windows have passed
	elapsed := now.Sub(w.currWindowStart)
	if elapsed >= limit.Period {
		windowsPassed := int(elapsed / limit.Period)
		
		// If 1 window passed, the current becomes previous
		if windowsPassed == 1 {
			w.prevCount = w.currCount
		} else {
			// If more than 1 window passed, previous window is too old
			w.prevCount = 0
		}
		// Reset current count
		w.currCount = 0
		// Update window start time
		w.currWindowStart = w.currWindowStart.Add(time.Duration(windowsPassed) * limit.Period)
	}

	// Calculate the weighted count
	// Requests in previous window * (Time remaining in current window / Window size) + Requests in current window
	timeInCurrent := now.Sub(w.currWindowStart).Seconds()
	windowSize := limit.Period.Seconds()
	
	// Weight of the previous window
	weight := math.Max(0, (windowSize-timeInCurrent)/windowSize)
	
	estimatedCount := float64(w.prevCount)*weight + float64(w.currCount)

	result := &Result{}
	if estimatedCount < float64(limit.Rate) {
		w.currCount++
		result.Allowed = true
		result.Remaining = int(float64(limit.Rate) - estimatedCount - 1)
		if result.Remaining < 0 {
			result.Remaining = 0
		}
		result.ResetAfter = 0 
	} else {
		result.Allowed = false
		result.Remaining = 0
		// Roughly estimate wait time as time until end of current window
		result.ResetAfter = w.currWindowStart.Add(limit.Period).Sub(now)
	}

	return result, nil
}
