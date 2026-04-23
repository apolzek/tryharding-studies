package handler

import (
	"github.com/tryharding/057/customer/internal/service"

	"github.com/gin-gonic/gin"
)

type createReq struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Document string `json:"document"`
}

func Register(r *gin.Engine, s *service.CustomerService) {
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.POST("/customers", func(c *gin.Context) {
		var req createReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		cust, err := s.Create(c.Request.Context(), req.Name, req.Email, req.Document)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, cust)
	})
	r.GET("/customers/:id", func(c *gin.Context) {
		cust, err := s.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if cust == nil {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		c.JSON(200, cust)
	})
	r.GET("/customers", func(c *gin.Context) {
		list, err := s.List(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, list)
	})
}
