package handlers

import (
	"context"
	"strconv"

	"github.com/vivekwar/materials-mind/backend/db"
)

// Daily usage caps, shared across chat creation, search, and follow-up
// handlers so the limits stay consistent and are declared in exactly one
// place. Values are intentionally small constants (not env-configurable)
// since they exist to bound Gemini API cost, not to be tuned per-deployment.
const (
	maxChatsPerDay    = 10
	maxMessagesPerDay = 30
)

// dailyMessageQuotaExceeded reports whether the authenticated user has hit
// their daily message cap. It backs both /search and /chat/followup so a
// user can't bypass the limit by only ever sending follow-up messages after
// their first search of the day.
func dailyMessageQuotaExceeded(ctx context.Context, userID string) bool {
	if userID == "" {
		return false
	}
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false
	}
	var msgCount int
	err = db.Pool.QueryRow(ctx,
		"SELECT count(*) FROM messages m JOIN chats c ON m.chat_id = c.id WHERE c.user_id = $1 AND m.created_at >= CURRENT_DATE",
		userIDInt,
	).Scan(&msgCount)
	return err == nil && msgCount >= maxMessagesPerDay
}
