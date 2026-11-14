package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"saas-starter/api/internal/store"
)

const (
	feedbackIPLimit     = 10 // messages per address per hour
	feedbackMessageMin  = 3
	feedbackMessageMax  = 2000
	feedbackEmailMax    = 254
	feedbackPageMax     = 200
	adminFeedbackPageSz = 50
)

// FeedbackInput is POST /api/feedback.
type FeedbackInput struct {
	Message string `json:"message"`
	Email   string `json:"email"`
	Page    string `json:"page"`
}

// AdminFeedbackPage is GET /api/admin/feedback.
type AdminFeedbackPage struct {
	Feedback []store.Feedback `json:"feedback"`
	Total    int              `json:"total"`
	Offset   int              `json:"offset"`
	PageSize int              `json:"page_size"`
}

// cleanText drops what Postgres refuses in text (NUL, broken UTF-8) and the spaces around it.
func cleanText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\x00", ""))
}

// SendFeedback stores a message from the feedback box. u is nil for visitors; a signed-in account's own address
// replaces whatever email was typed, except a guest's made-up one. Limited per IP, signed in or not.
func (s *Service) SendFeedback(ctx context.Context, ip string, u *store.User, in FeedbackInput) error {
	msg, email, page := cleanText(in.Message), cleanText(in.Email), cleanText(in.Page)
	if n := utf8.RuneCountInString(msg); n < feedbackMessageMin || n > feedbackMessageMax {
		return ErrInvalid
	}
	if utf8.RuneCountInString(email) > feedbackEmailMax || utf8.RuneCountInString(page) > feedbackPageMax {
		return ErrInvalid
	}
	// a typed address must be a real one; a signed-in account's own replaces it below
	if email != "" {
		email = strings.ToLower(email)
		if !validEmail(email) {
			return ErrInvalid
		}
	}
	userID := ""
	if u != nil {
		userID = u.ID
		if !isGuest(u.Email) {
			email = u.Email
		}
	}
	if !s.feedbackByIP.Hit(ip, s.now()) {
		return ErrTooManyAttempts
	}
	return s.store.SaveFeedback(ctx, userID, email, msg, page)
}

// AdminFeedback lists messages, newest first. Not found for non-admins, like AdminUsers.
func (s *Service) AdminFeedback(ctx context.Context, u store.User, offset int) (AdminFeedbackPage, error) {
	if !u.IsAdmin {
		return AdminFeedbackPage{}, ErrNotFound
	}
	offset = max(offset, 0)
	rows, total, err := s.store.AdminFeedback(ctx, adminFeedbackPageSz, offset)
	if err != nil {
		return AdminFeedbackPage{}, err
	}
	return AdminFeedbackPage{Feedback: rows, Total: total, Offset: offset, PageSize: adminFeedbackPageSz}, nil
}
