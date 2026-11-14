package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"saas-starter/api/internal/store"
)

type fakeStore struct {
	mu         sync.Mutex
	n          int
	users      map[string]store.User
	sessions   map[string]string
	events     map[string]bool
	identities map[string]string
	tokens     map[string]fakeToken
	gameRows   *fakeGames

	adminFilter  store.AdminUserFilter
	adminLimit   int
	adminOffset  int
	adminLookups int

	feedback []store.Feedback
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]store.User{}, sessions: map[string]string{}, events: map[string]bool{}}
}

func (f *fakeStore) id() string {
	f.n++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", f.n)
}

func (f *fakeStore) CreateUser(_ context.Context, email, hash string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == email {
			return store.User{}, store.ErrEmailTaken
		}
	}
	u := store.User{ID: f.id(), Email: email, PasswordHash: hash, Plan: "free", CreatedAt: time.Now()}
	f.users[u.ID] = u
	return u, nil
}

type fakeToken struct {
	userID, kind string
	expires      time.Time
}

func (f *fakeStore) SaveEmailToken(_ context.Context, userID, kind, tokenHash string, expires time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tokens == nil {
		f.tokens = map[string]fakeToken{}
	}
	for h, t := range f.tokens {
		if t.userID == userID && t.kind == kind {
			delete(f.tokens, h)
		}
	}
	f.tokens[tokenHash] = fakeToken{userID, kind, expires}
	return nil
}

func (f *fakeStore) ConfirmEmail(_ context.Context, tokenHash, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tokens[tokenHash]
	if !ok || t.kind != store.TokenVerify || t.userID != userID || time.Now().After(t.expires) {
		return store.ErrNotFound
	}
	delete(f.tokens, tokenHash)
	f.setVerified(userID)
	return nil
}

func (f *fakeStore) setVerified(userID string) {
	u := f.users[userID]
	now := time.Now()
	u.EmailVerifiedAt = &now
	f.users[userID] = u
}

func (f *fakeStore) ResetPassword(_ context.Context, tokenHash, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tokens[tokenHash]
	if !ok || t.kind != store.TokenReset || time.Now().After(t.expires) {
		return store.ErrNotFound
	}
	delete(f.tokens, tokenHash)
	u := f.users[t.userID]
	u.PasswordHash = hash
	f.users[t.userID] = u
	f.setVerified(t.userID)
	for tok, id := range f.sessions {
		if id == t.userID {
			delete(f.sessions, tok)
		}
	}
	return nil
}

func (f *fakeStore) ClaimUnverifiedAccount(_ context.Context, userID, provider, subject string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := f.users[userID]
	if u.EmailVerifiedAt != nil {
		return store.User{}, store.ErrEmailTaken
	}
	u.PasswordHash = ""
	now := time.Now()
	u.EmailVerifiedAt = &now
	f.users[userID] = u
	for tok, id := range f.sessions {
		if id == userID {
			delete(f.sessions, tok)
		}
	}
	if f.identities == nil {
		f.identities = map[string]string{}
	}
	f.identities[provider+"/"+subject] = userID
	return u, nil
}

func (f *fakeStore) AdminUsers(_ context.Context, filter store.AdminUserFilter, limit, offset int) ([]store.AdminUser, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.adminFilter, f.adminLimit, f.adminOffset = filter, limit, offset
	out := []store.AdminUser{}
	for _, u := range f.users {
		if strings.Contains(strings.ToLower(u.Email), strings.ToLower(filter.Search)) {
			out = append(out, f.adminRow(u))
		}
	}
	return out, len(out), nil
}

func (f *fakeStore) adminRow(u store.User) store.AdminUser {
	return store.AdminUser{ID: u.ID, Email: u.Email, Plan: u.Plan, Status: u.SubscriptionStatus, Confirmed: u.EmailVerifiedAt != nil, SignIn: []string{"password"}, CreatedAt: u.CreatedAt}
}

