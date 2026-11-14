package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"saas-starter/api/internal/store"
)

func TestSendFeedback(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := New(f, Config{})

	for name, in := range map[string]FeedbackInput{
		"empty":      {},
		"too short":  {Message: " hi "},
		"only NUL":   {Message: "\x00\x00\x00\x00"},
		"too long":   {Message: strings.Repeat("é", feedbackMessageMax+1)},
		"long email": {Message: "fine", Email: strings.Repeat("a", 250) + "@x.io"},
		"long page":  {Message: "fine", Page: "/" + strings.Repeat("p", feedbackPageMax)},
	} {
		if err := s.SendFeedback(ctx, "203.0.113.1", nil, in); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(f.feedback) != 0 {
		t.Fatalf("invalid messages were stored: %+v", f.feedback)
	}

	if err := s.SendFeedback(ctx, "203.0.113.1", nil, FeedbackInput{Message: "  a\x00bc\xff ", Email: " me@example.com ", Page: "/pricing"}); err != nil {
		t.Fatal(err)
	}
	if got := f.feedback[0]; got.Message != "abc" || got.Email != "me@example.com" || got.Page != "/pricing" || got.UserID != nil {
		t.Errorf("visitor row: %+v", got)
	}
	if err := s.SendFeedback(ctx, "203.0.113.1", nil, FeedbackInput{Message: strings.Repeat("é", feedbackMessageMax)}); err != nil {
		t.Errorf("longest message: %v", err)
	}

	// signed in: the account's own address wins over whatever was typed
	u := store.User{ID: "00000000-0000-4000-8000-000000000042", Email: "owner@example.com"}
	if err := s.SendFeedback(ctx, "203.0.113.2", &u, FeedbackInput{Message: "a dark mode please", Email: "someone@else.com"}); err != nil {
		t.Fatal(err)
	}
	if got := f.feedback[2]; got.UserID == nil || *got.UserID != u.ID || got.Email != u.Email {
		t.Errorf("account row: %+v", got)
	}
	// a typed address is checked and lowercased
	if err := s.SendFeedback(ctx, "203.0.113.4", nil, FeedbackInput{Message: "hello", Email: "not an email"}); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad typed email: %v", err)
	}
	if err := s.SendFeedback(ctx, "203.0.113.4", nil, FeedbackInput{Message: "hello", Email: "x@" + guestDomain}); !errors.Is(err, ErrInvalid) {
		t.Errorf("guest typed email: %v", err)
	}
	// a guest's address is made up, so the one they typed is kept
	g := store.User{ID: "00000000-0000-4000-8000-000000000043", Email: "guest-abc@" + guestDomain}
	if err := s.SendFeedback(ctx, "203.0.113.3", &g, FeedbackInput{Message: "more tiles", Email: "Real@Example.com"}); err != nil {
		t.Fatal(err)
	}
	if got := f.feedback[3]; got.UserID == nil || *got.UserID != g.ID || got.Email != "real@example.com" {
		t.Errorf("guest row: %+v", got)
	}

	// per IP: the limit holds whether signed in or not, and other addresses aren't affected
	for i := 0; i < feedbackIPLimit; i++ {
		s.SendFeedback(ctx, "198.51.100.7", nil, FeedbackInput{Message: "spam spam"})
	}
	if err := s.SendFeedback(ctx, "198.51.100.7", &u, FeedbackInput{Message: "spam spam"}); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("over the limit: %v", err)
	}
	if err := s.SendFeedback(ctx, "198.51.100.8", nil, FeedbackInput{Message: "a real one"}); err != nil {
		t.Errorf("another address: %v", err)
	}
}

func TestAdminFeedback(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := New(f, Config{})

	if _, err := s.AdminFeedback(ctx, store.User{ID: "someone"}, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("not an admin: %v", err)
	}
	p, err := s.AdminFeedback(ctx, store.User{IsAdmin: true}, -3)
	if err != nil {
		t.Fatal(err)
	}
	if p.Feedback == nil || p.Offset != 0 || f.adminOffset != 0 || f.adminLimit != adminFeedbackPageSz || p.PageSize != adminFeedbackPageSz {
		t.Errorf("empty page: %+v, limit %d offset %d", p, f.adminLimit, f.adminOffset)
	}
	s.SendFeedback(ctx, "203.0.113.1", nil, FeedbackInput{Message: "first"})
	s.SendFeedback(ctx, "203.0.113.1", nil, FeedbackInput{Message: "second"})
	p, _ = s.AdminFeedback(ctx, store.User{IsAdmin: true}, 0)
	if len(p.Feedback) != 2 || p.Feedback[0].Message != "second" || p.Total != 2 {
		t.Errorf("newest first: %+v", p)
	}
}
