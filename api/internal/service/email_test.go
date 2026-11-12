package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"saas-starter/api/internal/store"
)

func mailService(f *fakeStore) (*Service, *fakeMail) {
	m := &fakeMail{}
	return New(f, Config{Mail: m, SiteURL: "https://app.example/", ForgotTime: -1}), m
}

func linkToken(t *testing.T, body, path string) string {
	t.Helper()
	at := strings.Index(body, "https://app.example"+path+"?token=")
	if at < 0 {
		t.Fatalf("no %s link in %q", path, body)
	}
	raw, _, _ := strings.Cut(body[at:], "\n")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("token")
}

func TestSignupNeedsConfirmation(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, m := mailService(f)
	_, u, err := s.Signup(ctx, testIP, "jane@example.com", "longenough")
	if err != nil || u.EmailVerified {
		t.Fatalf("signup: %+v, %v", u, err)
	}
	token := linkToken(t, m.last().body, "/verify")
	if strings.Contains(hashToken(token), token) || f.tokens[hashToken(token)].userID != u.ID {
		t.Error("the token should be stored only as a hash")
	}
	other, err := f.CreateUser(ctx, "someone@example.com", "h")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyEmail(ctx, other, token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("another account: %v", err)
	}
	if err := s.VerifyEmail(ctx, f.users[u.ID], token); err != nil {
		t.Fatal(err)
	}
	if !userView(f.users[u.ID]).EmailVerified {
		t.Error("not confirmed after the link")
	}
	if err := s.VerifyEmail(ctx, f.users[u.ID], token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("a used link: %v", err)
	}
	if err := s.VerifyEmail(ctx, f.users[u.ID], "made-up"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("a made-up link: %v", err)
	}
}

func TestResendVerificationIsLimited(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, m := mailService(f)
	_, u, _ := s.Signup(ctx, testIP, "jane@example.com", "longenough")
	first := linkToken(t, m.last().body, "/verify")
	var err error
	sends := 0
	for sends < 20 {
		if err = s.ResendVerification(ctx, f.users[u.ID]); err != nil {
			break
		}
		sends++
	}
	if !errors.Is(err, ErrTooManyAttempts) || sends == 0 || sends >= 20 {
		t.Errorf("after %d sends: %v", sends, err)
	}
	if err := s.VerifyEmail(ctx, f.users[u.ID], first); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("an older link should stop working: %v", err)
	}
}

func TestPasswordReset(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, m := mailService(f)
	session, u, err := s.Signup(ctx, testIP, "jane@example.com", "old-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPasswordReset(ctx, testIP, " JANE@example.com "); err != nil {
		t.Fatal(err)
	}
	token := linkToken(t, m.last().body, "/reset")
	if err := s.ResetPassword(ctx, testIP, token, "short"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("weak: %v", err)
	}
	if err := s.ResetPassword(ctx, testIP, token, "new-password"); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, testIP, token, "another-password"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("a used link: %v", err)
	}
	if _, err := s.UserFromSession(ctx, session); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("an old session survived: %v", err)
	}
	if _, _, err := s.Login(ctx, testIP, "jane@example.com", "old-password"); !errors.Is(err, ErrBadLogin) {
		t.Errorf("the old password: %v", err)
	}
	_, after, err := s.Login(ctx, testIP, "jane@example.com", "new-password")
	if err != nil || after.ID != u.ID || !after.EmailVerified {
		t.Errorf("the new password: %+v, %v", after, err)
	}
}

func TestResendReportsAFailedSend(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s := New(f, Config{Mail: failingMail{}})
	u, err := f.CreateUser(ctx, "jane@example.com", "h")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ResendVerification(ctx, u); !errors.Is(err, ErrEmailFailed) {
		t.Errorf("got %v", err)
	}
}

type failingMail struct{}

func (failingMail) Send(context.Context, string, string, string) error {
	return errors.New("provider down")
}

