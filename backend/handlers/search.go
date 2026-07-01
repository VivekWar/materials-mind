package handlers

import (
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materials-mind/backend/services"
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
	if dailyMessageQuotaExceeded(c.Request.Context(), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Daily message limit reached (%d max).", maxMessagesPerDay)})
		return
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
