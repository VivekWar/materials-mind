package handlers

import (
	"context"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/utils"
)

func MockLogin(c *gin.Context) {
	mockUserID := "firebase_uid_123"
	mockEmail := "engineer@materialmind.ai"

	query := `INSERT INTO users (id, email) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING;`
	_, err := db.Pool.Exec(context.Background(), query, mockUserID, mockEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	token, err := utils.GenerateToken(mockUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	// Security: HttpOnly set to true prevents XSS token theft
	c.SetCookie("session_token", token, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user_id": mockUserID})
}