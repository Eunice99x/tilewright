package service

import (
	_ "embed"
	"encoding/json"

	"saas-starter/api/internal/store"
)

//go:embed app.json
var appJSON []byte

const (
	BillingSubscription = "subscription"
	BillingOneTime      = "one_time"
)

type Plan struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Price    int      `json:"price"`
	Billing  string   `json:"billing"`
	Features []string `json:"features"`
}

var App = func() (a struct {
	Name         string `json:"name"`
	Product      string `json:"product"`
	SupportEmail string `json:"support_email"`
	Plans        []Plan `json:"plans"`
}) {
	if err := json.Unmarshal(appJSON, &a); err != nil {
		panic("app.json: " + err.Error())
	}
	return a
}()

var plans = func() map[string]Plan {
	m := map[string]Plan{"admin": {ID: "admin", Name: "Admin"}}
	for _, p := range App.Plans {
		if p.ID != "free" && p.Billing != BillingSubscription && p.Billing != BillingOneTime {
			panic(`app.json: plan "` + p.ID + `" needs "billing": "subscription" or "one_time"`)
		}
		m[p.ID] = p
	}
	if _, ok := m["free"]; !ok {
		panic(`app.json: there must be a plan with the id "free"`)
	}
	return m
}()

func PaidPlanIDs() []string {
	var ids []string
	for _, p := range App.Plans {
		if p.ID != "free" {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func currentPlan(u store.User) Plan {
	if u.IsAdmin {
		return plans["admin"]
	}
	if p, ok := plans[u.Plan]; ok && u.Plan != "free" && u.Plan != "admin" {
		switch u.SubscriptionStatus {
		case "active", "trialing", "past_due":
			return p
		}
	}
	if p, ok := plans[u.LifetimePlan]; ok && p.Billing == BillingOneTime {
		return p
	}
	return plans["free"]
}
