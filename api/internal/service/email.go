package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"saas-starter/api/internal/store"
)

const (
	verifyLinkLife = 24 * time.Hour
	resetLinkLife  = time.Hour
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type ResendMailer struct {
	Key, From, ReplyTo string
	Client             *http.Client
}

func (m ResendMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.Key == "" {
		slog.Warn("email not sent: RESEND_API_KEY is not set", "subject", subject)
		return nil
	}
	msg := map[string]any{"from": m.From, "to": []string{to}, "subject": subject, "text": body}
	if m.ReplyTo != "" {
		msg["reply_to"] = m.ReplyTo
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.Key)
	req.Header.Set("Content-Type", "application/json")
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return fmt.Errorf("resend: http %d: %s", res.StatusCode, detail)
	}
	return nil
}

func (s *Service) emailLink(ctx context.Context, u store.User, kind, path string, life time.Duration) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.store.SaveEmailToken(ctx, u.ID, kind, hashToken(token), s.now().Add(life)); err != nil {
		return "", err
	}
	return strings.TrimRight(s.siteURL, "/") + path + "?token=" + url.QueryEscape(token), nil
}

func (s *Service) sendVerification(ctx context.Context, u store.User) error {
	link, err := s.emailLink(ctx, u, store.TokenVerify, "/verify", verifyLinkLife)
	if err != nil {
		return err
	}
	return s.mail.Send(ctx, u.Email, "Confirm your "+App.Name+" email",
		"Confirm this address:\n\n"+link+"\n\nThe link works for 24 hours. If you didn't sign up for "+App.Name+", ignore this email.\n")
}

func (s *Service) ResendVerification(ctx context.Context, u store.User) error {
	if u.EmailVerifiedAt != nil || isGuest(u.Email) {
		return nil
	}
	if !s.emailsByAddress.Hit(u.Email, s.now()) {
		return ErrTooManyAttempts
	}
	if err := s.sendVerification(ctx, u); err != nil {
		slog.Warn("send verification email", "err", err)
		return ErrEmailFailed
	}
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, u store.User, token string) error {
	err := s.store.ConfirmEmail(ctx, hashToken(token), u.ID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrInvalidToken
	}
	return err
}

func (s *Service) RequestPasswordReset(ctx context.Context, ip, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return ErrInvalidEmail
	}
	now := s.now()
	if !s.emailsByIP.Hit(ip, now) {
		return ErrTooManyAttempts
	}
	if s.forgotTime > 0 {
		defer func(until time.Time) {
			select {
			case <-time.After(time.Until(until)):
			case <-ctx.Done():
			}
		}(time.Now().Add(s.forgotTime))
	}
	if !s.emailsByAddress.Hit(email, now) {
		return nil
	}
	u, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if u.PasswordHash == "" {
		err = s.mail.Send(ctx, u.Email, "Sign in to "+App.Name,
			"Your "+App.Name+" account uses Google or GitHub to sign in, so it has no password to reset. Use the same button here:\n\n"+
				strings.TrimRight(s.siteURL, "/")+"/login\n")
	} else {
		var link string
		link, err = s.emailLink(ctx, u, store.TokenReset, "/reset", resetLinkLife)
		if err == nil {
			err = s.mail.Send(ctx, u.Email, "Reset your "+App.Name+" password",
				"Choose a new password:\n\n"+link+"\n\nThe link works for 1 hour. If you didn't ask for this, ignore this email and your password stays the same.\n")
		}
	}
	if err != nil {
		slog.Warn("send reset email", "err", err)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, ip, token, password string) error {
	if len(password) < 8 || len(password) > 72 {
		return ErrWeakPassword
	}
	if !s.loginsByIP.Hit(ip, s.now()) {
		return ErrTooManyAttempts
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	err = s.store.ResetPassword(ctx, hashToken(token), hash)
	if errors.Is(err, store.ErrNotFound) {
		return ErrInvalidToken
	}
	return err
}
