package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vivekwar/materials-mind/backend/db"
	"github.com/vivekwar/materials-mind/backend/domain"
	"github.com/vivekwar/materials-mind/backend/repositories"
)

type ChatService interface {
	CreateChat(ctx context.Context, userID, title string) (*domain.Chat, error)
	ListChats(ctx context.Context, userID string) ([]domain.Chat, error)
	GetMessages(ctx context.Context, chatID int64, userID string) ([]domain.Message, error)
	AddMessage(ctx context.Context, chatID int64, userID string, senderRole string, content json.RawMessage, contentText *string, tokensUsed int32) (*domain.Message, error)
	ArchiveChat(ctx context.Context, chatID int64, userID string) error
	UpdateChatTitle(ctx context.Context, chatID int64, userID string, title string) error
}

type chatService struct {
	repo repositories.ChatRepository
}

func NewChatService(repo repositories.ChatRepository) ChatService {
	return &chatService{repo: repo}
}

func (s *chatService) CreateChat(ctx context.Context, userID, title string) (*domain.Chat, error) {
	return s.repo.CreateChat(ctx, userID, title)
}

func (s *chatService) ListChats(ctx context.Context, userID string) ([]domain.Chat, error) {
	return s.repo.ListChats(ctx, userID)
}

func (s *chatService) GetMessages(ctx context.Context, chatID int64, userID string) ([]domain.Message, error) {
	// Verify ownership
	_, err := s.repo.GetChat(ctx, chatID, userID)
	if err != nil {
		return nil, errors.New("chat not found or unauthorized")
	}

	cacheKey := fmt.Sprintf("chat:%d:messages", chatID)
	if db.RedisClient != nil {
		if cached, err := db.RedisClient.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
			var msgs []domain.Message
			if err := json.Unmarshal([]byte(cached), &msgs); err == nil {
				return msgs, nil
			}
		}
	}

	msgs, err := s.repo.GetMessages(ctx, chatID, userID)
	if err == nil && db.RedisClient != nil {
		if bytes, err := json.Marshal(msgs); err == nil {
			db.RedisClient.Set(ctx, cacheKey, bytes, 1*time.Hour)
		}
	}
	return msgs, err
}

func (s *chatService) AddMessage(ctx context.Context, chatID int64, userID string, senderRole string, content json.RawMessage, contentText *string, tokensUsed int32) (*domain.Message, error) {
	// Verify ownership
	_, err := s.repo.GetChat(ctx, chatID, userID)
	if err != nil {
		return nil, errors.New("chat not found or unauthorized")
	}

	var senderID *string
	if senderRole == "user" {
		senderID = &userID
	}

	msg, err := s.repo.AddMessage(ctx, chatID, senderRole, senderID, content, contentText, tokensUsed)
	if err == nil && db.RedisClient != nil {
		cacheKey := fmt.Sprintf("chat:%d:messages", chatID)
		db.RedisClient.Del(ctx, cacheKey)
	}
	return msg, err
}

func (s *chatService) ArchiveChat(ctx context.Context, chatID int64, userID string) error {
	return s.repo.ArchiveChat(ctx, chatID, userID)
}

func (s *chatService) UpdateChatTitle(ctx context.Context, chatID int64, userID string, title string) error {
	return s.repo.UpdateChatTitle(ctx, chatID, userID, title)
}
