package handlers

// import (
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
// 	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
// 	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
// 	"github.com/gin-gonic/gin"
// )

// func OverviewHandler(c *gin.Context) {
// 	now := time.Now()

// 	// -------------------- STATS --------------------
// 	row, err := db.Q.GetStats(c, gen.GetStatsParams{
// 		Column1: int32(now.Month()),
// 		Column2: int32(now.Year()),
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// RevenueMonth may be string or float64 depending on DB type
// 	var rev float64
// 	switch v := row.RevenueMonth.(type) {
// 	case float64:
// 		rev = v
// 	case string:
// 		rev, _ = strconv.ParseFloat(v, 64)
// 	}

// 	res := models.OverviewResponse{
// 		Stats: models.Stats{
// 			TotalBooks:         int(row.TotalBooks),
// 			ActiveUsers:        int(row.ActiveUsers),
// 			TotalSubscriptions: int(row.TotalSubscriptions),
// 			RevenueMonth:       rev,
// 		},
// 	}

// 	c.JSON(http.StatusOK, res)
// }
