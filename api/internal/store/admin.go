package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
)

type AdminUser struct {
	ID        string         `db:"id" json:"id"`
	Email     string         `db:"email" json:"email"`
	Plan      string         `db:"plan" json:"plan"`
	Status    string         `db:"subscription_status" json:"status"`
	Confirmed bool           `db:"confirmed" json:"confirmed"`
	SignIn    pq.StringArray `db:"sign_in" json:"sign_in"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	LastSeen  *time.Time     `db:"last_seen" json:"last_seen"` // the newest live sign-in; nil when none is left
}

type AdminUserDetail struct {
	AdminUser
	PaddleCustomerID string `db:"paddle_customer_id" json:"paddle_customer_id"`
	SubscriptionID   string `db:"subscription_id" json:"subscription_id"`
	LifetimePlan     string `db:"lifetime_plan" json:"lifetime_plan"`
	LifetimeTxn      string `db:"lifetime_transaction_id" json:"lifetime_transaction_id"`
}

type AdminUserFilter struct {
	Search    string
	Plan      string
	Confirmed *bool
}

const adminPlan = `CASE WHEN u.is_admin THEN 'admin'
	WHEN u.plan NOT IN ('free', 'admin') AND u.subscription_status IN ('active', 'trialing', 'past_due') THEN u.plan
	WHEN u.lifetime_plan <> '' THEN u.lifetime_plan
	ELSE 'free' END`

const adminUserCols = `u.id, u.email, ` + adminPlan + ` AS plan, u.subscription_status, u.email_verified_at IS NOT NULL AS confirmed,
	ARRAY(SELECT m FROM (
		SELECT 'password' AS m WHERE u.password_hash <> ''
		UNION SELECT i.provider FROM oauth_identities i WHERE i.user_id = u.id
	) methods ORDER BY m = 'password' DESC, m) AS sign_in,
	u.created_at,
	(SELECT max(s.created_at) FROM sessions s WHERE s.user_id = u.id) AS last_seen`

var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace

func (s *Store) AdminUsers(ctx context.Context, f AdminUserFilter, limit, offset int) ([]AdminUser, int, error) {
	pattern := ""
	if f.Search != "" {
		pattern = "%" + escapeLike(f.Search) + "%"
	}

	const where = ` FROM users u WHERE ($1 = '' OR u.email::text ILIKE $1 ESCAPE '\')
		AND ($2 = '' OR ` + adminPlan + ` = $2) AND ($3::boolean IS NULL OR (u.email_verified_at IS NOT NULL) = $3)`
	var total int
	err := s.db.GetContext(ctx, &total, `SELECT count(*)`+where, pattern, f.Plan, f.Confirmed)
	if err != nil {
		return nil, 0, err
	}
	out := []AdminUser{}
	err = s.db.SelectContext(ctx, &out, `SELECT `+adminUserCols+where+` ORDER BY u.created_at DESC, u.id DESC LIMIT $4 OFFSET $5`,
		pattern, f.Plan, f.Confirmed, limit, offset)
	return out, total, err
}

func (s *Store) AdminUserByID(ctx context.Context, id string) (AdminUserDetail, error) {
	var d AdminUserDetail
	err := s.db.GetContext(ctx, &d, `SELECT `+adminUserCols+`, u.paddle_customer_id, u.subscription_id, u.lifetime_plan, u.lifetime_transaction_id
		FROM users u WHERE u.id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}
