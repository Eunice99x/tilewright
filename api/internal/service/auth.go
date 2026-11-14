package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"saas-starter/api/internal/store"
)

const sessionLength = 30 * 24 * time.Hour

var dummyHash, _ = hashPassword("not-a-real-password")

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`

	EmailVerified bool `json:"email_verified"`
	IsAdmin       bool `json:"is_admin"`

	Name  string `json:"name"`
	Guest bool   `json:"guest"`
}

func userView(u store.User) User {
	name := u.DisplayName
	if name == "" {
		name = displayName(u.Email)
	}
	return User{ID: u.ID, Email: u.Email, Plan: currentPlan(u).ID, CreatedAt: u.CreatedAt, EmailVerified: u.EmailVerifiedAt != nil, IsAdmin: u.IsAdmin,
		Name: name, Guest: isGuest(u.Email)}
}

// validEmail also refuses the guest domain: nobody may sign up, sign in with OAuth or reset as a guest address.
func validEmail(email string) bool {
	a, err := mail.ParseAddress(email)
	return err == nil && a.Address == email && len(email) <= 254 && !isGuest(email)
}

func (s *Service) Signup(ctx context.Context, ip, email, password string) (token string, u User, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return "", u, ErrInvalidEmail
	}
	if len(password) < 8 || len(password) > 72 {
		return "", u, ErrWeakPassword
	}
	if !s.signupsByIP.Hit(ip, s.now()) {
		return "", u, ErrTooManyAttempts
	}
	hash, err := hashPassword(password)
	if err != nil {
		return "", u, err
	}
	su, err := s.store.CreateUser(ctx, email, hash)
	if err != nil {
		return "", u, err
	}
	if err := s.sendVerification(ctx, su); err != nil {
		slog.Warn("send verification email", "err", err)
	}
	token, err = s.newSession(ctx, su.ID)
	return token, userView(su), err
}

func (s *Service) Login(ctx context.Context, ip, email, password string) (string, User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	now := s.now()
	if !s.loginsByIP.Hit(ip, now) {
		return "", User{}, ErrTooManyAttempts
	}
	if len(email) > 254 {
		return "", User{}, ErrBadLogin
	}
	if !s.loginsByEmail.Hit(email, now) {
		return "", User{}, ErrTooManyAttempts
	}
	su, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		checkPassword(password, dummyHash)
		return "", User{}, ErrBadLogin
	}
	if err != nil {
		return "", User{}, err
	}
	if su.PasswordHash == "" {
		checkPassword(password, dummyHash)
		return "", User{}, ErrBadLogin
	}
	if !checkPassword(password, su.PasswordHash) {
		return "", User{}, ErrBadLogin
	}
	token, err := s.newSession(ctx, su.ID)
	return token, userView(su), err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, hashToken(token))
}

func (s *Service) UserFromSession(ctx context.Context, token string) (store.User, error) {
	if token == "" {
		return store.User{}, ErrUnauthorized
	}
	u, err := s.store.UserBySession(ctx, hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return u, ErrUnauthorized
	}
	return u, err
}

type Me struct {
	User User `json:"user"`
}

func (s *Service) Me(u store.User) Me {
	return Me{User: userView(u)}
}

func (s *Service) newSession(ctx context.Context, userID string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.store.CreateSession(ctx, hashToken(token), userID, s.now().Add(sessionLength)); err != nil {
		return "", err
	}
	return token, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
