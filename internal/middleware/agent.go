package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
)

func RequireAgent() gin.HandlerFunc {
    return func(c *gin.Context) {
        role, _ := c.Get("role") // AuthMiddleware sets this
        if role != "agent" && role != "admin" {
            service.AbortWithError(c, http.StatusForbidden, "agent access required")
            return
        }
        c.Next()
    }
}