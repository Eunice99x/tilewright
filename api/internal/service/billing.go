package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"saas-starter/api/internal/store"
)

type Paddle struct {
	WebhookSecret string
	PricePlans    map[string]string
	APIKey        string
	Client        *http.Client
}

func (p Paddle) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *Service) HandlePaddleWebhook(ctx context.Context, body []byte, signature string) error {
	if s.paddle.WebhookSecret == "" {
		return ErrBillingDisabled
	}
	if err := verifyPaddleSignature(s.paddle.WebhookSecret, body, signature, s.now()); err != nil {
		return err
	}

	var ev struct {
		EventID    string    `json:"event_id"`
		EventType  string    `json:"event_type"`
		OccurredAt time.Time `json:"occurred_at"`
		Data       struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			CustomerID string `json:"customer_id"`
			CustomData struct {
				UserID  string `json:"user_id"`
				Product string `json:"product"`
			} `json:"custom_data"`
			Items []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &ev); err != nil || ev.EventID == "" {
		return ErrInvalid
	}

	if ev.Data.CustomData.Product != App.Product {
		return nil
	}

	if ev.EventType == "transaction.completed" {
		plan := ""
		for _, it := range ev.Data.Items {
			if p, ok := s.paddle.PricePlans[it.Price.ID]; ok {
				plan = p
			}
		}

		if plan == "" {
			return fmt.Errorf("paddle event %s: transaction %s has an unknown price (%d known)", ev.EventID, ev.Data.ID, len(s.paddle.PricePlans))
		}

		if plans[plan].Billing == BillingOneTime {
			return s.applyPurchase(ctx, ev.EventID, ev.EventType, ev.Data.CustomData.UserID, plan, ev.Data.ID)
		}
	}

	var sub *store.Subscription
	if strings.HasPrefix(ev.EventType, "subscription.") {
		d := ev.Data
		plan := ""
		for _, it := range d.Items {
			if p, ok := s.paddle.PricePlans[it.Price.ID]; ok {
				plan = p
			}
		}

		if !uuidRe.MatchString(d.CustomData.UserID) {
			return fmt.Errorf("paddle event %s: subscription %s has no valid user_id", ev.EventID, d.ID)
		}
		if plan == "" {
			return fmt.Errorf("paddle event %s: subscription %s has an unknown price (%d known)", ev.EventID, d.ID, len(s.paddle.PricePlans))
		}
		sub = &store.Subscription{
			UserID: d.CustomData.UserID, Plan: plan, Status: d.Status,
			CustomerID: d.CustomerID, SubscriptionID: d.ID, OccurredAt: ev.OccurredAt,
		}
	}
	err := s.store.ApplyPaddleEvent(ctx, ev.EventID, ev.EventType, sub)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("paddle event %s: user %s doesn't exist", ev.EventID, sub.UserID)
	}
	return err
}

func (s *Service) applyPurchase(ctx context.Context, eventID, eventType, userID, plan, transactionID string) error {

	if !uuidRe.MatchString(userID) {
		return fmt.Errorf("paddle event %s: transaction %s has no valid user_id", eventID, transactionID)
	}
	err := s.store.ApplyPaddlePurchase(ctx, eventID, eventType, store.Purchase{UserID: userID, Plan: plan, TransactionID: transactionID})
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("paddle event %s: user %s doesn't exist", eventID, userID)
	}
	return err
}

func verifyPaddleSignature(secret string, body []byte, header string, now time.Time) error {
	if secret == "" {
		return errors.New("PADDLE_WEBHOOK_SECRET is not set")
	}
	var ts string
	var h1s []string
	for _, part := range strings.Split(header, ";") {
		k, v, _ := strings.Cut(part, "=")
		switch k {
		case "ts":
			ts = v
		case "h1":
			h1s = append(h1s, v)
		}
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || len(h1s) == 0 {
		return ErrBadSignature
	}
	if d := now.Sub(time.Unix(sec, 0)); d > 5*time.Minute || d < -5*time.Minute {
		return ErrBadSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":"))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	for _, h1 := range h1s {
		if hmac.Equal([]byte(want), []byte(h1)) {
			return nil
		}
	}
	return ErrBadSignature
}
