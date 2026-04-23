package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/tryharding/057/auth/internal/service"

	"github.com/gin-gonic/gin"
)

type loginReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(r *gin.Engine, s *service.AuthService) {
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.POST("/register", func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		id, err := s.Register(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, gin.H{"id": id})
	})

	r.POST("/login", func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		tok, err := s.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"access_token": tok, "token_type": "Bearer"})
	})

	r.GET("/validate", func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		tok := strings.TrimPrefix(h, "Bearer ")
		sub, err := s.Validate(tok)
		if err != nil {
			c.JSON(401, gin.H{"error": "invalid token"})
			return
		}
		c.JSON(200, gin.H{"sub": sub})
	})
}
