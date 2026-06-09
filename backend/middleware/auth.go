package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/utils"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""

		// 1. Try to get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		// 2. Fallback to cookie
		if tokenString == "" {
			cookie, err := c.Cookie("session_token")
			if err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No session token provided"})
			return
		}

		userID, err := utils.VerifyToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid session"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}