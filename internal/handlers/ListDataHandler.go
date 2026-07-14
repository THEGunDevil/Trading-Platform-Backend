package handlers

// import (
// 	"log"
// 	"math"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
// 	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
// 	"github.com/gin-gonic/gin"
// 	"github.com/jackc/pgx/v5/pgtype"
// )

// type ListReservationPaginatedParams struct {
// 	Limit  int32  `json:"limit"`
// 	Offset int32  `json:"offset"`
// 	Status string `json:"status"` // ← Should be string, not []string
// }

// func timestampToPtr(ts pgtype.Timestamptz) *time.Time {
// 	if ts.Valid {
// 		return &ts.Time
// 	}
// 	return nil
// }
// func ListDataByStatusHandler(c *gin.Context) {
// 	status := strings.ToLower(strings.TrimSpace(c.Query("status"))) // ✅ normalize
// 	log.Printf("📥 Received status: '%s'", status)                   // ✅ debug log

// 	// Pagination
// 	pageQuery := c.DefaultQuery("page", "1")
// 	limitQuery := c.DefaultQuery("limit", "20")

// 	page, err := strconv.Atoi(pageQuery)
// 	if err != nil || page < 1 {
// 		page = 1
// 	}
// 	limit, err := strconv.Atoi(limitQuery)
// 	if err != nil || limit < 1 {
// 		limit = 20
// 	}
// 	offset := (page - 1) * limit

// 	// if roleStr == "admin" {
// 	switch status {
// 	// =====================
// 	// Case: Reservations
// 	// =====================
// 	case "pending", "notified", "fulfilled", "cancelled":
// 		params := gen.ListReservationPaginatedByStatusesParams{
// 			Limit:   int32(limit),
// 			Offset:  int32(offset),
// 			Column3: []string{status},
// 		}

// 		reservations, err := db.Q.ListReservationPaginatedByStatuses(c.Request.Context(), params)
// 		if err != nil {
// 			log.Printf("❌ Failed to fetch reservations: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reservations"})
// 			return
// 		}

// 		var reservationResp []models.ReservationListResponse // ← Use new model
// 		for _, r := range reservations {
// 			reservationResp = append(reservationResp, models.ReservationListResponse{
// 				ID:          r.ID.Bytes,
// 				UserID:      r.UserID.Bytes,
// 				BookID:      r.BookID.Bytes,
// 				Status:      r.Status,
// 				CreatedAt:   r.CreatedAt.Time,
// 				NotifiedAt:  timestampToPtr(r.NotifiedAt),
// 				FulfilledAt: timestampToPtr(r.FulfilledAt),
// 				CancelledAt: timestampToPtr(r.CancelledAt),
// 				UserName:    r.UserName,
// 				UserEmail:   r.Email,
// 				BookTitle:   r.BookTitle,
// 				BookAuthor:  r.Author,
// 				BookImage:   r.ImageUrl,
// 			})
// 		}

// 		c.JSON(http.StatusOK, gin.H{
// 			"reservations": reservationResp,
// 			"page":         page,
// 			"limit":        limit,
// 			"count":        len(reservationResp),
// 		})

// 	// =====================
// 	// Case: Borrowed Books
// 	// =====================
// 	case "borrowed_at":
// 		// Example: use current date to filter borrowed books
// 		params := gen.ListBorrowPaginatedByBorrowedAtParams{
// 			Limit:  int32(limit),
// 			Offset: int32(offset),
// 		}

// 		borrows, err := db.Q.ListBorrowPaginatedByBorrowedAt(c.Request.Context(), params)
// 		if err != nil {
// 			log.Print("failed to fetch borrowed data", err)
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch borrowed data"})
// 			return
// 		}
// 		var borrowResp []models.BorrowResponse
// 		for _, b := range borrows {
// 			borrowResp = append(borrowResp, models.BorrowResponse{
// 				ID:         b.ID.Bytes,
// 				UserID:     b.UserID.Bytes,
// 				UserName:   b.UserName,
// 				BookID:     b.BookID.Bytes,
// 				BorrowedAt: b.BorrowedAt.Time,
// 				DueDate:    b.DueDate.Time,
// 				ReturnedAt: &b.ReturnedAt.Time,
// 				BookTitle:  b.BookTitle,
// 			})
// 		}
// 		c.JSON(http.StatusOK, gin.H{
// 			"borrows": borrowResp,
// 			"page":    page,
// 			"limit":   limit,
// 			"count":   len(borrowResp),
// 		})
// 	case "returned_at":
// 		params := gen.ListBorrowPaginatedByReturnedAtParams{
// 			Limit:  int32(limit),
// 			Offset: int32(offset),
// 		}

