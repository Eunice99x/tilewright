package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSignupLogin(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := newTestService(f, &clock{time.Now()}, Paddle{})

	if _, _, err := s.Signup(ctx, testIP, "jane@example.com", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("short password: got %v", err)
	}
	if _, _, err := s.Signup(ctx, testIP, "not an email", "longenough"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("bad email: got %v", err)
	}
	token, u, err := s.Signup(ctx, testIP, "  Jane@Example.com ", "longenough")
	if err != nil || token == "" {
		t.Fatalf("signup: %v", err)
	}
	if u.Email != "jane@example.com" || u.Plan != "free" {
		t.Errorf("user = %+v", u)
	}
	if stored := f.users[u.ID].PasswordHash; stored == "longenough" || stored == "" {
		t.Error("password stored in plain text")
	}
	if _, ok := f.sessions[token]; ok {
		t.Error("session token stored in plain text")
	}
	if _, _, err := s.Signup(ctx, testIP, "jane@example.com", "longenough"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate signup: got %v", err)
	}

	if got, err := s.UserFromSession(ctx, token); err != nil || got.ID != u.ID {
		t.Errorf("session lookup: %v %v", got, err)
	}
	if _, _, err := s.Login(ctx, testIP, "jane@example.com", "wrongpass"); !errors.Is(err, ErrBadLogin) {
		t.Errorf("wrong password: got %v", err)
	}
	if _, _, err := s.Login(ctx, testIP, "nobody@example.com", "longenough"); !errors.Is(err, ErrBadLogin) {
		t.Errorf("unknown email: got %v", err)
	}
	token2, _, err := s.Login(ctx, testIP, "JANE@example.com", "longenough")
	if err != nil || token2 == token {
		t.Fatalf("login: %v", err)
	}
	if err := s.Logout(ctx, token2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserFromSession(ctx, token2); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("after logout: got %v", err)
	}
	if _, err := s.UserFromSession(ctx, ""); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("empty token: got %v", err)
	}
}
