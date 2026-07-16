package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/config"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/handlers"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/middleware"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, relying on OS environment variables")
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	cldURL := fmt.Sprintf("cloudinary://%s:%s@%s", apiKey, apiSecret, cloudName)
	service.InitCloudinary(cldURL)

	cfg := config.LoadConfig()

	// TODO: switch to db.Connect(cfg) with cfg.DBURL before deploying to production.
	// db.Connect(cfg)
	db.LocalConnect(cfg)
	defer db.Close()
	store := db.NewStore(db.DB)
	orderSvc := service.NewOrderService(store)
	orderHandler := handlers.NewOrderHandler(orderSvc)
	r := gin.New()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RateLimiter()) // re-enabled — SkipRateLimit() below only makes sense if this is active

	// Health check
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes (public)
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", handlers.RegisterHandler)
		authGroup.POST("/signin", handlers.LoginHandler)
		authGroup.POST("/refresh", handlers.RefreshHandler)
		authGroup.POST("/logout", handlers.LogoutHandler)
	}

	// User routes (protected)
	userGroup := r.Group("/users")
	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/", middleware.SkipRateLimit(), middleware.AdminOnly(), handlers.GetUsersHandler)
		userGroup.GET("/user/email", middleware.AdminOnly(), middleware.SkipRateLimit(), handlers.SearchUsersPaginatedHandler)
		userGroup.GET("/user/:id", handlers.GetUserByIDHandler)
		userGroup.PATCH("/user/:id", handlers.UpdateUserByIDHandler)
		userGroup.PATCH("/user/ban/:id", middleware.AdminOnly(), handlers.BanUserByIDHandler)
	}
	// Balance routes (protected)
	balanceGroup := r.Group("/api/balances")
	balanceGroup.Use(middleware.AuthMiddleware())
	{
		balanceGroup.GET("/", handlers.ListBalances)
		balanceGroup.GET("/:asset", handlers.GetBalance)
	}
	bannedUserGroup := r.Group("/banned-users")
	bannedUserGroup.Use(middleware.AuthMiddleware())
	{
		bannedUserGroup.GET("/", middleware.AdminOnly(), handlers.GetUsersHandler)
	}
	// 3. Order routes (protected)
	orderGroup := r.Group("/api/orders")
	orderGroup.Use(middleware.AuthMiddleware())
	{
		// This is where you actually USE orderHandler!
		orderGroup.POST("/", orderHandler.PlaceOrder)
		orderGroup.DELETE("/:id", orderHandler.CancelOrder)
	}
	// Prediction routes
	predictionGroup := r.Group("/predictions")
	predictionGroup.Use(middleware.AuthMiddleware())
	{
		predictionGroup.POST("/place", handlers.PlacePrediction)
		predictionGroup.GET("/result/:id", handlers.GetPredictionResult)
		predictionGroup.GET("/active", handlers.GetActivePredictions)
		predictionGroup.GET("/history", handlers.GetPredictionHistory)
		predictionGroup.POST("/cancel/:id", handlers.CancelPrediction)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Server running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}

// Notification routes (protected)
// notificationGroup := r.Group("/notifications")
// notificationGroup.Use(middleware.AuthMiddleware())
// {
// 	notificationGroup.GET("/get", handlers.GetUserNotificationsByUserIDHandler)
// 	notificationGroup.PATCH("/mark-read", handlers.MarkAllNotificationsAsReadHandler)
// }
// List routes (protected, admin only)
// listGroup := r.Group("/list")
// listGroup.Use(middleware.AuthMiddleware(), middleware.AdminOnly()) // ← Auth MUST come before Admin
// {
// 	listGroup.GET("/data-paginated", handlers.ListDataByStatusHandler)
// }
