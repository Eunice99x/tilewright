package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, ctx := New(db), context.Background()

	email := fmt.Sprintf("it-%d@example.com", time.Now().UnixNano())
	u, err := s.CreateUser(ctx, email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE id = $1`, u.ID) })
	if _, err := s.CreateUser(ctx, "IT-"+email[3:], "hash"); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("duplicate email (other case): %v", err)
	}

	subject := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	oauthEmail := "oauth-" + email
	ou, err := s.CreateOAuthUser(ctx, oauthEmail, "github", subject)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE id = $1`, ou.ID) })
	if ou.PasswordHash != "" {
		t.Error("OAuth user has a password hash")
	}
	if got, err := s.UserByOAuthIdentity(ctx, "github", subject); err != nil || got.ID != ou.ID {
		t.Errorf("by identity: %+v, %v", got, err)
	}
	if _, err := s.UserByOAuthIdentity(ctx, "google", subject); !errors.Is(err, ErrNotFound) {
		t.Errorf("other provider: %v", err)
	}
	if _, err := s.CreateOAuthUser(ctx, oauthEmail, "google", subject); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("taken email: %v", err)
	}
	if _, err := s.CreateOAuthUser(ctx, "second-"+oauthEmail, "github", subject); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("taken identity: %v", err)
	}
	if _, err := s.UserByEmail(ctx, "second-"+oauthEmail); !errors.Is(err, ErrNotFound) {
		t.Errorf("a failed identity left a user behind: %v", err)
	}

	if err := s.SaveEmailToken(ctx, u.ID, TokenVerify, "t1-"+u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveEmailToken(ctx, u.ID, TokenVerify, "t2-"+u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmEmail(ctx, "t1-"+u.ID, u.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("a replaced link still works: %v", err)
	}
	if err := s.ConfirmEmail(ctx, "t2-"+u.ID, ou.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another account's link works: %v", err)
	}
	if err := s.ResetPassword(ctx, "t2-"+u.ID, "h"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a link of another kind works: %v", err)
	}
	if err := s.ConfirmEmail(ctx, "t2-"+u.ID, u.ID); err != nil {
		t.Errorf("confirm: %v", err)
	}
	if got, _ := s.UserByID(ctx, u.ID); got.EmailVerifiedAt == nil {
		t.Error("not confirmed")
	}
	if err := s.ConfirmEmail(ctx, "t2-"+u.ID, u.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("a used link works again: %v", err)
	}
	if err := s.SaveEmailToken(ctx, u.ID, TokenReset, "t3-"+u.ID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, "t3-"+u.ID, "h"); !errors.Is(err, ErrNotFound) {
		t.Errorf("an expired link works: %v", err)
	}
	squat, err := s.CreateUser(ctx, "squat-"+email, "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE id = $1`, squat.ID) })
	if err := s.CreateSession(ctx, "sq-"+squat.ID, squat.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	subject2 := "claim-" + subject
	claimed, err := s.ClaimUnverifiedAccount(ctx, squat.ID, "google", subject2)
	if err != nil || claimed.PasswordHash != "" || claimed.EmailVerifiedAt == nil {
		t.Fatalf("claim: %+v, %v", claimed, err)
	}
	if _, err := s.UserBySession(ctx, "sq-"+squat.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("claim kept a session: %v", err)
	}
	if _, err := s.ClaimUnverifiedAccount(ctx, squat.ID, "github", subject2); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("claim of a confirmed account: %v", err)
	}
	if err := s.CreateSession(ctx, "rs-"+u.ID, u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveEmailToken(ctx, u.ID, TokenReset, "t4-"+u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, "t4-"+u.ID, "newhash"); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, "t4-"+u.ID, "again"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a used reset link: %v", err)
	}
	if _, err := s.UserBySession(ctx, "rs-"+u.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("reset kept a session: %v", err)
	}
	if got, err := s.UserByID(ctx, u.ID); err != nil || got.PasswordHash != "newhash" || got.EmailVerifiedAt == nil {
		t.Errorf("after reset: %+v, %v", got, err)
	}
	if ou2, err := s.UserByOAuthIdentity(ctx, "github", subject); err != nil || ou2.EmailVerifiedAt == nil {
		t.Errorf("an OAuth account should start confirmed: %+v, %v", ou2, err)
	}

	if err := s.CreateSession(ctx, "h-"+u.ID, u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got, err := s.UserBySession(ctx, "h-"+u.ID); err != nil || got.ID != u.ID {
		t.Errorf("session: %v", err)
	}
	if err := s.CreateSession(ctx, "old-"+u.ID, u.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserBySession(ctx, "old-"+u.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expired session: %v", err)
	}

	evt := "evt-" + u.ID
	t.Cleanup(func() { db.Exec(`DELETE FROM paddle_events WHERE event_id = $1`, evt) })
	sub := &Subscription{UserID: u.ID, Plan: "starter", Status: "active", CustomerID: "ctm", SubscriptionID: "sub", OccurredAt: time.Now()}
	if err := s.ApplyPaddleEvent(ctx, evt, "subscription.activated", sub); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.UserByID(ctx, u.ID); got.Plan != "starter" || got.SubscriptionStatus != "active" {
		t.Errorf("after paddle event: %+v", got)
	}

	stale := "evt-stale-" + u.ID
	t.Cleanup(func() { db.Exec(`DELETE FROM paddle_events WHERE event_id = $1`, stale) })
	old := &Subscription{UserID: u.ID, Plan: "starter", Status: "canceled", CustomerID: "ctm", SubscriptionID: "old", OccurredAt: time.Now()}
	if err := s.ApplyPaddleEvent(ctx, stale, "subscription.canceled", old); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.UserByID(ctx, u.ID); got.SubscriptionStatus != "active" || got.SubscriptionID != "sub" {
		t.Errorf("after stale cancel: %+v", got)
	}
	var n int
	db.Get(&n, `SELECT count(*) FROM paddle_events WHERE event_id = $1`, stale)
	if n != 1 {
		t.Errorf("ignored event not recorded: %d", n)
	}

	lost := "evt-lost-" + u.ID
	t.Cleanup(func() { db.Exec(`DELETE FROM paddle_events WHERE event_id = $1`, lost) })
	nobody := &Subscription{UserID: "00000000-0000-4000-8000-000000000000", Plan: "starter", Status: "active", SubscriptionID: "sub", OccurredAt: time.Now()}
	if err := s.ApplyPaddleEvent(ctx, lost, "subscription.activated", nobody); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user: %v", err)
	}
	db.Get(&n, `SELECT count(*) FROM paddle_events WHERE event_id = $1`, lost)
	if n != 0 {
		t.Errorf("event for unknown user recorded: %d", n)
	}
}

func TestPaddlePurchaseIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, ctx := New(db), context.Background()

	mark := fmt.Sprintf("buy%d", time.Now().UnixNano())
	u, err := s.CreateUser(ctx, mark+"@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE id = $1`, u.ID) })
	evt := "evt-" + mark
	t.Cleanup(func() { db.Exec(`DELETE FROM paddle_events WHERE event_id LIKE $1`, evt+"%") })
	p := Purchase{UserID: u.ID, Plan: "lifetime", TransactionID: "txn_" + mark}

	if err := s.ApplyPaddlePurchase(ctx, evt, "transaction.completed", p); err != nil {
		t.Fatal(err)
	}
	got, err := s.UserByID(ctx, u.ID)
	if err != nil || got.LifetimePlan != "lifetime" || got.SubscriptionID != "" || got.SubscriptionStatus != "" {
		t.Fatalf("after purchase: %+v, %v", got, err)
	}
	if bySession := func() User {
		if err := s.CreateSession(ctx, "s-"+mark, u.ID, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		v, err := s.UserBySession(ctx, "s-"+mark)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}(); bySession.LifetimePlan != "lifetime" {
		t.Errorf("session lookup lost the plan: %+v", bySession)
	}
	rows, _, err := s.AdminUsers(ctx, AdminUserFilter{Search: mark, Plan: "lifetime"}, 50, 0)
	if err != nil || len(rows) != 1 || rows[0].Plan != "lifetime" {
		t.Errorf("admin table: %+v, %v", rows, err)
	}
	d, err := s.AdminUserByID(ctx, u.ID)
	if err != nil || d.LifetimePlan != "lifetime" || d.LifetimeTxn != p.TransactionID {
		t.Errorf("admin detail: %+v, %v", d, err)
	}

	if _, err := db.Exec(`UPDATE users SET lifetime_plan = '', lifetime_transaction_id = '' WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyPaddlePurchase(ctx, evt, "transaction.completed", p); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.UserByID(ctx, u.ID); got.LifetimePlan != "" {
		t.Errorf("a repeat restored access: %+v", got)
	}

	lost := evt + "-lost"
	if err := s.ApplyPaddlePurchase(ctx, lost, "transaction.completed", Purchase{UserID: "00000000-0000-4000-8000-000000000000", Plan: "lifetime"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown user: %v", err)
	}
	var n int
	db.Get(&n, `SELECT count(*) FROM paddle_events WHERE event_id = $1`, lost)
	if n != 0 {
		t.Errorf("event for unknown user recorded: %d", n)
	}
}

func TestSubscriptionApplies(t *testing.T) {
	cases := []struct {
		name, curID, curStatus, id, status string
		want                               bool
	}{
		{"first activation", "", "", "sub_1", "active", true},
		{"same subscription renews", "sub_1", "active", "sub_1", "active", true},
		{"same subscription canceled", "sub_1", "active", "sub_1", "canceled", true},
		{"same subscription past due", "sub_1", "active", "sub_1", "past_due", true},
		{"same subscription paused", "sub_1", "active", "sub_1", "paused", true},
		{"old subscription canceled", "sub_2", "active", "sub_1", "canceled", false},
		{"old subscription past due", "sub_2", "active", "sub_1", "past_due", false},
		{"old subscription paused", "sub_2", "trialing", "sub_1", "paused", false},
		{"other subscription active while active", "sub_1", "active", "sub_2", "active", false},
		{"cancel for a user with no subscription", "", "", "sub_1", "canceled", false},
		{"new subscription after a canceled one", "sub_1", "canceled", "sub_2", "active", true},
		{"new subscription replaces a past due one", "sub_1", "past_due", "sub_2", "trialing", true},
		{"canceled after a canceled one", "sub_1", "canceled", "sub_2", "canceled", false},
	}
	for _, c := range cases {
		sub := &Subscription{SubscriptionID: c.id, Status: c.status}
		if got := SubscriptionApplies(c.curID, c.curStatus, sub); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestAdminUsersIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, ctx := New(db), context.Background()

	mark := fmt.Sprintf("adm%d", time.Now().UnixNano())
	base := time.Now().UTC().Add(-time.Hour)
	add := func(i int, email, hash, plan, status string, confirmed, admin bool) string {
		t.Helper()
		var id string
		err := db.Get(&id, `INSERT INTO users (email, password_hash, plan, subscription_status, created_at, email_verified_at, is_admin,
				paddle_customer_id, subscription_id)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() END, $7, 'ctm_'||$8, 'sub_'||$8) RETURNING id`,
			email, hash, plan, status, base.Add(time.Duration(i)*time.Minute), confirmed, admin, mark)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE id = $1`, id) })
		return id
	}

	free := add(0, "One."+mark+"@example.com", "pw-hash-sentinel", "free", "", false, false)
	live := add(1, "two."+mark+"@example.com", "pw-hash-sentinel", "starter", "active", true, false)
	lapsed := add(2, "three."+mark+"@example.com", "", "starter", "canceled", true, false)
	boss := add(3, "four."+mark+"@example.com", "pw-hash-sentinel", "free", "", true, true)
	pct := add(4, "pct%"+mark+"@example.com", "", "free", "", false, false)
	under := add(5, "und_"+mark+"@example.com", "", "free", "", false, false)
	slash := add(6, `sla\`+mark+"@example.com", "", "free", "", false, false)

	if _, err := db.Exec(`INSERT INTO oauth_identities (provider, subject, user_id) VALUES ('github', $1, $2), ('google', $1, $2), ('google', $3, $2)`, mark, lapsed, mark+"b"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO oauth_identities (provider, subject, user_id) VALUES ('google', $1, $2)`, mark+"c", live); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, "session-hash-sentinel-"+mark, live, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	search := func(f AdminUserFilter) ([]string, int) {
		t.Helper()
		if f.Search == "" {
			f.Search = mark
		}
		rows, total, err := s.AdminUsers(ctx, f, 50, 0)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
		}
		return ids, total
	}
	yes, no := true, false

	ids, total := search(AdminUserFilter{})
	if want := []string{slash, under, pct, boss, lapsed, live, free}; !slices.Equal(ids, want) || total != 7 {
		t.Errorf("newest first: %v (%d), want %v", ids, total, want)
	}
	if ids, _ := search(AdminUserFilter{Search: strings.ToUpper("two." + mark)}); !slices.Equal(ids, []string{live}) {
		t.Errorf("any case: %v", ids)
	}
	if ids, _ := search(AdminUserFilter{Search: "ne." + mark + "@EXAMPLE"}); !slices.Equal(ids, []string{free}) {
		t.Errorf("stored case kept, partial match: %v", ids)
	}
	for q, want := range map[string]string{"t%" + mark: pct, "d_" + mark: under, `a\` + mark: slash} {
		if ids, _ := search(AdminUserFilter{Search: q}); !slices.Equal(ids, []string{want}) {
			t.Errorf("%q must match itself only: %v, want %s", q, ids, want)
		}
	}
	for _, q := range []string{"t_" + mark, "%" + mark + "%%", "o%" + mark, "' OR '1'='1"} {
		if ids, _ := search(AdminUserFilter{Search: q}); len(ids) != 0 {
			t.Errorf("%q is not a pattern: %v", q, ids)
		}
	}

	for plan, want := range map[string][]string{
		"starter": {live},
		"admin":   {boss},
		"free":    {slash, under, pct, lapsed, free},
		"no-such": {},
	} {
		if ids, total := search(AdminUserFilter{Plan: plan}); !slices.Equal(ids, want) || total != len(want) {
			t.Errorf("plan %s: %v (%d), want %v", plan, ids, total, want)
		}
	}
	if ids, _ := search(AdminUserFilter{Confirmed: &yes}); !slices.Equal(ids, []string{boss, lapsed, live}) {
		t.Errorf("confirmed: %v", ids)
	}
	if ids, _ := search(AdminUserFilter{Confirmed: &no}); !slices.Equal(ids, []string{slash, under, pct, free}) {
		t.Errorf("not confirmed: %v", ids)
	}
	if ids, _ := search(AdminUserFilter{Plan: "free", Confirmed: &yes}); !slices.Equal(ids, []string{lapsed}) {
		t.Errorf("both: %v", ids)
	}

	page, total, err := s.AdminUsers(ctx, AdminUserFilter{Search: mark}, 2, 2)
	if err != nil || total != 7 || len(page) != 2 || page[0].ID != pct || page[1].ID != boss {
		t.Errorf("page: %+v, %d, %v", page, total, err)
	}
	if page, total, err := s.AdminUsers(ctx, AdminUserFilter{Search: mark}, 2, 100); err != nil || total != 7 || page == nil || len(page) != 0 {
		t.Errorf("past the end keeps the total: %+v, %d, %v", page, total, err)
	}

	d, err := s.AdminUserByID(ctx, live)
	if err != nil {
		t.Fatal(err)
	}
	if d.Plan != "starter" || d.Status != "active" || !d.Confirmed || d.PaddleCustomerID != "ctm_"+mark || d.SubscriptionID != "sub_"+mark {
		t.Errorf("detail: %+v", d)
	}
	if want := []string{"password", "google"}; !slices.Equal(d.SignIn, want) {
		t.Errorf("sign-in: %v", d.SignIn)
	}
	methods := func(id string) []string {
		t.Helper()
		d, err := s.AdminUserByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return d.SignIn
	}
	if got := methods(lapsed); !slices.Equal(got, []string{"github", "google"}) {
		t.Errorf("OAuth only: %v", got)
	}
	if got := methods(pct); got == nil || len(got) != 0 {
		t.Errorf("no method must be an empty list: %#v", got)
	}
	if _, err := s.AdminUserByID(ctx, "00000000-0000-4000-8000-999999999999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}

	rows, _, err := s.AdminUsers(ctx, AdminUserFilter{Search: mark}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal([]any{rows, d})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"pw-hash-sentinel", "session-hash-sentinel", "password_hash", "token"} {
		if strings.Contains(string(out), secret) {
			t.Errorf("JSON holds %q: %s", secret, out)
		}
	}
}
