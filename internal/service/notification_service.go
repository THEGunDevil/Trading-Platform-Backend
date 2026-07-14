package service

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"github.com/THEGunDevil/GoForBackend/internal/db"
// 	gen "github.com/THEGunDevil/GoForBackend/internal/db/gen"
// 	"github.com/THEGunDevil/GoForBackend/internal/models"
// 	"github.com/google/uuid"
// 	"github.com/jackc/pgx/v5/pgtype"
// )

// // NotificationService handles creating event-based notifications
// func NotificationService(ctx context.Context, req models.SendNotificationRequest) error {
// 	log.Printf("🔔 [DEBUG] NotificationService called for Type=%s | Title=%s",
// 		req.Type, req.NotificationTitle)

// 	// Validate optional ObjectID
// 	var pgObjectID pgtype.UUID
// 	if req.ObjectID != nil {
// 		pgObjectID = UUIDToPGType(*req.ObjectID)
// 	} else {
// 		pgObjectID = pgtype.UUID{Valid: false} // NULL in DB
// 	}

// 	// Prepare params for CreateEvent
// 	eventArg := gen.CreateEventParams{
// 		ObjectID:    pgObjectID,
// 		ObjectTitle: StringToPGText(req.ObjectTitle),
// 		Type:        req.Type,
// 		Title:       req.NotificationTitle,
// 		Message:     req.Message,
// 		Metadata:    req.Metadata,
// 	}

// 	// Insert event into events table
// 	event, err := db.Q.CreateEvent(ctx, eventArg)
// 	if err != nil {
// 		log.Printf("❌ [DEBUG] Failed to create event: %v", err)
// 		return fmt.Errorf("failed to create event: %w", err)
// 	}
// 	log.Printf("✅ [DEBUG] Event created successfully: ID=%v", event.ID)

// 	// LOGIC CHANGE:
// 	// Only assign to user_notification_status if we are targeting a SPECIFIC user.
// 	// If it is a broadcast (UserID is nil/empty), do nothing.
// 	if req.UserID != uuid.Nil {
// 		assignArg := gen.AssignNotificationToUserParams{
// 			UserID:  UUIDToPGType(req.UserID),
// 			EventID: event.ID,
// 		}

// 		// Use the new Assign function (sets is_read = FALSE)
// 		err := db.Q.AssignNotificationToUser(ctx, assignArg)
// 		if err != nil {
// 			log.Printf("❌ Failed to assign notification to user: %v", err)
// 			return fmt.Errorf("failed to assign notification: %w", err)
// 		}
// 		log.Printf("✅ User notification status created (Unread) for UserID=%v", req.UserID)
// 	} else {
// 		log.Printf("📢 [DEBUG] Broadcast notification created (No specific assignment needed)")
// 	}

// 	return nil
// }