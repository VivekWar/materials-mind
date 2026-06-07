package handlers

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/utils"
)

type authUser struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
}

func GothLogin(c *gin.Context) {
	provider := c.Param("provider")
	req := c.Request.WithContext(context.WithValue(c.Request.Context(), gothic.ProviderParamKey, provider))
	gothic.BeginAuthHandler(c.Writer, req)
}

func GothCallback(c *gin.Context) {
	provider := c.Param("provider")
	req := c.Request.WithContext(context.WithValue(c.Request.Context(), gothic.ProviderParamKey, provider))

	log.Printf("GothCallback received cookies: %v", c.Request.Cookies())

	gothUser, err := gothic.CompleteUserAuth(c.Writer, req)
	if err != nil {
		log.Printf("GothCallback error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OAuth authentication failed: " + err.Error()})
		return
	}

	authUser, err := upsertUser(c.Request.Context(), gothUser.Email, gothUser.Name, gothUser.Provider, gothUser.UserID)
	if err != nil {
		log.Printf("Database error in upsertUser: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	issueSessionCookie(c, authUser.UserID)

	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:5173"
	}
	c.Redirect(http.StatusFound, frontendOrigin+"/chat")
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
