package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/utils"
)

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("session_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No session cookie"})
			return
		}

		userID, err := utils.VerifyToken(cookie)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid session"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}