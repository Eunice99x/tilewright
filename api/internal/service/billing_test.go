package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"saas-starter/api/internal/store"
)

func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":"))
	mac.Write(body)
	return fmt.Sprintf("ts=%s;h1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyPaddleSignature(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ts := fmt.Sprint(now.Unix())
	body := []byte(`{"event_id":"evt_1"}`)

	if err := verifyPaddleSignature("secret", body, sign("secret", ts, body), now); err != nil {
		t.Errorf("valid signature rejected: %v", err)
	}
	if err := verifyPaddleSignature("secret", []byte(`{"event_id":"evt_2"}`), sign("secret", ts, body), now); err == nil {
		t.Error("changed body accepted")
	}
	if err := verifyPaddleSignature("secret", body, sign("other", ts, body), now); err == nil {
		t.Error("wrong secret accepted")
	}
	if err := verifyPaddleSignature("secret", body, sign("secret", ts, body), now.Add(10*time.Minute)); err == nil {
		t.Error("old webhook (replay) accepted")
	}
	if err := verifyPaddleSignature("secret", body, "garbage", now); err == nil {
		t.Error("garbage header accepted")
	}
}

func TestPaddleWebhookSetsPlan(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	ts := fmt.Sprint(c.t.Unix())

	off := newTestService(f, c, Paddle{})
	if err := off.HandlePaddleWebhook(ctx, []byte(`{}`), ""); !errors.Is(err, ErrBillingDisabled) {
		t.Errorf("no secret: got %v", err)
	}

	s := newTestService(f, c, Paddle{WebhookSecret: "whsec", PricePlans: map[string]string{"pri_paid": testPlan}})
	_, u, _ := s.Signup(ctx, testIP, "pay@example.com", "longenough")
	body := []byte(fmt.Sprintf(`{"event_id":"evt_1","event_type":"subscription.activated","occurred_at":"2027-01-15T08:00:00Z",
		"data":{"id":"sub_1","status":"active","customer_id":"ctm_1","custom_data":{"user_id":%q,"product":%q},"items":[{"price":{"id":"pri_paid"}}]}}`, u.ID, App.Product))

	if err := s.HandlePaddleWebhook(ctx, body, sign("wrong", ts, body)); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("bad signature: got %v", err)
	}
	if f.users[u.ID].Plan != "free" {
		t.Fatal("plan changed by an unsigned webhook")
	}
	if err := s.HandlePaddleWebhook(ctx, body, sign("whsec", ts, body)); err != nil {
		t.Fatal(err)
	}
	if got := currentPlan(f.users[u.ID]); got.ID != testPlan {
		t.Errorf("plan after webhook = %+v", got)
	}
}

func TestVerifyPaddleSignatureRotation(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	ts := fmt.Sprint(now.Unix())
	body := []byte(`{"event_id":"evt_1"}`)
	h1 := func(secret string) string { return strings.SplitN(sign(secret, ts, body), ";", 2)[1] }

	for _, header := range []string{
		"ts=" + ts + ";" + h1("old") + ";" + h1("new"),
		"ts=" + ts + ";" + h1("new") + ";" + h1("old"),
	} {
		if err := verifyPaddleSignature("new", body, header, now); err != nil {
			t.Errorf("%s: %v", header, err)
		}
	}
	header := "ts=" + ts + ";" + h1("other") + ";" + h1("another")
	if err := verifyPaddleSignature("new", body, header, now); !errors.Is(err, ErrBadSignature) {
		t.Errorf("no matching h1: %v", err)
	}
}

func paddleEvent(s *Service, c *clock, id, subID, status, userID, price string) error {
	body := []byte(fmt.Sprintf(`{"event_id":%q,"event_type":"subscription.updated","occurred_at":"2027-01-15T08:00:00Z",
		"data":{"id":%q,"status":%q,"customer_id":"ctm_1","custom_data":{"user_id":%q,"product":%q},"items":[{"price":{"id":%q}}]}}`, id, subID, status, userID, App.Product, price))
	return s.HandlePaddleWebhook(context.Background(), body, sign("whsec", fmt.Sprint(c.t.Unix()), body))
}

func internal500(err error) bool {
	return err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrInvalid) && !errors.Is(err, ErrBadSignature)
}

