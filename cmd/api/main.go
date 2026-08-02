package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/internal/config"
	"github.com/internal/db"
	"github.com/internal/handlers"
	"github.com/internal/middleware"
	"github.com/internal/service"
	"github.com/internal/ws"
	"github.com/joho/godotenv" // added for .env loading
)

func main() {
	_ = godotenv.Load()

	// ── Environment ────────────────────────────────────────────────────────────
	if err := service.Init(); err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	// ── Database ───────────────────────────────────────────────────────────────
	cfg := config.LoadConfig()
	db.Connect(cfg)
	// db.LocalConnect(cfg) // use local connect for development
	defer db.Close()

	// ── Store ──────────────────────────────────────────────────────────────────
	store := db.NewStore(db.DB)

	// ── Services ───────────────────────────────────────────────────────────────
	orderSvc := service.NewOrderService(store, nil)
	limitEngine := service.NewLimitOrderEngine(orderSvc)
	orderSvc.SetEngine(limitEngine)

	// ── Limit Order Engine ─────────────────────────────────────────────────────
	if err := limitEngine.LoadOpenOrders(context.Background()); err != nil {
		log.Printf("⚠️  Limit order engine: failed to load open orders: %v", err)
	}
	go limitEngine.StartWebSocketMonitor()
	log.Println("✅ Limit order engine running")

	// ── HTTP Router ────────────────────────────────────────────────────────────
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.Use(
		cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "http://192.168.1.103:3000", "https://trading-platform-nu-seven.vercel.app"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		gin.Logger(),
		gin.Recovery(),
	)

	// ── WebSocket Hub (panic‑safe) ────────────────────────────────────────────
	supportHub := ws.NewSupportHub(store.Queries)

	// ✅ Load JWT secret from environment (set in .env or production env)
	jwtSecret := os.Getenv("JWT_ACCESS_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET is not set")
	}

	// Pass the secret to route registration
	registerRoutes(r, orderSvc, limitEngine, store, supportHub, jwtSecret)

	// ── Server ─────────────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// ── Graceful Shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⏳ Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}
	log.Println("✅ Server stopped cleanly")
}

