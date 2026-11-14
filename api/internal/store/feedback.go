package store

import (
	"context"
	"time"
)

// Feedback is one message from the feedback box.
type Feedback struct {
	ID        string    `db:"id" json:"id"`
	UserID    *string   `db:"user_id" json:"user_id"` // nil for visitors who weren't signed in
	Email     string    `db:"email" json:"email"`     // "" when they left none
	Message   string    `db:"message" json:"message"`
	Page      string    `db:"page" json:"page"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// SaveFeedback stores one message. userID "" means a visitor without an account.
func (s *Store) SaveFeedback(ctx context.Context, userID, email, message, page string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO feedback (user_id, email, message, page) VALUES (NULLIF($1, '')::uuid, $2, $3, $4)`,
		userID, email, message, page)
	return err
}

// AdminFeedback returns a page of messages, newest first, and how many there are in all.
func (s *Store) AdminFeedback(ctx context.Context, limit, offset int) ([]Feedback, int, error) {
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT count(*) FROM feedback`); err != nil {
		return nil, 0, err
	}
	out := []Feedback{}
	err := s.db.SelectContext(ctx, &out, `SELECT id, user_id, email, message, page, created_at FROM feedback
		ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	return out, total, err
}