func TestPaddleWebhookUnplaceableEventsAreRetried(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	s := newTestService(f, c, Paddle{WebhookSecret: "whsec", PricePlans: map[string]string{"pri_paid": testPlan}})
	_, u, _ := s.Signup(ctx, testIP, "pay@example.com", "longenough")

	cases := []struct{ name, event, user, price string }{
		{"unknown price", "evt_price", u.ID, "pri_new"},
		{"unknown user", "evt_user", "00000000-0000-4000-8000-999999999999", "pri_paid"},
		{"bad user id", "evt_bad", "not-a-uuid", "pri_paid"},
		{"no user id", "evt_none", "", "pri_paid"},
	}
	for _, tc := range cases {
		if err := paddleEvent(s, c, tc.event, "sub_1", "active", tc.user, tc.price); !internal500(err) {
			t.Errorf("%s: got %v, want an error that becomes a 500", tc.name, err)
		}
		if f.events[tc.event] {
			t.Errorf("%s: event recorded, so a retry would be dropped", tc.name)
		}
	}

	s.paddle.PricePlans["pri_new"] = testPlan
	if err := paddleEvent(s, c, "evt_price", "sub_1", "active", u.ID, "pri_new"); err != nil {
		t.Fatal(err)
	}
	if got := currentPlan(f.users[u.ID]); got.ID != testPlan {
		t.Errorf("after retry: %+v", got)
	}

	body := []byte(fmt.Sprintf(`{"event_id":"evt_tx","event_type":"transaction.updated","data":{"custom_data":{"product":%q}}}`, App.Product))
	if err := s.HandlePaddleWebhook(ctx, body, sign("whsec", fmt.Sprint(c.t.Unix()), body)); err != nil || !f.events["evt_tx"] {
		t.Errorf("other event type: %v, recorded %v", err, f.events["evt_tx"])
	}
}

func TestPaddleWebhookSubscriptionOwnership(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	s := newTestService(f, c, Paddle{WebhookSecret: "whsec", PricePlans: map[string]string{"pri_paid": testPlan}})
	_, u, _ := s.Signup(ctx, testIP, "own@example.com", "longenough")

	send := func(id, subID, status string) {
		t.Helper()
		if err := paddleEvent(s, c, id, subID, status, u.ID, "pri_paid"); err != nil {
			t.Fatal(err)
		}
		if !f.events[id] {
			t.Errorf("%s not recorded", id)
		}
	}
	state := func() (string, string) { return f.users[u.ID].SubscriptionID, f.users[u.ID].SubscriptionStatus }

	send("e1", "sub_1", "canceled")
	if id, _ := state(); id != "" {
		t.Errorf("cancel set subscription %q", id)
	}
	send("e2", "sub_1", "active")
	send("e3", "sub_old", "canceled")
	send("e4", "sub_old", "past_due")
	send("e5", "sub_old", "paused")
	send("e6", "sub_2", "active")
	if id, st := state(); id != "sub_1" || st != "active" || currentPlan(f.users[u.ID]).ID != testPlan {
		t.Errorf("after other subscriptions: %s %s", id, st)
	}
	send("e7", "sub_1", "canceled")
	if id, st := state(); id != "sub_1" || st != "canceled" || currentPlan(f.users[u.ID]).ID != "free" {
		t.Errorf("after own cancel: %s %s", id, st)
	}
	send("e8", "sub_2", "active")
	if id, st := state(); id != "sub_2" || st != "active" || currentPlan(f.users[u.ID]).ID != testPlan {
		t.Errorf("after resubscribing: %s %s", id, st)
	}
}

func TestWebhookIgnoresOtherProducts(t *testing.T) {

	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	svc := newTestService(f, c, Paddle{WebhookSecret: "whsec", PricePlans: map[string]string{"pri_paid": testPlan}})
	body := []byte(`{"event_id":"evt_other","event_type":"subscription.activated","occurred_at":"2027-01-15T08:00:00Z",
		"data":{"id":"sub_x","status":"active","customer_id":"ctm_x","custom_data":{"user_id":"not-ours"},"items":[{"price":{"id":"pri_unknown"}}]}}`)
	if err := svc.HandlePaddleWebhook(context.Background(), body, sign("whsec", fmt.Sprint(c.t.Unix()), body)); err != nil {
		t.Fatalf("another product's event should be ignored, got %v", err)
	}
}

func TestUserViewShowsEffectivePlan(t *testing.T) {

	u := userFromPlan(testPlan, "canceled")
	if got := userView(u).Plan; got != "free" {
		t.Errorf("canceled: plan %q, want free", got)
	}
	if got := userView(userFromPlan(testPlan, "active")).Plan; got != testPlan {
		t.Errorf("active: plan %q, want %s", got, testPlan)
	}
}

func userFromPlan(plan, status string) store.User {
	return store.User{Plan: plan, SubscriptionStatus: status}
}

func purchase(s *Service, c *clock, eventID, userID, price, secret string) error {
	body := []byte(fmt.Sprintf(`{"event_id":%q,"event_type":"transaction.completed","occurred_at":"2027-01-15T08:00:00Z",
		"data":{"id":"txn_1","status":"completed","customer_id":"ctm_1","custom_data":{"user_id":%q,"product":%q},"items":[{"price":{"id":%q}}]}}`, eventID, userID, App.Product, price))
	return s.HandlePaddleWebhook(context.Background(), body, sign(secret, fmt.Sprint(c.t.Unix()), body))
}

func oneTimeService() (*Service, *fakeStore, *clock, string) {
	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	s := newTestService(f, c, Paddle{WebhookSecret: "whsec", APIKey: "k", PricePlans: map[string]string{"pri_once": testLifetime, "pri_paid": testPlan}})
	_, u, _ := s.Signup(context.Background(), testIP, "once@example.com", "longenough")
	return s, f, c, u.ID
}

