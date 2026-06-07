package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/utils"
	"google.golang.org/api/idtoken"
)

type googleLoginRequest struct {
	Credential string `json:"credential" binding:"required"`
}

type authUser struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
}

func MockLogin(c *gin.Context) {
	user, err := upsertUser(c.Request.Context(), "engineer@materialmind.ai", "MaterialMind Engineer", "mock", "firebase_uid_123")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	issueSessionCookie(c, user.UserID)
	c.JSON(http.StatusOK, gin.H{"message": "login successful", "user": user})
}

func GoogleLogin(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "GOOGLE_CLIENT_ID is not configured"})
		return
	}

	var req googleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Google credential"})
		return
	}

	payload, err := idtoken.Validate(c.Request.Context(), req.Credential, clientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Google credential"})
		return
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	sub, _ := payload.Claims["sub"].(string)
	if email == "" || sub == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Google credential is missing profile claims"})
		return
	}

	user, err := upsertUser(c.Request.Context(), email, name, "google", sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	issueSessionCookie(c, user.UserID)
	c.JSON(http.StatusOK, gin.H{"message": "login successful", "user": user})
}

func Me(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var user authUser
	err := db.Pool.QueryRow(
		c.Request.Context(),
		`SELECT id::text, email, COALESCE(full_name, '') FROM users WHERE id::text = $1`,
		userID,
	).Scan(&user.UserID, &user.Email, &user.Name)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func Logout(c *gin.Context) {
	clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func upsertUser(ctx context.Context, email, name, provider, providerID string) (*authUser, error) {
	var user authUser
	err := db.Pool.QueryRow(
		ctx,
		`
		INSERT INTO users (email, full_name, provider, provider_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET
			full_name = COALESCE(NULLIF(EXCLUDED.full_name, ''), users.full_name),
			provider = EXCLUDED.provider,
			provider_id = EXCLUDED.provider_id
		RETURNING id::text, email, COALESCE(full_name, '')
		`,
		email,
		name,
		provider,
		providerID,
	).Scan(&user.UserID, &user.Email, &user.Name)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func issueSessionCookie(c *gin.Context, userID string) {
	token, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session_token", token, 3600*24, "/", "", true, true)
}

func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("session_token", "", -1, "/", "", true, true)
}
