package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// allowedPathsForBannedUsers lists exact route patterns (as registered with
// Gin, e.g. "/users/user/:id" if it has a param) that a banned user may
// still access. Matched against c.FullPath(), which returns the route
// template Gin matched — NOT the literal request URL — so entries here
// must exactly match how the route is registered below in main.go.
var allowedPathsForBannedUsers = map[string]bool{
	"/users/user": true,
	"/contact/send": true,
}

// setUserContext stores the authenticated user's info on the Gin context so
// downstream handlers can read it via c.Get(...). Centralized here so the
// normal-user and banned-user code paths can't drift out of sync with each
// other over time.
func setUserContext(c *gin.Context, userUUID uuid.UUID, user gen.User) {
	c.Set("userID", userUUID)
	c.Set("role", user.Role.String)
	c.Set("isBanned", user.IsBanned.Bool)
	c.Set("isPermanentBan", user.IsPermanentBan.Bool)
	c.Set("banReason", user.BanReason.String)
	c.Set("banUntil", user.BanUntil.Time)
}

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("🔹 AuthMiddleware started")
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("❌ Authorization header missing")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
			return
		}
		log.Println("✅ Authorization header found")

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			log.Printf("❌ Invalid auth header format: %v\n", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		token, err := service.VerifyToken(tokenString, false)
		if err != nil {
			log.Printf("❌ Token verification failed: %v\n", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		log.Println("✅ Token verified successfully")

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			log.Println("❌ Invalid token claims")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		log.Printf("✅ Token claims: %+v\n", claims)

		subStr, ok := claims["sub"].(string)
		if !ok {
			log.Println("❌ Missing sub claim")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing sub claim"})
			return
		}

		userUUID, err := uuid.Parse(subStr)
		if err != nil {
			log.Printf("❌ Invalid user UUID: %v\n", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
			return
		}
		log.Printf("✅ User UUID: %v\n", userUUID)

		user, err := db.Q.GetUserByID(c.Request.Context(), pgtype.UUID{Bytes: userUUID, Valid: true})
		if err != nil {
			log.Printf("❌ User not found: %v\n", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		log.Printf("✅ User fetched: %s\n", user.UserName)

		// Explicitly validate the token_version claim's presence and type,
		// rather than silently defaulting to 0 on a missing/malformed claim.
		tokenVersionRaw, ok := claims["token_version"]
		if !ok {
			log.Println("❌ Missing token_version claim")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "malformed token"})
			return
		}
		tokenVersionFloat, ok := tokenVersionRaw.(float64)
		if !ok {
			log.Println("❌ token_version claim has unexpected type")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "malformed token"})
			return
		}
		tokenVersion := int32(tokenVersionFloat)

		if tokenVersion != user.TokenVersion {
			log.Printf("❌ Token version mismatch: token=%v, user=%v\n", tokenVersion, user.TokenVersion)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}
		log.Println("✅ Token version validated")

		// Handle banned users
		if user.IsBanned.Bool {
			log.Println("⚠️ User is banned")

			if allowedPathsForBannedUsers[c.FullPath()] {
				log.Printf("✅ Banned user accessing allowed route %s\n", c.FullPath())
				setUserContext(c, userUUID, user)
				c.Set("banned_user", true)
				c.Set("token_version", int(tokenVersion))
				c.Next()
				return
			}

			log.Printf("❌ Banned user tried to access: %s\n", c.FullPath())
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":            "your account is banned",
				"reason":           user.BanReason.String,
				"is_permanent_ban": user.IsPermanentBan.Bool,
				"ban_until":        user.BanUntil.Time,
			})
			return
		}

		// Normal (non-banned) user
		setUserContext(c, userUUID, user)
		c.Set("token_version", int(tokenVersion))

		log.Println("✅ Context set for downstream handlers")
		c.Next()
		log.Println("🔹 AuthMiddleware finished")
	}
}

// AdminOnly ensures the request is from an admin
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}