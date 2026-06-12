package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/utils"
)

type authUser struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	ChatsUsed    int    `json:"chats_used"`
	MaxChats     int    `json:"max_chats"`
	MessagesUsed int    `json:"messages_used"`
	MaxMessages  int    `json:"max_messages"`
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

	token, err := utils.GenerateToken(authUser.UserID)
	if err != nil {
		log.Printf("Token generation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	// Still issue cookie as a fallback
	isProd := gin.Mode() == gin.ReleaseMode
	if isProd {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}
	c.SetCookie("session_token", token, 3600*24, "/", "", isProd, true)

	frontendOrigin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:5173"
	}

	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head><title>Authenticating...</title></head>
		<body>
			<script>
				if (window.opener) {
					window.opener.postMessage({ type: 'AUTH_SUCCESS', token: '%s' }, '%s');
					window.close();
				} else {
					window.location.href = '%s/chat?token=%s';
				}
			</script>
		</body>
		</html>
	`, token, frontendOrigin, frontendOrigin, token)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func Me(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	var user authUser
	err = db.Pool.QueryRow(
		c.Request.Context(),
		`SELECT 
			id::text, 
			email, 
			COALESCE(full_name, ''),
			(SELECT count(*) FROM chats WHERE user_id = $1 AND created_at >= CURRENT_DATE)::int,
			(SELECT count(*) FROM messages m JOIN chats c ON m.chat_id = c.id WHERE c.user_id = $1 AND m.created_at >= CURRENT_DATE)::int
		 FROM users WHERE id = $1`,
		userIDInt,
	).Scan(&user.UserID, &user.Email, &user.Name, &user.ChatsUsed, &user.MessagesUsed)
	
	if err != nil {
		log.Printf("Error in Me query: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session user not found"})
		return
	}

	user.MaxChats = 10
	user.MaxMessages = 30

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func Logout(c *gin.Context) {
	clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func upsertUser(ctx context.Context, email, name, provider, providerID string) (*authUser, error) {
	var user authUser
	var dbProvider string
	err := db.Pool.QueryRow(
		ctx,
		`
		INSERT INTO users (email, full_name, provider, provider_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET
			full_name = COALESCE(NULLIF(EXCLUDED.full_name, ''), users.full_name)
		RETURNING id::text, email, COALESCE(full_name, ''), provider
		`,
		email,
		name,
		provider,
		providerID,
	).Scan(&user.UserID, &user.Email, &user.Name, &dbProvider)
	if err != nil {
		return nil, err
	}
	if dbProvider != provider {
		return nil, fmt.Errorf("account exists with a different provider")
	}
	return &user, nil
}

func issueSessionCookie(c *gin.Context, userID string) {
	token, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	isProd := gin.Mode() == gin.ReleaseMode
	if isProd {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}
	c.SetCookie("session_token", token, 3600*24, "/", "", isProd, true)
}

func clearSessionCookie(c *gin.Context) {
	isProd := gin.Mode() == gin.ReleaseMode
	if isProd {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}
	c.SetCookie("session_token", "", -1, "/", "", isProd, true)
}
