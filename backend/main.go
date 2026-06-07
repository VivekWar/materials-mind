package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/handlers"
	"github.com/vivekwar/materialmind/middleware"
	"github.com/vivekwar/materialmind/repositories"
	"github.com/vivekwar/materialmind/services"
	"github.com/vivekwar/materialmind/utils"
)

func main() {
	if err := loadEnvFile(); err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// 1. Fail-Fast Infrastructure Initialization
	utils.InitJWT()
	db.InitPostgres()
	db.InitRedis()
	services.InitGemini()

	// Dependency Injection
	chatRepo := repositories.NewChatRepository(db.Pool)
	materialRepo := repositories.NewMaterialRepository(db.Pool)

	chatSvc := services.NewChatService(chatRepo)
	searchSvc := services.NewSearchService(materialRepo)

	chatHandler := handlers.NewChatHandler(chatSvc)
	searchHandler := handlers.NewSearchHandler(searchSvc)

	// 2. Router Setup
	r := gin.Default()
	r.Use(corsMiddleware())

	// 3. Public Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "MaterialMind API is active"})
	})
	r.POST("/api/auth/mock-login", handlers.MockLogin)
	r.POST("/api/auth/google", handlers.GoogleLogin)

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
	}

	// 5. Boot Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 MaterialMind API running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Critical: Server crashed: %v", err)
	}
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
		allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:5173"
		}

		if origin == allowedOrigin {
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
