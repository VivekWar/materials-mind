package utils

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func InitOAuth() error {
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		// Fallback to JWT_SECRET to not break existing local environments immediately
		sessionSecret = os.Getenv("JWT_SECRET")
		if sessionSecret == "" {
			return fmt.Errorf("Critical: SESSION_SECRET (or JWT_SECRET) environment variable is required for OAuth")
		}
	}

	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.MaxAge(86400 * 30) // 30 days
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	
	isProduction := os.Getenv("GIN_MODE") == "release"
	store.Options.Secure = isProduction
	if isProduction {
		store.Options.SameSite = http.SameSiteNoneMode 
	} else {
		store.Options.SameSite = http.SameSiteLaxMode
	}
	gothic.Store = store

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return fmt.Errorf("Critical: GOOGLE_CLIENT_ID environment variable is required for OAuth")
	}

	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientSecret == "" {
		return fmt.Errorf("Critical: GOOGLE_CLIENT_SECRET environment variable is required for OAuth")
	}

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		return fmt.Errorf("Critical: BACKEND_URL environment variable is required for OAuth callbacks")
	}
	
	googleProvider := google.New(clientID, clientSecret, backendURL+"/api/auth/google/callback")
	googleProvider.SetPrompt("select_account")

	goth.UseProviders(googleProvider)

	return nil
}
