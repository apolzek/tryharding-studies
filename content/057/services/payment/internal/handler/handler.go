package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tryharding/057/payment/internal/chaos"
	"github.com/tryharding/057/payment/internal/idempotency"
	"github.com/tryharding/057/payment/internal/payment"

	"github.com/gin-gonic/gin"
)

type chargeReq struct {
	OrderID string  `json:"order_id" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
}

type chaosReq struct {
	LatencyMs int     `json:"latency_ms"`
	ErrorRate float64 `json:"error_rate"`
}

func Register(r *gin.Engine, gw *payment.Gateway, idem *idempotency.Store, ch *chaos.State) {
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.POST("/charges", func(c *gin.Context) {
		var req chargeReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if status, cached, err := idem.Get(c.Request.Context(), key); err == nil {
			c.Header("Idempotent-Replayed", "true")
			c.Data(status, "application/json", cached)
			return
		}
		status, body, _ := gw.Charge(c.Request.Context(), req.OrderID, req.Amount, key)
		raw, _ := json.Marshal(body)
		if key != "" {
			_ = idem.Put(c.Request.Context(), key, status, raw)
		}
		c.Data(status, "application/json", raw)
	})

	r.GET("/chaos", func(c *gin.Context) { c.JSON(http.StatusOK, ch.Snapshot()) })
	r.POST("/chaos/inject", func(c *gin.Context) {
		var req chaosReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		ch.Set(req.LatencyMs, req.ErrorRate)
		c.JSON(200, ch.Snapshot())
	})
}
