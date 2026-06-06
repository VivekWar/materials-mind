package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/vivekwar/materialmind/services"
)

type ChatTurn struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type FollowUpChatRequest struct {
	Message            string     `json:"message" binding:"required,min=1,max=1000"`
	History            []ChatTurn `json:"history"`
	InitialReport      string     `json:"initial_report"`
	TopRecommendations []string   `json:"top_recommendations"`
}

type FollowUpChatResponse struct {
	Reply      string `json:"reply"`
	TokensUsed int    `json:"tokens_used"`
}

func ChatFollowup(c *gin.Context) {
	var req FollowUpChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if services.GeminiClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gemini client is not initialized"})
		return
	}

	model := services.GeminiClient.GenerativeModel(services.GenerativeModelName)
	prompt := buildFollowUpPrompt(req)

	resp, err := model.GenerateContent(c.Request.Context(), genai.Text(prompt))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "follow-up generation failed"})
		return
	}

	text, err := extractModelText(resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty follow-up response"})
		return
	}

	c.JSON(http.StatusOK, FollowUpChatResponse{
		Reply:      strings.TrimSpace(text),
		TokensUsed: 0,
	})
}

func buildFollowUpPrompt(req FollowUpChatRequest) string {
	var historyBuilder strings.Builder
	for _, turn := range req.History {
		role := strings.ToLower(strings.TrimSpace(turn.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		historyBuilder.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(role), content))
	}

	var topRecommendations strings.Builder
	for _, item := range req.TopRecommendations {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		topRecommendations.WriteString("- ")
		topRecommendations.WriteString(item)
		topRecommendations.WriteString("\n")
	}

	return fmt.Sprintf(`You are a concise materials engineering assistant.

Conversation history:
%s

Initial engineering report:
%s

Top recommendations:
%s

User follow-up:
%s

Rules:
- Answer only the user's follow-up question.
- Stay grounded in the report and recommendation list above.
- Do not invent new material properties.
- Keep the response clear, practical, and short enough for a product chat UI.`, historyBuilder.String(), strings.TrimSpace(req.InitialReport), topRecommendations.String(), strings.TrimSpace(req.Message))
}