// registerRoutes wires all route groups onto the engine.
func registerRoutes(
	r *gin.Engine,
	orderSvc *service.OrderService,
	limitEngine *service.LimitOrderEngine,
	store *db.Store,
	supportHub *ws.SupportHub,
	jwtSecret string, // <-- new parameter
) {
	log.Println("✅ Registering routes...")
	log.Printf("   Store: %v", store != nil)

	orderHandler := handlers.NewOrderHandler(orderSvc)

	// ✅ Create the WebSocket handler with the pre-loaded secret
	supportWSHandler := &handlers.SupportWSHandler{
		Hub:       supportHub,
		Queries:   store.Queries,
		JWTSecret: jwtSecret, // this was previously missing
	}

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket endpoint
	r.GET("/ws/support", supportWSHandler.HandleWebSocket)

	publicHandler := handlers.NewPublicHandler(store.Queries)
	r.GET("/public/deposit-address", publicHandler.GetDepositAddress)
	// ── Support REST ───────────────────────────────────────────────────
	supportRest := handlers.NewSupportRestHandler(store.Queries, supportHub)
	support := r.Group("/support")
	support.Use(middleware.AuthMiddleware())
	{
		support.POST("/sessions", supportRest.CreateSession)
		support.POST("/upload/cloudinary", supportRest.UploadToCloudinary)
		support.GET("/sessions/:id/messages", supportRest.GetMessages)
		support.GET("/sessions/open", supportRest.GetOpenSession)
		support.PATCH("/messages/:id", supportRest.UpdateMessage)
		support.DELETE("/messages/:id", supportRest.DeleteMessage)
	}

	// Agent routes (protected)
	agentGroup := r.Group("/agent")
	agentGroup.Use(middleware.AuthMiddleware(), middleware.RequireAgent())
	{
		agentHandler := handlers.NewAgentHandler(store.Queries)
		agentGroup.GET("/conversations", agentHandler.ListConversations)
		agentGroup.GET("/conversations/:id", agentHandler.GetConversation)
		agentGroup.POST("/conversations/:id/assign", agentHandler.AssignConversation)
		agentGroup.POST("/conversations/:id/close", agentHandler.CloseConversation)
	}

	// Auth — rate limited to prevent brute force, no auth required
	auth := r.Group("/auth")
	auth.Use(middleware.RateLimiter())
	{
		auth.POST("/register", handlers.RegisterHandler)
		auth.POST("/signin", handlers.LoginHandler)
		auth.POST("/refresh", handlers.RefreshHandler)
		auth.POST("/logout", handlers.LogoutHandler)
	}

	// Balances — authenticated, no rate limit (polled frequently)
	balances := r.Group("/balances")
	balances.Use(middleware.AuthMiddleware())
	{
		balances.GET("", handlers.ListBalances)
		balances.GET("/:asset", handlers.GetBalance)
	}

	// Predictions — authenticated; polling routes have no rate limit
	predictions := r.Group("/predictions")
	predictions.Use(middleware.AuthMiddleware())
	{
		predictions.POST("/place", middleware.RateLimiter(), handlers.PlacePrediction)
		predictions.GET("/active", handlers.GetActivePredictions)
		predictions.GET("/result/:id", handlers.GetPredictionResult)
		predictions.GET("/history", middleware.RateLimiter(), handlers.GetPredictionHistory)
		predictions.POST("/cancel/:id", middleware.RateLimiter(), handlers.CancelPrediction)
	}

	// Users — authenticated, rate limited
	users := r.Group("/users")
	users.Use(middleware.AuthMiddleware(), middleware.RateLimiter())
	{
		users.GET("/user/:id", handlers.GetUserByIDHandler)
		users.PATCH("/user/:id", handlers.UpdateUserByIDHandler)
	}

	withdrawalSvc := service.NewWithdrawalService(store)

	// User withdrawal request
	userWithdrawalHandler := handlers.NewWithdrawalHandler(withdrawalSvc)
	r.POST("/withdrawals", middleware.AuthMiddleware(), middleware.RateLimiter(), userWithdrawalHandler.RequestWithdrawal)

	// Orders — authenticated, rate limited
	orders := r.Group("/orders")
	orders.Use(middleware.AuthMiddleware(), middleware.RateLimiter())
	{
		orders.POST("", orderHandler.PlaceOrder)
		orders.DELETE("/:id", orderHandler.CancelOrder)
	}

	// Admin routes (protected, admin only)
	adminGroup := r.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		adminHandler := handlers.NewAdminHandler(store.Queries)

		adminGroup.GET("/dashboard", adminHandler.GetAdminDashboard)
		adminGroup.POST("/deposit", adminHandler.AdminDeposit)
		adminGroup.GET("/users", adminHandler.GetUsersHandler)
		adminGroup.GET("/users/agents", adminHandler.GetAgentUsersHandler)
		adminGroup.GET("/users/banned", adminHandler.GetBannedUsersHandler)
		adminGroup.GET("/users/search", adminHandler.SearchUsersHandler)
		adminGroup.PATCH("/users/ban/:id", adminHandler.BanUser)
		adminGroup.PATCH("/users/unban/:id", adminHandler.UnbanUser)
		adminGroup.PATCH("/users/role/:id", adminHandler.UpdateUserRole)

		withdrawalHandler := handlers.NewAdminWithdrawalHandler(store.Queries)
		adminGroup.GET("/withdrawals/search", withdrawalHandler.SearchWithdrawals)
		adminGroup.PATCH("/withdrawals/:id/approve", withdrawalHandler.ApproveWithdrawal)
		adminGroup.PATCH("/withdrawals/:id/reject", withdrawalHandler.RejectWithdrawal)
		adminGroup.GET("/withdrawals/completed", withdrawalHandler.CompletedWithdrawals)
		adminGroup.GET("/withdrawals/pending", withdrawalHandler.PendingWithdrawals)
		adminGroup.GET("/withdrawals/rejected", withdrawalHandler.RejectedWithdrawals)

		adminSettingsHandler := handlers.NewAdminSettingsHandler(store.Queries)
		adminGroup.GET("/settings/deposit-address", adminSettingsHandler.GetDepositAddress)
		adminGroup.PUT("/settings/deposit-address", adminSettingsHandler.UpdateDepositAddress)
		adminGroup.PATCH("/settings/will-profit/:id", adminSettingsHandler.UpdateWillProfitHandler)
	}
}
