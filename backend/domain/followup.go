package domain

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
