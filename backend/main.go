package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vivekwar/materials-mind/backend/db"
	"github.com/vivekwar/materials-mind/backend/handlers"
	"github.com/vivekwar/materials-mind/backend/middleware"
	"github.com/vivekwar/materials-mind/backend/repositories"
	"github.com/vivekwar/materials-mind/backend/services"
	"github.com/vivekwar/materials-mind/backend/utils"
)

func main() {
	if err := loadEnvFile(); err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// 1. Fail-Fast Infrastructure Initialization
	utils.InitJWT()
	if err := utils.InitOAuth(); err != nil {
		log.Fatalf("OAuth Initialization Failed: %v", err)
	}
	db.InitPostgres()
	db.InitRedis()
	services.InitGemini()

	// Dependency Injection
	chatRepo := repositories.NewChatRepository(db.Pool)
	materialRepo := repositories.NewMaterialRepository(db.Pool)

	chatSvc := services.NewChatService(chatRepo)
	searchSvc := services.NewSearchService(materialRepo)

	chatHandler := handlers.NewChatHandler(chatSvc, searchSvc)
	searchHandler := handlers.NewSearchHandler(searchSvc)

	// 2. Router Setup
	r := gin.Default()
	r.Use(corsMiddleware())

	// 3. Public Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "MaterialMind API is active"})
	})

	// Detailed health check for load balancers and monitoring
	r.GET("/healthz", func(c *gin.Context) {
		dbOK := db.Pool != nil && db.Pool.Ping(c.Request.Context()) == nil
		redisOK := db.RedisClient != nil && db.RedisClient.Ping(c.Request.Context()).Err() == nil
		status := "healthy"
		httpCode := 200
		if !dbOK || !redisOK {
			status = "degraded"
			httpCode = 503
		}
		c.JSON(httpCode, gin.H{
			"status":    status,
			"db":        boolStatus(dbOK),
			"redis":     boolStatus(redisOK),
			"version":   "1.0.0",
			"gin_mode":  os.Getenv("GIN_MODE"),
		})
	})
	r.GET("/api/auth/:provider/login", handlers.GothLogin)
	r.GET("/api/auth/:provider/callback", handlers.GothCallback)

	// 4. Secure Routes (Protected by Auth & Rate Limiting)
	protected := r.Group("/api")
	protected.Use(middleware.RequireAuth())
	protected.Use(middleware.RateLimit())
	{
		protected.GET("/auth/me", handlers.Me)
		protected.POST("/auth/logout", handlers.Logout)

		// The Hybrid RAG Endpoint
		protected.POST("/search", searchHandler.HybridSearch)
		protected.POST("/chat/followup", searchHandler.ChatFollowup)

		// Chat persistence and message management
		protected.POST("/chat/create", chatHandler.CreateChat)
		protected.GET("/chat/list", chatHandler.ListChats)
		protected.GET("/chat/list/:user_id", chatHandler.ListChats)
		protected.GET("/chat/:chat_id/messages", chatHandler.GetMessages)
		protected.POST("/chat/:chat_id/messages", chatHandler.AddMessage)
		protected.POST("/chat/:chat_id/archive", chatHandler.ArchiveChat)
		protected.POST("/chat/:chat_id/title/generate", chatHandler.GenerateTitle)
	}

	// 5. Boot Server (graceful shutdown so in-flight requests/LLM calls survive
	// container rotation on Hugging Face Spaces / Kubernetes SIGTERM signals)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("🚀 MaterialMind API running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Critical: Server crashed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received, draining in-flight requests...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Critical: Graceful shutdown failed: %v", err)
	}
	log.Println("Server exited cleanly")
}

func loadEnvFile() error {
	if err := godotenv.Load("../.env"); err == nil {
		return nil
	}

	if err := godotenv.Load(".env"); err == nil {
		return nil
	}

	return os.ErrNotExist
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		allowed := false
		configuredOrigins := strings.Split(os.Getenv("FRONTEND_ORIGIN"), ",")
		isProduction := os.Getenv("GIN_MODE") == "release"

		// 1. Strict Production Rule: Only allow exact matched origins
		if origin != "" {
			for _, co := range configuredOrigins {
				if origin == strings.TrimRight(strings.TrimSpace(co), "/") {
					allowed = true
					break
				}
			}
		}

		// 2. Development/Preview Rules: Only active when NOT in production
		if !isProduction && origin != "" {
			if strings.HasPrefix(origin, "http://localhost:") || 
			   strings.HasPrefix(origin, "http://127.0.0.1:") || 
			   strings.HasSuffix(origin, ".pages.dev") {
				allowed = true
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func boolStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "unreachable"
}
