package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/domain"
)

func (h *SearchHandler) ChatFollowup(c *gin.Context) {
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
