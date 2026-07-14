package handlers

// import (
// 	"log"
// 	"net/http"

// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
// 	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// 	"github.com/jackc/pgx/v5/pgtype"
// )

// // GetUserNotificationsByUserIDHandler fetches notifications for a user with optional pagination
// func GetUserNotificationsByUserIDHandler(c *gin.Context) {
// 	userIDVal, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "userID not found in context"})
// 		return
// 	}

// 	userID, ok := userIDVal.(uuid.UUID)
// 	if !ok {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid userID type"})
// 		return
// 	}

// 	// Pagination parameters
// 	limit := int32(50)
// 	offset := int32(0)

// 	params := gen.GetUserNotificationsByUserIDParams{
// 		ID:     pgtype.UUID{Bytes: userID, Valid: true},
// 		Limit:  limit,
// 		Offset: offset,
// 	}

// 	notifications, err := db.Q.GetUserNotificationsByUserID(
// 		c.Request.Context(),
// 		params,
// 	)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
// 		return
// 	}

// 	// ✅ FIX: Initialize as empty slice so it returns [] instead of null
// 	response := make([]models.Notification, 0)

// 	for _, n := range notifications {
// 		var objectID *uuid.UUID
// 		if n.ObjectID.Valid {
// 			id := n.ObjectID.Bytes
// 			uuidVal, err := uuid.FromBytes(id[:])
// 			if err == nil {
// 				objectID = &uuidVal
// 			}
// 		}

// 		response = append(response, models.Notification{
// 			ID:                n.EventID.Bytes,
// 			UserID:            userID,
// 			UserName:          "",
// 			ObjectID:          objectID,
// 			ObjectTitle:       n.ObjectTitle.String,
// 			Type:              n.Type,
// 			NotificationTitle: n.NotificationTitle,
// 			Message:           n.Message,
// 			Metadata:          n.Metadata,
// 			IsRead:            n.IsRead, // This is now correct (false for broadcast, true if read)
// 			CreatedAt:         n.CreatedAt.Time,
// 		})
// 	}

// 	c.JSON(http.StatusOK, response)
// }

// // MarkAllNotificationsAsReadHandler marks all unread notifications for the user as read
// func MarkAllNotificationsAsReadHandler(c *gin.Context) {
// 	userIDVal, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 		return
// 	}

// 	userUUID, ok := userIDVal.(uuid.UUID)
// 	if !ok {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID in context"})
// 		return
// 	}

// 	err := db.Q.MarkAllNotificationsAsRead(
// 		c.Request.Context(),
// 		pgtype.UUID{Bytes: userUUID, Valid: true},
// 	)

// 	if err != nil {
// 		log.Printf("Failed to mark all as read for user %v: %v", userUUID, err)
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read successfully"})
// }
