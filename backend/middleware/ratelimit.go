package middleware

import (
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/db"
)

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		key := "ratelimit:" + c.ClientIP()
		if userID != "" {
			key = "ratelimit:user:" + userID
		}

		count, err := db.RedisClient.Incr(c.Request.Context(), key).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter failed"})
			return
		}

		if count == 1 {
			db.RedisClient.Expire(c.Request.Context(), key, time.Minute)
		}

		if count > 20 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded."})
			return
		}
		c.Next()
	}
}