package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email_taken")
)

type Store struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Store {
	return &Store{db: db}
}

type User struct {
	ID                 string     `db:"id"`
	Email              string     `db:"email"`
	PasswordHash       string     `db:"password_hash"`
	Plan               string     `db:"plan"`
	PaddleCustomerID   string     `db:"paddle_customer_id"`
	SubscriptionID     string     `db:"subscription_id"`
	SubscriptionStatus string     `db:"subscription_status"`
	CreatedAt          time.Time  `db:"created_at"`
	EmailVerifiedAt    *time.Time `db:"email_verified_at"`
	IsAdmin            bool       `db:"is_admin"`
	LifetimePlan       string     `db:"lifetime_plan"`
	DisplayName        string     `db:"display_name"`
}

const userCols = `id, email, password_hash, plan, paddle_customer_id, subscription_id, subscription_status, created_at, email_verified_at, is_admin, lifetime_plan, display_name`

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING `+userCols, email, passwordHash)
	if isUniqueViolation(err) {
		return u, ErrEmailTaken
	}
	return u, err
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	return s.getUser(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email)
}

func (s *Store) UserByOAuthIdentity(ctx context.Context, provider, subject string) (User, error) {
	return s.getUser(ctx, `SELECT `+userCols+` FROM users
		WHERE id = (SELECT user_id FROM oauth_identities WHERE provider = $1 AND subject = $2)`, provider, subject)
}

func (s *Store) CreateOAuthUser(ctx context.Context, email, provider, subject string) (User, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	var u User
	err = tx.GetContext(ctx, &u, `INSERT INTO users (email, password_hash, email_verified_at) VALUES ($1, '', now()) RETURNING `+userCols, email)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO oauth_identities (provider, subject, user_id) VALUES ($1, $2, $3)`, provider, subject, u.ID); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return u, tx.Commit()
}

func (s *Store) UserByID(ctx context.Context, id string) (User, error) {
	return s.getUser(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id)
}

func (s *Store) getUser(ctx context.Context, query string, args ...any) (User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

const (
	TokenVerify = "verify"
	TokenReset  = "reset"
)

func (s *Store) SaveEmailToken(ctx context.Context, userID, kind, tokenHash string, expires time.Time) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `DELETE FROM email_tokens WHERE user_id = $1 AND kind = $2`, userID, kind); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO email_tokens (token_hash, user_id, kind, expires_at) VALUES ($1, $2, $3, $4)`, tokenHash, userID, kind, expires); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ConfirmEmail(ctx context.Context, tokenHash, userID string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := spendToken(ctx, tx, TokenVerify, tokenHash, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()) WHERE id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	var userID string
	err = tx.GetContext(ctx, &userID, `SELECT user_id FROM email_tokens WHERE token_hash = $1 AND kind = $2 AND expires_at > now() FOR UPDATE`, tokenHash, TokenReset)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := spendToken(ctx, tx, TokenReset, tokenHash, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = $2, email_verified_at = COALESCE(email_verified_at, now()) WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM email_tokens WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func spendToken(ctx context.Context, tx *sqlx.Tx, kind, tokenHash, userID string) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM email_tokens WHERE token_hash = $1 AND kind = $2 AND user_id = $3 AND expires_at > now()`, tokenHash, kind, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ClaimUnverifiedAccount(ctx context.Context, userID, provider, subject string) (User, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	var u User
	err = tx.GetContext(ctx, &u, `UPDATE users SET password_hash = '', email_verified_at = now()
		WHERE id = $1 AND email_verified_at IS NULL RETURNING `+userCols, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	for _, q := range []string{`DELETE FROM sessions WHERE user_id = $1`, `DELETE FROM email_tokens WHERE user_id = $1`} {
		if _, err := tx.ExecContext(ctx, q, userID); err != nil {
			return User{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO oauth_identities (provider, subject, user_id) VALUES ($1, $2, $3)`, provider, subject, userID); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return u, tx.Commit()
}

func (s *Store) CreateSession(ctx context.Context, tokenHash, userID string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, tokenHash, userID, expires)
	return err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (User, error) {
	return s.getUser(ctx, `SELECT u.id, u.email, u.password_hash, u.plan, u.paddle_customer_id, u.subscription_id,
			u.subscription_status, u.created_at, u.email_verified_at, u.is_admin, u.lifetime_plan, u.display_name
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash)
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

type Subscription struct {
	UserID, Plan, Status, CustomerID, SubscriptionID string
	OccurredAt                                       time.Time
}

func SubscriptionApplies(curID, curStatus string, sub *Subscription) bool {
	if sub.SubscriptionID == curID {
		return true
	}
	return isActive(sub.Status) && !isActive(curStatus)
}

func isActive(status string) bool { return status == "active" || status == "trialing" }

func (s *Store) ApplyPaddleEvent(ctx context.Context, eventID, eventType string, sub *Subscription) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO paddle_events (event_id, event_type) VALUES ($1, $2) ON CONFLICT DO NOTHING`, eventID, eventType)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	if sub != nil {
		var cur struct {
			ID        string     `db:"subscription_id"`
			Status    string     `db:"subscription_status"`
			UpdatedAt *time.Time `db:"paddle_updated_at"`
		}
		err := tx.GetContext(ctx, &cur, `SELECT subscription_id, subscription_status, paddle_updated_at
			FROM users WHERE id = $1 FOR UPDATE`, sub.UserID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if (cur.UpdatedAt == nil || !sub.OccurredAt.Before(*cur.UpdatedAt)) && SubscriptionApplies(cur.ID, cur.Status, sub) {
			_, err = tx.ExecContext(ctx, `
				UPDATE users SET plan = $2, subscription_status = $3, paddle_customer_id = $4,
					subscription_id = $5, paddle_updated_at = $6
				WHERE id = $1`,
				sub.UserID, sub.Plan, sub.Status, sub.CustomerID, sub.SubscriptionID, sub.OccurredAt)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

type Purchase struct {
	UserID, Plan, TransactionID string
}

func (s *Store) ApplyPaddlePurchase(ctx context.Context, eventID, eventType string, p Purchase) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO paddle_events (event_id, event_type) VALUES ($1, $2) ON CONFLICT DO NOTHING`, eventID, eventType)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	res, err = tx.ExecContext(ctx, `UPDATE users SET lifetime_plan = $2, lifetime_transaction_id = $3 WHERE id = $1`, p.UserID, p.Plan, p.TransactionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
