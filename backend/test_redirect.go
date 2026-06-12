package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/callback", func(c *gin.Context) {
		token := "fake_token_123"
		frontendOrigin := strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/")
		if frontendOrigin == "" {
			frontendOrigin = "http://localhost:5173"
		}
		c.Redirect(http.StatusFound, frontendOrigin+"/chat?token="+token)
	})
	
	req, _ := http.NewRequest("GET", "/callback", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	fmt.Printf("Redirected to: %s\n", w.Header().Get("Location"))
}
