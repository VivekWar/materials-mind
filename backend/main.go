package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/handlers"
	"github.com/vivekwar/materialmind/middleware"
	"github.com/vivekwar/materialmind/repositories"
	"github.com/vivekwar/materialmind/services"
	"github.com/vivekwar/materialmind/utils"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
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

	chatHandler := handlers.NewChatHandler(chatSvc, searchSvc)
	searchHandler := handlers.NewSearchHandler(searchSvc)

	// 1.5 Goth OAuth Initialization
	sessionSecret := os.Getenv("JWT_SECRET")
	if sessionSecret == "" {
		sessionSecret = "fallback-secret-for-dev"
	}
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.MaxAge(86400 * 30) // 30 days
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	// Use Secure=true in production (GIN_MODE=release) so cookies are sent over HTTPS.
	// In local dev (GIN_MODE unset or "debug"), keep it false so HTTP works.
	isProduction := os.Getenv("GIN_MODE") == "release"
	store.Options.Secure = isProduction
	if isProduction {
		store.Options.SameSite = http.SameSiteNoneMode // Required for cross-site cookies (HF Space → CF Pages)
	} else {
		store.Options.SameSite = http.SameSiteLaxMode
	}
	gothic.Store = store

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), backendURL+"/api/auth/google/callback"),
	)

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
		
		allowed := false
		// Allow configured origin, any localhost, or any cloudflare pages preview/prod deployment
		if origin == os.Getenv("FRONTEND_ORIGIN") || 
		   strings.HasPrefix(origin, "http://localhost:") || 
		   strings.HasSuffix(origin, ".pages.dev") {
			allowed = true
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
