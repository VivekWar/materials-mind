package repositories

import (
	"context"
	"encoding/json"

	"github.com/vivekwar/materialmind/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, userID, title string) (*domain.Chat, error)
	ListChats(ctx context.Context, userID string) ([]domain.Chat, error)
	GetChat(ctx context.Context, chatID int64, userID string) (*domain.Chat, error)
	ArchiveChat(ctx context.Context, chatID int64, userID string) error
	
	GetMessages(ctx context.Context, chatID int64, userID string) ([]domain.Message, error)
	AddMessage(ctx context.Context, chatID int64, senderRole string, senderID *string, content json.RawMessage, contentText *string, tokensUsed int32) (*domain.Message, error)
	UpdateChatTitle(ctx context.Context, chatID int64, userID string, title string) error
}

type chatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) ChatRepository {
	return &chatRepository{pool: pool}
}

func (r *chatRepository) CreateChat(ctx context.Context, userID, title string) (*domain.Chat, error) {
	var chat domain.Chat
	err := r.pool.QueryRow(
		ctx,
		`
		INSERT INTO chats (user_id, title, is_active)
		VALUES ($1, $2, true)
		RETURNING id, user_id, title, is_active, created_at, updated_at
		`,
		userID,
		title,
	).Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.IsActive, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) ListChats(ctx context.Context, userID string) ([]domain.Chat, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT id, user_id, title, is_active, created_at, updated_at
		FROM chats
		WHERE user_id = $1 AND is_active = true
		ORDER BY updated_at DESC
		`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []domain.Chat
	for rows.Next() {
		var chat domain.Chat
		if err := rows.Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.IsActive, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (r *chatRepository) GetChat(ctx context.Context, chatID int64, userID string) (*domain.Chat, error) {
	var chat domain.Chat
	err := r.pool.QueryRow(
		ctx,
		`SELECT id, user_id, title, is_active, created_at, updated_at FROM chats WHERE id = $1 AND user_id = $2 AND is_active = true`,
		chatID,
		userID,
	).Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.IsActive, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *chatRepository) ArchiveChat(ctx context.Context, chatID int64, userID string) error {
	cmd, err := r.pool.Exec(
		ctx,
		`
		UPDATE chats
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		`,
		chatID,
		userID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *chatRepository) GetMessages(ctx context.Context, chatID int64, userID string) ([]domain.Message, error) {
	rows, err := r.pool.Query(
		ctx,
		`
		SELECT m.id, m.chat_id, m.sender_role, m.sender_id, m.content, m.content_text, m.tokens_used, m.created_at
		FROM messages m
		JOIN chats c ON m.chat_id = c.id
		WHERE m.chat_id = $1 AND c.user_id = $2
		ORDER BY m.created_at ASC, m.id ASC
		`,
		chatID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderRole, &msg.SenderID, &msg.Content, &msg.ContentText, &msg.TokensUsed, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *chatRepository) AddMessage(ctx context.Context, chatID int64, senderRole string, senderID *string, content json.RawMessage, contentText *string, tokensUsed int32) (*domain.Message, error) {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var msg domain.Message
	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO messages (chat_id, sender_role, sender_id, content, content_text, tokens_used)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, chat_id, sender_role, sender_id, content, content_text, tokens_used, created_at
		`,
		chatID,
		senderRole,
		senderID,
		string(content),
		contentText,
		tokensUsed,
	).Scan(&msg.ID, &msg.ChatID, &msg.SenderRole, &msg.SenderID, &msg.Content, &msg.ContentText, &msg.TokensUsed, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE chats SET updated_at = NOW() WHERE id = $1`, chatID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (r *chatRepository) UpdateChatTitle(ctx context.Context, chatID int64, userID string, title string) error {
	tag, err := r.pool.Exec(
		ctx,
		`UPDATE chats SET title = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`,
		title,
		chatID,
		userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