// 		borrows, err := db.Q.ListBorrowPaginatedByReturnedAt(c.Request.Context(), params)
// 		if err != nil {
// 			log.Print("failed to fetch borrowed data", err)
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch borrowed data"})
// 			return
// 		}
// 		var borrowResp []models.BorrowResponse
// 		for _, b := range borrows {
// 			borrowResp = append(borrowResp, models.BorrowResponse{
// 				ID:         b.ID.Bytes,
// 				UserID:     b.UserID.Bytes,
// 				UserName:   b.UserName,
// 				BookID:     b.BookID.Bytes,
// 				BorrowedAt: b.BorrowedAt.Time,
// 				DueDate:    b.DueDate.Time,
// 				ReturnedAt: &b.ReturnedAt.Time,
// 				BookTitle:  b.BookTitle,
// 			})
// 		}
// 		c.JSON(http.StatusOK, gin.H{
// 			"borrows": borrowResp,
// 			"page":    page,
// 			"limit":   limit,
// 			"count":   len(borrowResp),
// 		})
// 	case "not_returned":
// 		params := gen.ListBorrowPaginatedByNotReturnedAtParams{
// 			Limit:  int32(limit),
// 			Offset: int32(offset),
// 		}

// 		borrows, err := db.Q.ListBorrowPaginatedByNotReturnedAt(c.Request.Context(), params)
// 		if err != nil {
// 			log.Print("failed to fetch borrowed data", err)
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to fetch borrowed data"})
// 			return
// 		}
// 		var borrowResp []models.BorrowResponse
// 		for _, b := range borrows {
// 			borrowResp = append(borrowResp, models.BorrowResponse{
// 				ID:         b.ID.Bytes,
// 				UserID:     b.UserID.Bytes,
// 				UserName:   b.UserName,
// 				BookID:     b.BookID.Bytes,
// 				BorrowedAt: b.BorrowedAt.Time,
// 				DueDate:    b.DueDate.Time,
// 				ReturnedAt: &b.ReturnedAt.Time,
// 				BookTitle:  b.BookTitle,
// 			})
// 		}
// 		c.JSON(http.StatusOK, gin.H{
// 			"borrows": borrowResp,
// 			"page":    page,
// 			"limit":   limit,
// 			"count":   len(borrowResp),
// 		})
// 	case "user_name", "book_title":
// 		query := strings.TrimSpace(c.Query("query")) // get search term

// 		if query == "" {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Query cannot be empty"})
// 			return
// 		}

// 		option := status // ← Use status as the option (column); remove c.Query("option") and validation

// 		// Fetch total count for pagination
// 		countParams := gen.CountSearchBorrowsByColumnParams{
// 			Column1: option,
// 			Column2: pgtype.Text{String: query, Valid: true},
// 		}
// 		totalCount, err := db.Q.CountSearchBorrowsByColumn(c.Request.Context(), countParams)
// 		if err != nil {
// 			log.Printf("failed to count borrowed data: %v", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count borrowed data"})
// 			return
// 		}
// 		totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

// 		// Fetch paginated results
// 		params := gen.SearchBorrowsWithPaginationParams{
// 			Column1: option,                                  // must be "user_name" or "book_title"
// 			Column2: pgtype.Text{String: query, Valid: true}, // wrap search term properly
// 			Limit:   int32(limit),
// 			Offset:  int32(offset),
// 		}

// 		borrows, err := db.Q.SearchBorrowsWithPagination(c.Request.Context(), params)
// 		if err != nil {
// 			log.Print("failed to fetch borrowed data", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch borrowed data"})
// 			return
// 		}

// 		var borrowResp []models.BorrowResponse
// 		for _, b := range borrows {
// 			borrowResp = append(borrowResp, models.BorrowResponse{
// 				ID:         b.ID.Bytes,
// 				UserID:     b.UserID.Bytes,
// 				UserName:   b.UserName.(string),
// 				BookID:     b.BookID.Bytes,
// 				BorrowedAt: b.BorrowedAt.Time,
// 				DueDate:    b.DueDate.Time,
// 				ReturnedAt: &b.ReturnedAt.Time,
// 				BookTitle:  b.BookTitle,
// 			})
// 		}

// 		c.JSON(http.StatusOK, gin.H{
// 			"borrows":     borrowResp,
// 			"page":        page,
// 			"limit":       limit,
// 			"count":       len(borrowResp),
// 			"total_pages": totalPages, // ← Add this
// 		})

// 	default:
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
// 	}
// 	// } else {
// 	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
// 	// }
// }
