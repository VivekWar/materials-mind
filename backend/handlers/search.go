package handlers

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/services"
)

const minSearchQueryRunes = 3

type SearchHandler struct {
	searchSvc services.SearchService
}

func NewSearchHandler(searchSvc services.SearchService) *SearchHandler {
	return &SearchHandler{searchSvc: searchSvc}
}

type SearchRequest struct {
	Query          string `json:"query" binding:"required,min=3,max=5000"`
	IndustryDomain string `json:"industry_domain"`
}

func (h *SearchHandler) HybridSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	if utf8.RuneCountInString(req.Query) < minSearchQueryRunes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is too short"})
		return
	}

	userID := c.GetString("user_id")
	if userID != "" {
		userIDInt, err := strconv.ParseInt(userID, 10, 64)
		if err == nil {
			var msgCount int
			err = db.Pool.QueryRow(c.Request.Context(), "SELECT count(*) FROM messages m JOIN chats c ON m.chat_id = c.id WHERE c.user_id = $1 AND m.created_at >= CURRENT_DATE", userIDInt).Scan(&msgCount)
			if err == nil && msgCount >= 30 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Daily message limit reached (30 max)."})
				return
			}
		}
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Status(http.StatusOK)

	recommendationChan, err := h.searchSvc.ProcessSearchStream(c.Request.Context(), req.Query, req.IndustryDomain)
	if err != nil {
		c.SSEvent("error", "An error occurred while processing your search.")
		c.Writer.Flush()
		return
	}

	for event := range recommendationChan {
		switch event.Type {
		case "structured_result":
			c.SSEvent("structured_result", event.Data)
		case "message":
			c.SSEvent("message", event.Data)
		case "error":
			c.SSEvent("error", event.Data)
		case "done":
			c.SSEvent("done", "true")
			return
		}
		c.Writer.Flush()
	}
}
