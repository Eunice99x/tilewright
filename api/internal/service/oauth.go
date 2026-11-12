package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"saas-starter/api/internal/store"
)

const (
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	githubTokenURL    = "https://github.com/login/oauth/access_token"
	githubUserURL     = "https://api.github.com/user"
	githubEmailsURL   = "https://api.github.com/user/emails?per_page=100"
)

type OAuth struct {
	GoogleID, GoogleSecret string
	GitHubID, GitHubSecret string
	Client                 *http.Client
}

type oauthIdentity struct{ subject, email string }

func (s *Service) OAuthLogin(ctx context.Context, ip, provider, code, redirectURI string) (string, User, error) {
	var fetchIdentity func(ctx context.Context, code, redirectURI string) (oauthIdentity, error)
	var id, secret string
	switch provider {
	case "google":
		fetchIdentity, id, secret = s.googleIdentity, s.oauth.GoogleID, s.oauth.GoogleSecret
	case "github":
		fetchIdentity, id, secret = s.githubIdentity, s.oauth.GitHubID, s.oauth.GitHubSecret
	default:
		return "", User{}, ErrInvalid
	}
	if id == "" || secret == "" {
		return "", User{}, ErrOAuthDisabled
	}
	if code == "" || redirectURI == "" {
		return "", User{}, ErrInvalid
	}
	if !s.loginsByIP.Hit(ip, s.now()) {
		return "", User{}, ErrTooManyAttempts
	}

	who, err := fetchIdentity(ctx, code, redirectURI)
	if err != nil {
		if !errors.Is(err, ErrOAuthNoVerifiedEmail) {
			slog.Warn("oauth exchange failed", "provider", provider, "err", err)
			err = ErrOAuthFailed
		}
		return "", User{}, err
	}
	email := strings.ToLower(strings.TrimSpace(who.email))
	if !validEmail(email) {
		return "", User{}, ErrOAuthNoVerifiedEmail
	}

	su, err := s.findOrCreateOAuthUser(ctx, ip, provider, who.subject, email)
	if err != nil {
		return "", User{}, err
	}
	token, err := s.newSession(ctx, su.ID)
	return token, userView(su), err
}

func (s *Service) findOrCreateOAuthUser(ctx context.Context, ip, provider, subject, email string) (store.User, error) {
	su, err := s.store.UserByOAuthIdentity(ctx, provider, subject)
	if !errors.Is(err, store.ErrNotFound) {
		return su, err
	}
	su, err = s.store.UserByEmail(ctx, email)
	if err == nil {
		return s.existingAccount(ctx, su, provider, subject)
	}
	if !errors.Is(err, store.ErrNotFound) {
		return su, err
	}
	if !s.signupsByIP.Hit(ip, s.now()) {
		return su, ErrTooManyAttempts
	}
	su, err = s.store.CreateOAuthUser(ctx, email, provider, subject)
	if !errors.Is(err, store.ErrEmailTaken) {
		return su, err
	}

	su, err = s.store.UserByOAuthIdentity(ctx, provider, subject)
	if !errors.Is(err, store.ErrNotFound) {
		return su, err
	}
	su, err = s.store.UserByEmail(ctx, email)
	if err != nil {
		return su, err
	}
	return s.existingAccount(ctx, su, provider, subject)
}

func (s *Service) existingAccount(ctx context.Context, su store.User, provider, subject string) (store.User, error) {
	switch {
	case su.PasswordHash == "":
		return su, ErrOAuthOtherIdentity
	case su.EmailVerifiedAt != nil:
		return su, ErrOAuthPasswordAccount
	}
	claimed, err := s.store.ClaimUnverifiedAccount(ctx, su.ID, provider, subject)
	if errors.Is(err, store.ErrEmailTaken) {
		return su, ErrOAuthPasswordAccount
	}
	return claimed, err
}

func (s *Service) googleIdentity(ctx context.Context, code, redirectURI string) (oauthIdentity, error) {
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	err := s.oauthCall(ctx, googleTokenURL, "", url.Values{
		"code": {code}, "client_id": {s.oauth.GoogleID}, "client_secret": {s.oauth.GoogleSecret},
		"redirect_uri": {redirectURI}, "grant_type": {"authorization_code"},
	}, &tok)
	if err != nil {
		return oauthIdentity{}, err
	}
	if tok.AccessToken == "" {
		return oauthIdentity{}, errors.New("google: no access token")
	}
	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := s.oauthCall(ctx, googleUserInfoURL, tok.AccessToken, nil, &info); err != nil {
		return oauthIdentity{}, err
	}
	if info.Sub == "" {
		return oauthIdentity{}, errors.New("google: no subject")
	}
	if info.Email == "" || !info.EmailVerified {
		return oauthIdentity{}, ErrOAuthNoVerifiedEmail
	}
	return oauthIdentity{subject: info.Sub, email: info.Email}, nil
}

func (s *Service) githubIdentity(ctx context.Context, code, redirectURI string) (oauthIdentity, error) {
	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	err := s.oauthCall(ctx, githubTokenURL, "", url.Values{
		"code": {code}, "client_id": {s.oauth.GitHubID}, "client_secret": {s.oauth.GitHubSecret}, "redirect_uri": {redirectURI},
	}, &tok)
	if err != nil {
		return oauthIdentity{}, err
	}
	if tok.AccessToken == "" {
		return oauthIdentity{}, fmt.Errorf("github: no access token: %s", tok.Error)
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if err := s.oauthCall(ctx, githubUserURL, tok.AccessToken, nil, &user); err != nil {
		return oauthIdentity{}, err
	}
	if user.ID == 0 {
		return oauthIdentity{}, errors.New("github: no user id")
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := s.oauthCall(ctx, githubEmailsURL, tok.AccessToken, nil, &emails); err != nil {
		return oauthIdentity{}, err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return oauthIdentity{subject: strconv.FormatInt(user.ID, 10), email: e.Email}, nil
		}
	}
	return oauthIdentity{}, ErrOAuthNoVerifiedEmail
}

func (s *Service) oauthCall(ctx context.Context, target, bearer string, form url.Values, out any) error {
	method, body := http.MethodGet, io.Reader(nil)
	if form != nil {
		method, body = http.MethodPost, strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", App.Name)
	res, err := s.oauth.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("%s %s: http %d", method, req.URL.Host+req.URL.Path, res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
}