func TestResendMailerWithoutKeyIsANoOp(t *testing.T) {
	called := false
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("no request should be made")
	})}
	if err := (ResendMailer{Client: client}).Send(context.Background(), "jane@example.com", "s", "b"); err != nil {
		t.Errorf("got %v", err)
	}
	if called {
		t.Error("called the provider without a key")
	}
	f := newFakeStore()
	u, _ := f.CreateUser(context.Background(), "jane@example.com", "h")
	if err := New(f, Config{}).ResendVerification(context.Background(), u); err != nil {
		t.Errorf("resend with no key: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestForgotTakesTheSameTime(t *testing.T) {
	f := newFakeStore()
	s := New(f, Config{Mail: &fakeMail{}, ForgotTime: 80 * time.Millisecond})
	start := time.Now()
	if err := s.RequestPasswordReset(context.Background(), testIP, "nobody@example.com"); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d < 80*time.Millisecond {
		t.Errorf("an unknown address answered in %v", d)
	}
}

func TestResetLinkExpires(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, m := mailService(f)
	if _, _, err := s.Signup(ctx, testIP, "jane@example.com", "old-password"); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPasswordReset(ctx, testIP, "jane@example.com"); err != nil {
		t.Fatal(err)
	}
	h := hashToken(linkToken(t, m.last().body, "/reset"))
	if life := time.Until(f.tokens[h].expires); life > resetLinkLife || life < resetLinkLife-time.Minute {
		t.Errorf("reset link lasts %v", life)
	}
}

func TestResetDoesNotRevealAccounts(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	s, m := mailService(f)
	if err := s.RequestPasswordReset(ctx, testIP, "nobody@example.com"); err != nil {
		t.Errorf("unknown address: %v", err)
	}
	if len(m.sent) != 0 {
		t.Errorf("sent %d emails to an unknown address", len(m.sent))
	}
	if err := s.RequestPasswordReset(ctx, testIP, "not an email"); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("bad address: %v", err)
	}
	if _, err := f.CreateOAuthUser(ctx, "oauth@example.com", "google", "1"); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPasswordReset(ctx, testIP, "oauth@example.com"); err != nil {
		t.Fatal(err)
	}
	if b := m.last().body; strings.Contains(b, "token=") || !strings.Contains(b, "/login") {
		t.Errorf("an OAuth account should be pointed to the sign-in buttons: %q", b)
	}

	for range 30 {
		if err := s.RequestPasswordReset(ctx, "198.51.100."+"7", "oauth@example.com"); err != nil && !errors.Is(err, ErrTooManyAttempts) {
			t.Fatal(err)
		}
	}
	if n := len(m.sent); n > 6 {
		t.Errorf("%d emails to one address", n)
	}
}

type portalTransport struct {
	status int
	body   string
	req    *http.Request
	sent   string
}

func (p *portalTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	p.req = r
	b, _ := io.ReadAll(r.Body)
	p.sent = string(b)
	return &http.Response{StatusCode: p.status, Body: io.NopCloser(strings.NewReader(p.body)), Header: http.Header{}, Request: r}, nil
}

func TestBillingPortal(t *testing.T) {
	ctx := context.Background()
	tr := &portalTransport{status: 200, body: `{"data":{"urls":{"general":{"overview":"https://p/over"},"subscriptions":[
		{"id":"sub_other","cancel_subscription":"https://p/no","update_subscription_payment_method":"https://p/no"},
		{"id":"sub_1","cancel_subscription":"https://p/cancel","update_subscription_payment_method":"https://p/card"}]}}}`}
	s := New(newFakeStore(), Config{Paddle: Paddle{APIKey: "k", Client: &http.Client{Transport: tr}}})
	u := store.User{PaddleCustomerID: "ctm_1", SubscriptionID: "sub_1"}
	links, err := s.BillingPortal(ctx, u)
	if err != nil || links.Cancel != "https://p/cancel" || links.UpdatePayment != "https://p/card" || links.Overview != "https://p/over" {
		t.Fatalf("got %+v, %v", links, err)
	}
	if got := tr.req.URL.String(); got != "https://api.paddle.com/customers/ctm_1/portal-sessions" {
		t.Errorf("url %s", got)
	}
	if tr.req.Header.Get("Authorization") != "Bearer k" || !strings.Contains(tr.sent, "sub_1") {
		t.Errorf("request: %v %s", tr.req.Header, tr.sent)
	}
	if _, err := s.BillingPortal(ctx, store.User{}); !errors.Is(err, ErrNoSubscription) {
		t.Errorf("no subscription: %v", err)
	}
	tr.status, tr.body = 403, `{"error":"forbidden"}`
	if _, err := s.BillingPortal(ctx, u); err == nil || errors.Is(err, ErrNoSubscription) {
		t.Errorf("paddle refusal: %v", err)
	}
	off := New(newFakeStore(), Config{})
	if _, err := off.BillingPortal(ctx, u); !errors.Is(err, ErrBillingDisabled) {
		t.Errorf("no key: %v", err)
	}
}
