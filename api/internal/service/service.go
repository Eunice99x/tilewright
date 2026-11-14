package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"time"

	"saas-starter/api/internal/game"
	"saas-starter/api/internal/store"
)

var (
	ErrInvalid         = errors.New("invalid_input")
	ErrInvalidEmail    = errors.New("invalid_email")
	ErrWeakPassword    = errors.New("weak_password")
	ErrBadLogin        = errors.New("bad_credentials")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrEmailTaken      = store.ErrEmailTaken
	ErrNotFound        = store.ErrNotFound
	ErrTooManyAttempts = errors.New("too_many_attempts")
	ErrBadSignature    = errors.New("bad_signature")
	ErrBillingDisabled = errors.New("billing_not_configured")

	ErrOAuthDisabled        = errors.New("oauth_not_configured")
	ErrOAuthFailed          = errors.New("oauth_failed")
	ErrOAuthNoVerifiedEmail = errors.New("oauth_no_verified_email")
	ErrOAuthPasswordAccount = errors.New("use_password_login")
	ErrOAuthOtherIdentity   = errors.New("oauth_other_identity")
	ErrInvalidToken         = errors.New("invalid_token")
	ErrNoSubscription       = errors.New("no_subscription")
	ErrEmailFailed          = errors.New("email_failed")
)

type Store interface {
	CreateUser(ctx context.Context, email, passwordHash string) (store.User, error)
	UserByEmail(ctx context.Context, email string) (store.User, error)
	UserByOAuthIdentity(ctx context.Context, provider, subject string) (store.User, error)
	CreateOAuthUser(ctx context.Context, email, provider, subject string) (store.User, error)
	AdminUsers(ctx context.Context, f store.AdminUserFilter, limit, offset int) ([]store.AdminUser, int, error)
	AdminUserByID(ctx context.Context, id string) (store.AdminUserDetail, error)
	SaveFeedback(ctx context.Context, userID, email, message, page string) error
	AdminFeedback(ctx context.Context, limit, offset int) ([]store.Feedback, int, error)
	ClaimUnverifiedAccount(ctx context.Context, userID, provider, subject string) (store.User, error)
	SaveEmailToken(ctx context.Context, userID, kind, tokenHash string, expires time.Time) error
	ConfirmEmail(ctx context.Context, tokenHash, userID string) error
	ResetPassword(ctx context.Context, tokenHash, passwordHash string) error
	CreateSession(ctx context.Context, tokenHash, userID string, expires time.Time) error
	UserBySession(ctx context.Context, tokenHash string) (store.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error

	ApplyPaddleEvent(ctx context.Context, eventID, eventType string, sub *store.Subscription) error
	ApplyPaddlePurchase(ctx context.Context, eventID, eventType string, p store.Purchase) error

	CreateGuest(ctx context.Context, email, name string) (store.User, error)
	CreateGame(ctx context.Context, userID, mode, state string, inviteCode *string) (store.Game, error)
	ActiveSoloGame(ctx context.Context, userID string) (store.Game, error)
	GameByInvite(ctx context.Context, code string) (store.Game, error)
	AddPlayer(ctx context.Context, gameID, userID string, seat int) error
	PlayerGame(ctx context.Context, userID, id string) (store.PlayerGame, error)
	PlayerGames(ctx context.Context, userID string, limit int) ([]store.PlayerGame, error)
	SaveGame(ctx context.Context, g store.Game) (store.Game, error)
	BestGames(ctx context.Context, userID string, limit int) ([]store.GameSummary, error)
}

type Config struct {
	Paddle     Paddle
	OAuth      OAuth
	Mail       Mailer
	SiteURL    string
	ForgotTime time.Duration
	Now        func() time.Time
	Words      game.Dictionary
	Shuffle    game.Shuffler
}

type Service struct {
	store      Store
	paddle     Paddle
	oauth      OAuth
	mail       Mailer
	siteURL    string
	forgotTime time.Duration
	now        func() time.Time
	dict       game.Dictionary
	shuffle    game.Shuffler

	loginsByEmail *attempts
	loginsByIP    *attempts
	signupsByIP   *attempts
	guestsByIP    *attempts

	emailsByAddress *attempts
	emailsByIP      *attempts

	feedbackByIP *attempts
}

func New(st Store, cfg Config) *Service {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.OAuth.Client == nil {
		cfg.OAuth.Client = &http.Client{Timeout: 8 * time.Second}
	}
	if cfg.Mail == nil {
		cfg.Mail = ResendMailer{}
	}
	if cfg.SiteURL == "" {
		cfg.SiteURL = "http://localhost:3100"
	}
	if cfg.Shuffle == nil {
		cfg.Shuffle = rand.Shuffle
	}
	if cfg.ForgotTime == 0 {
		cfg.ForgotTime = 1500 * time.Millisecond
	}
	return &Service{
		store:      st,
		paddle:     cfg.Paddle,
		oauth:      cfg.OAuth,
		mail:       cfg.Mail,
		siteURL:    cfg.SiteURL,
		forgotTime: cfg.ForgotTime,
		now:        cfg.Now,
		dict:       cfg.Words,
		shuffle:    cfg.Shuffle,

		loginsByEmail: newAttempts(loginEmailLimit, loginWindow),
		loginsByIP:    newAttempts(loginIPLimit, loginWindow),
		signupsByIP:   newAttempts(signupIPLimit, signupWindow),
		guestsByIP:    newAttempts(guestIPLimit, time.Hour),

		emailsByAddress: newAttempts(emailAddressLimit, time.Hour),
		emailsByIP:      newAttempts(emailIPLimit, time.Hour),

		feedbackByIP: newAttempts(feedbackIPLimit, time.Hour),
	}
}