func (f *fakeStore) AdminUserByID(_ context.Context, id string) (store.AdminUserDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.adminLookups++
	u, ok := f.users[id]
	if !ok {
		return store.AdminUserDetail{}, store.ErrNotFound
	}
	return store.AdminUserDetail{AdminUser: f.adminRow(u), PaddleCustomerID: u.PaddleCustomerID, SubscriptionID: u.SubscriptionID}, nil
}

type fakeMail struct {
	mu   sync.Mutex
	sent []sentMail
}

type sentMail struct{ to, subject, body string }

func (m *fakeMail) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMail{to, subject, body})
	return nil
}

func (m *fakeMail) last() sentMail {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return sentMail{}
	}
	return m.sent[len(m.sent)-1]
}

func (f *fakeStore) UserByOAuthIdentity(_ context.Context, provider, subject string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.identities[provider+"/"+subject]; ok {
		return f.users[id], nil
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeStore) CreateOAuthUser(ctx context.Context, email, provider, subject string) (store.User, error) {
	u, err := f.CreateUser(ctx, email, "")
	if err != nil {
		return u, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.identities == nil {
		f.identities = map[string]string{}
	}
	f.identities[provider+"/"+subject] = u.ID
	now := time.Now()
	u.EmailVerifiedAt = &now
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) UserByEmail(_ context.Context, email string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeStore) CreateSession(_ context.Context, h, userID string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[h] = userID
	return nil
}

func (f *fakeStore) UserBySession(_ context.Context, h string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.sessions[h]; ok {
		return f.users[id], nil
	}
	return store.User{}, store.ErrNotFound
}

func (f *fakeStore) DeleteSession(_ context.Context, h string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, h)
	return nil
}

func (f *fakeStore) ApplyPaddleEvent(_ context.Context, eventID, _ string, sub *store.Subscription) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.events[eventID] {
		return nil
	}
	if sub != nil {
		u, ok := f.users[sub.UserID]
		if !ok {
			return store.ErrNotFound
		}
		if store.SubscriptionApplies(u.SubscriptionID, u.SubscriptionStatus, sub) {
			u.Plan, u.SubscriptionStatus, u.PaddleCustomerID, u.SubscriptionID = sub.Plan, sub.Status, sub.CustomerID, sub.SubscriptionID
			f.users[sub.UserID] = u
		}
	}
	f.events[eventID] = true
	return nil
}

const testIP = "192.0.2.1"

const (
	testPlan     = "test_subscription"
	testLifetime = "test_lifetime"
)

func init() {
	plans[testPlan] = Plan{ID: testPlan, Billing: BillingSubscription}
	plans[testLifetime] = Plan{ID: testLifetime, Billing: BillingOneTime}
}

func (f *fakeStore) ApplyPaddlePurchase(_ context.Context, eventID, _ string, p store.Purchase) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.events[eventID] {
		return nil
	}
	u, ok := f.users[p.UserID]
	if !ok {
		return store.ErrNotFound
	}
	u.LifetimePlan = p.Plan
	f.users[p.UserID] = u
	f.events[eventID] = true
	return nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }

func newTestService(f *fakeStore, c *clock, paddle Paddle) *Service {
	return New(f, Config{Now: c.now, Paddle: paddle})
}

func (f *fakeStore) SaveFeedback(_ context.Context, userID, email, message, page string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row := store.Feedback{ID: f.id(), Email: email, Message: message, Page: page, CreatedAt: time.Now()}
	if userID != "" {
		row.UserID = &userID
	}
	f.feedback = append(f.feedback, row)
	return nil
}

func (f *fakeStore) AdminFeedback(_ context.Context, limit, offset int) ([]store.Feedback, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.adminLimit, f.adminOffset = limit, offset
	out := []store.Feedback{}
	for i := len(f.feedback) - 1; i >= 0; i-- {
		out = append(out, f.feedback[i])
	}
	return out, len(out), nil
}
