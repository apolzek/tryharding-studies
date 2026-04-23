package chaos

import (
	"math/rand"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type State struct {
	mu          sync.RWMutex
	LatencyMs   int
	ErrorRate   float64
}

func New() *State { return &State{} }

func (s *State) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{"latency_ms": s.LatencyMs, "error_rate": s.ErrorRate}
}

func (s *State) Set(latencyMs int, errRate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if latencyMs < 0 { latencyMs = 0 }
	if errRate < 0 { errRate = 0 }
	if errRate > 1 { errRate = 1 }
	s.LatencyMs = latencyMs
	s.ErrorRate = errRate
}

func Middleware(s *State) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/chaos" || c.Request.URL.Path == "/chaos/inject" {
			c.Next()
			return
		}
		s.mu.RLock()
		lat := s.LatencyMs
		er := s.ErrorRate
		s.mu.RUnlock()
		if lat > 0 {
			time.Sleep(time.Duration(lat) * time.Millisecond)
		}
		if er > 0 && rand.Float64() < er {
			c.AbortWithStatusJSON(503, gin.H{"error": "chaos: injected failure"})
			return
		}
		c.Next()
	}
}