func TestOneTimePurchaseGivesLifetimeAccess(t *testing.T) {
	s, f, c, id := oneTimeService()
	if err := purchase(s, c, "evt_bad", id, "pri_once", "wrong"); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("bad signature: %v", err)
	}
	if got := currentPlan(f.users[id]); got.ID != "free" {
		t.Fatalf("plan changed by an unsigned webhook: %+v", got)
	}
	if err := purchase(s, c, "evt_1", id, "pri_once", "whsec"); err != nil {
		t.Fatal(err)
	}
	if got := userView(f.users[id]).Plan; got != testLifetime {
		t.Errorf("after purchase: %q", got)
	}

	c.add(10 * 365 * 24 * time.Hour)
	if got := currentPlan(f.users[id]); got.ID != testLifetime {
		t.Errorf("years later: %+v", got)
	}

	if _, err := s.BillingPortal(context.Background(), f.users[id]); !errors.Is(err, ErrNoSubscription) {
		t.Errorf("portal: %v", err)
	}
}

func TestOneTimeRepeatDeliveryIsHarmless(t *testing.T) {
	s, f, c, id := oneTimeService()
	for range 3 {
		if err := purchase(s, c, "evt_1", id, "pri_once", "whsec"); err != nil {
			t.Fatal(err)
		}
	}
	if got := currentPlan(f.users[id]); got.ID != testLifetime {
		t.Errorf("after repeats: %+v", got)
	}

	u := f.users[id]
	u.LifetimePlan = ""
	f.users[id] = u
	if err := purchase(s, c, "evt_1", id, "pri_once", "whsec"); err != nil {
		t.Fatal(err)
	}
	if got := currentPlan(f.users[id]); got.ID != "free" {
		t.Errorf("a repeat restored revoked access: %+v", got)
	}
}

func TestOneTimeUnplaceableEventsAreRetried(t *testing.T) {
	s, f, c, id := oneTimeService()
	for name, tc := range map[string]struct{ user, price string }{
		"unknown price": {id, "pri_new"},
		"unknown user":  {"00000000-0000-4000-8000-999999999999", "pri_once"},
		"bad user id":   {"not-a-uuid", "pri_once"},
	} {
		event := "evt_" + name
		if err := purchase(s, c, event, tc.user, tc.price, "whsec"); !internal500(err) {
			t.Errorf("%s: got %v, want an error that becomes a 500", name, err)
		}
		if f.events[event] {
			t.Errorf("%s: event recorded, so a retry would be dropped", name)
		}
	}
}

func TestSubscriptionTransactionGivesNoLifetimeAccess(t *testing.T) {
	s, f, c, id := oneTimeService()
	if err := purchase(s, c, "evt_sub_txn", id, "pri_paid", "whsec"); err != nil {
		t.Fatal(err)
	}
	if u := f.users[id]; u.LifetimePlan != "" || currentPlan(u).ID != "free" {
		t.Errorf("after a subscription's transaction: %+v", u)
	}
	if !f.events["evt_sub_txn"] {
		t.Error("not recorded")
	}
}

func TestLifetimeOutlivesASubscription(t *testing.T) {
	s, f, c, id := oneTimeService()
	if err := purchase(s, c, "evt_1", id, "pri_once", "whsec"); err != nil {
		t.Fatal(err)
	}
	if err := paddleEvent(s, c, "e1", "sub_1", "active", id, "pri_paid"); err != nil {
		t.Fatal(err)
	}
	if got := currentPlan(f.users[id]); got.ID != testPlan {
		t.Errorf("a live subscription wins: %+v", got)
	}
	if err := paddleEvent(s, c, "e2", "sub_1", "canceled", id, "pri_paid"); err != nil {
		t.Fatal(err)
	}
	if got := currentPlan(f.users[id]); got.ID != testLifetime {
		t.Errorf("after cancelling, back to lifetime: %+v", got)
	}
}

func TestOneTimeIgnoresOtherProducts(t *testing.T) {
	s, f, c, id := oneTimeService()
	body := []byte(fmt.Sprintf(`{"event_id":"evt_x","event_type":"transaction.completed","data":{"id":"txn_x","custom_data":{"user_id":%q,"product":"another-app"},"items":[{"price":{"id":"pri_other"}}]}}`, id))
	if err := s.HandlePaddleWebhook(context.Background(), body, sign("whsec", fmt.Sprint(c.t.Unix()), body)); err != nil {
		t.Fatal(err)
	}
	if currentPlan(f.users[id]).ID != "free" {
		t.Error("another product's payment gave access")
	}
}

func TestAppJSONPlansAreSellable(t *testing.T) {
	for _, p := range App.Plans {
		if p.ID != "free" && p.Billing != BillingSubscription && p.Billing != BillingOneTime {
			t.Errorf("%s: billing %q", p.ID, p.Billing)
		}
	}
}
