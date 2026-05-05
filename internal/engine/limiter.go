package engine

import (
	"sync"

	"golang.org/x/time/rate"
)

// LimiterManager handles per-phone number rate limiting for multi-tenant isolation.
type LimiterManager struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

func NewLimiterManager() *LimiterManager {
	return &LimiterManager{
		limiters: make(map[string]*rate.Limiter),
	}
}

// Allow checks if the request should be allowed based on tenant and phone number.
// Uses double-checked locking for thread-safe limiter creation.
func (m *LimiterManager) Allow(tenantID, phoneNumberID string) bool {
	key := tenantID + ":" + phoneNumberID

	m.mu.RLock()
	limiter, exists := m.limiters[key]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		// Double-check after acquiring write lock
		limiter, exists = m.limiters[key]
		if !exists {
			// Meta's limit is ~80 msgs/sec, we use 75 for safety with a burst of 10.
			limiter = rate.NewLimiter(75, 10)
			m.limiters[key] = limiter
		}
		m.mu.Unlock()
	}

	return limiter.Allow()
}
