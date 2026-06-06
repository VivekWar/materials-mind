package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/handlers"
	"github.com/vivekwar/materialmind/middleware"
	"github.com/vivekwar/materialmind/services"
)

func main() {
	if err := loadEnvFile(); err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// 1. Fail-Fast Infrastructure Initialization
	db.InitPostgres()
	db.InitRedis()
	services.InitGemini()

	// 2. Router Setup
	r := gin.Default()

	// 3. Public Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "MaterialMind API is active"})
	})
	r.POST("/api/auth/mock-login", handlers.MockLogin)

	// 4. Secure Routes (Protected by Auth & Rate Limiting)
	protected := r.Group("/api")
	protected.Use(middleware.RequireAuth())
	protected.Use(middleware.RateLimit())
	{
		// The Hybrid RAG Endpoint
		protected.POST("/search", handlers.HybridSearch)
		protected.POST("/chat/followup", handlers.ChatFollowup)
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
