package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"saas-starter/api/internal/store"
)

const paddleAPI = "https://api.paddle.com"

type PortalLinks struct {
	Overview      string `json:"overview"`
	Cancel        string `json:"cancel"`
	UpdatePayment string `json:"update_payment"`
}

func (s *Service) BillingPortal(ctx context.Context, u store.User) (PortalLinks, error) {
	var out PortalLinks
	if s.paddle.APIKey == "" {
		return out, ErrBillingDisabled
	}
	if u.PaddleCustomerID == "" || u.SubscriptionID == "" {
		return out, ErrNoSubscription
	}
	body, err := json.Marshal(map[string][]string{"subscription_ids": {u.SubscriptionID}})
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, paddleAPI+"/customers/"+url.PathEscape(u.PaddleCustomerID)+"/portal-sessions", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Authorization", "Bearer "+s.paddle.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := s.paddle.client().Do(req)
	if err != nil {
		return out, fmt.Errorf("paddle portal: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return out, fmt.Errorf("paddle portal: http %d: %.300s", res.StatusCode, raw)
	}
	var parsed struct {
		Data struct {
			URLs struct {
				General struct {
					Overview string `json:"overview"`
				} `json:"general"`
				Subscriptions []struct {
					ID                              string `json:"id"`
					CancelSubscription              string `json:"cancel_subscription"`
					UpdateSubscriptionPaymentMethod string `json:"update_subscription_payment_method"`
				} `json:"subscriptions"`
			} `json:"urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return out, fmt.Errorf("paddle portal: %w", err)
	}
	out.Overview = parsed.Data.URLs.General.Overview
	for _, sub := range parsed.Data.URLs.Subscriptions {
		if sub.ID == u.SubscriptionID {
			out.Cancel, out.UpdatePayment = sub.CancelSubscription, sub.UpdateSubscriptionPaymentMethod
		}
	}
	if out.Overview == "" {
		return out, fmt.Errorf("paddle portal: no links in the answer")
	}
	return out, nil
}
