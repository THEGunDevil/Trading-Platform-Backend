package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/internal/service"
)

func RequireAgent() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)
		if roleStr != "agent" && roleStr != "admin" {
			service.AbortWithError(c, http.StatusForbidden, "agent access required")
			return
		}
		c.Next()
	}
}
