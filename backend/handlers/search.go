package handlers

import (
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
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
	Query          string `json:"query" binding:"required,min=3,max=500"`
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

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Status(http.StatusOK)

	recommendation, err := h.searchSvc.ProcessSearch(c.Request.Context(), req.Query, req.IndustryDomain)
	if err != nil {
		if err.Error() == "no matching materials found" {
			c.SSEvent("message", "I could not find matching materials for this query. Please add more constraints like strength, weight, temperature, or budget.")
			c.SSEvent("done", "true")
			c.Writer.Flush()
			return
		}
		c.SSEvent("error", "An error occurred while processing your search.")
		c.Writer.Flush()
		return
	}

	structuredJSON, err := json.Marshal(recommendation)
	if err != nil {
		c.SSEvent("error", "Failed to format generated response")
		c.Writer.Flush()
		return
	}

	c.SSEvent("structured_result", string(structuredJSON))
	c.SSEvent("message", recommendation.Report)
	c.SSEvent("done", "true")
	c.Writer.Flush()
}
