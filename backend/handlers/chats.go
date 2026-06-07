package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vivekwar/materialmind/services"
)

type ChatHandler struct {
	chatSvc services.ChatService
}

func NewChatHandler(chatSvc services.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc}
}

type CreateChatRequest struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
}

type AddMessageRequest struct {
	SenderRole  string          `json:"sender_role" binding:"required,oneof=user assistant system"`
	Content     json.RawMessage `json:"content" binding:"required"`
	ContentText *string         `json:"content_text"`
	TokensUsed  int32           `json:"tokens_used"`
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chat, err := h.chatSvc.CreateChat(c.Request.Context(), userID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"chat": chat})
}

func (h *ChatHandler) ListChats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	chats, err := h.chatSvc.ListChats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID := c.GetString("user_id")
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}

	messages, err := h.chatSvc.GetMessages(c.Request.Context(), chatID, userID)
	if err != nil {
		if err.Error() == "chat not found or unauthorized" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (h *ChatHandler) AddMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}

	var req AddMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content cannot be empty"})
		return
	}

	msg, err := h.chatSvc.AddMessage(c.Request.Context(), chatID, userID, req.SenderRole, req.Content, req.ContentText, req.TokensUsed)
	if err != nil {
		if err.Error() == "chat not found or unauthorized" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add message"})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

func (h *ChatHandler) ArchiveChat(c *gin.Context) {
	userID := c.GetString("user_id")
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}

	err := h.chatSvc.ArchiveChat(c.Request.Context(), chatID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to archive chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "chat archived"})
}

func parseChatID(c *gin.Context) (int64, bool) {
	chatID, err := strconv.ParseInt(c.Param("chat_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat_id"})
		return 0, false
	}
	return chatID, true
}
