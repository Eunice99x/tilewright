package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"saas-starter/api/internal/store"
)

func TestAdminPlan(t *testing.T) {
	if p := currentPlan(store.User{IsAdmin: true, Plan: "free"}); p.ID != "admin" {
		t.Errorf("admin plan: %+v", p)
	}
	for _, status := range []string{"", "active"} {
		if p := currentPlan(store.User{Plan: "admin", SubscriptionStatus: status}); p.ID != "free" {
			t.Errorf("a stored plan name must not grant admin (status %q): %+v", status, p)
		}
	}
}

func TestAdminUsers(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := New(f, Config{})
	admin := store.User{IsAdmin: true}

	if _, err := s.AdminUsers(ctx, store.User{ID: "someone"}, AdminUsersQuery{}); !errors.Is(err, ErrNotFound) {
		t.Errorf("not an admin: %v", err)
	}
	p, err := s.AdminUsers(ctx, admin, AdminUsersQuery{Search: "  a\x00b\xff  ", Plan: testPlan, Confirmed: "false", Offset: -5})
	if err != nil {
		t.Fatal(err)
	}
	if f.adminFilter.Search != "ab" || f.adminFilter.Plan != testPlan || f.adminFilter.Confirmed == nil || *f.adminFilter.Confirmed {
		t.Errorf("filter: %+v", f.adminFilter)
	}
	if f.adminLimit != 50 || f.adminOffset != 0 || p.Offset != 0 || p.PageSize != 50 {
		t.Errorf("paging: limit %d, offset %d, %+v", f.adminLimit, f.adminOffset, p)
	}
	if p.Users == nil {
		t.Error("no users must be [] in JSON, not null")
	}
	if _, err := s.AdminUsers(ctx, admin, AdminUsersQuery{Search: strings.Repeat("é", 300)}); err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(f.adminFilter.Search)); got != adminSearchMax {
		t.Errorf("long search kept %d characters", got)
	}
	for plan := range plans {
		if _, err := s.AdminUsers(ctx, admin, AdminUsersQuery{Plan: plan}); err != nil {
			t.Errorf("plan %s: %v", plan, err)
		}
	}
	for _, q := range []AdminUsersQuery{{Plan: "gold"}, {Plan: "' OR 1=1"}, {Confirmed: "yes"}} {
		if _, err := s.AdminUsers(ctx, admin, q); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: %v", q, err)
		}
	}
}

func TestAdminUserIDs(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := New(f, Config{})
	admin := store.User{IsAdmin: true}
	u, err := f.CreateUser(ctx, "someone@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AdminUser(ctx, store.User{ID: u.ID}, u.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("the account itself, not an admin: %v", err)
	}
	for _, id := range []string{"", "1", "not-a-uuid", u.ID + "0", "../me", "' OR 1=1 --"} {
		if _, err := s.AdminUser(ctx, admin, id); !errors.Is(err, ErrNotFound) {
			t.Errorf("malformed %q: %v", id, err)
		}
	}
	if f.adminLookups != 0 {
		t.Errorf("malformed ids reached the database %d times: Postgres would answer 500", f.adminLookups)
	}
	if _, err := s.AdminUser(ctx, admin, "00000000-0000-4000-8000-999999999999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}
	d, err := s.AdminUser(ctx, admin, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Email != u.Email {
		t.Errorf("detail: %+v", d)
	}
}

func TestAdminResponsesHoldNoSecrets(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := New(f, Config{})
	admin := store.User{IsAdmin: true}
	token, u, err := s.Signup(ctx, "203.0.113.9", "target@example.com", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	stored := f.users[u.ID]
	stored.PaddleCustomerID, stored.SubscriptionID = "ctm_shown", "sub_shown"
	f.users[u.ID] = stored
	list, err := s.AdminUsers(ctx, admin, AdminUsersQuery{})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.AdminUser(ctx, admin, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal([]any{list, detail})
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)
	for _, secret := range []string{stored.PasswordHash, token, hashToken(token), "correct horse", "password_hash", "token"} {
		if secret == "" {
			t.Fatal("a sentinel is empty, so this test proves nothing")
		}
		if strings.Contains(body, secret) {
			t.Errorf("response holds %q: %s", secret, body)
		}
	}
	for _, shown := range []string{"ctm_shown", "sub_shown"} {
		if !strings.Contains(body, shown) {
			t.Errorf("response lost %q: %s", shown, body)
		}
	}
}
