package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materials-mind/backend/domain"
)

func (h *SearchHandler) ChatFollowup(c *gin.Context) {
	userID := c.GetString("user_id")
	if dailyMessageQuotaExceeded(c.Request.Context(), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Daily message limit reached (%d max).", maxMessagesPerDay)})
		return
	}

	var req domain.FollowUpChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.searchSvc.ProcessFollowup(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "follow-up generation failed"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
