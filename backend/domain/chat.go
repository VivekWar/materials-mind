package domain

import (
	"encoding/json"
	"time"
)

type Chat struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID          int64           `json:"id"`
	ChatID      int64           `json:"chat_id"`
	SenderRole  string          `json:"sender_role"`
	SenderID    *string         `json:"sender_id"`
	Content     json.RawMessage `json:"content"`
	ContentText *string         `json:"content_text"`
	TokensUsed  int32           `json:"tokens_used"`
	CreatedAt   time.Time       `json:"created_at"`
}
