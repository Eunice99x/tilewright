package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAttempts(t *testing.T) {
	a := newAttempts(3, time.Minute)
	now := time.Unix(1_800_000_000, 0)
	for i := range 3 {
		if a.Blocked("k", now) {
			t.Fatalf("blocked before attempt %d", i)
		}
		if !a.Hit("k", now) {
			t.Fatalf("attempt %d blocked", i)
		}
	}
	if !a.Blocked("k", now) || a.Hit("k", now) {
		t.Error("4th attempt allowed")
	}
	if a.Blocked("other", now) || !a.Hit("other", now) {
		t.Error("limits must be per key")
	}
	if a.Hit("k", now.Add(59*time.Second)) {
		t.Error("allowed before the window ended")
	}
	if a.Blocked("k", now.Add(time.Minute)) || !a.Hit("k", now.Add(time.Minute)) {
		t.Error("still blocked after the window")
	}
}

func TestAttemptsSweep(t *testing.T) {
	a := newAttempts(3, time.Minute)
	now := time.Unix(1_800_000_000, 0)
	for i := range 100 {
		a.Hit(fmt.Sprint("ip", i), now)
	}
	a.Hit("late", now.Add(30*time.Second))
	a.Hit("new", now.Add(2*time.Minute))
	if len(a.counts) != 1 {
		t.Errorf("after sweep: %d entries, want 1", len(a.counts))
	}
}

func TestLoginLimits(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	s := newTestService(f, c, Paddle{})
	s.Signup(ctx, testIP, "victim@example.com", "longenough")

	for i := range loginEmailLimit {
		if _, _, err := s.Login(ctx, fmt.Sprint("198.51.100.", i), "Victim@example.com ", "wrongpass"); !errors.Is(err, ErrBadLogin) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, _, err := s.Login(ctx, "198.51.100.200", "victim@example.com", "longenough"); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("over the email limit, right password: %v", err)
	}
	c.add(loginWindow)
	if _, _, err := s.Login(ctx, "198.51.100.200", "victim@example.com", "longenough"); err != nil {
		t.Errorf("after the window: %v", err)
	}

	for i := range loginIPLimit {
		if _, _, err := s.Login(ctx, "203.0.113.9", fmt.Sprint("u", i, "@example.com"), "wrongpass"); !errors.Is(err, ErrBadLogin) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, _, err := s.Login(ctx, "203.0.113.9", "another@example.com", "wrongpass"); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("over the IP limit: %v", err)
	}
	if _, _, err := s.Login(ctx, "203.0.113.10", "another@example.com", "wrongpass"); !errors.Is(err, ErrBadLogin) {
		t.Errorf("another IP is unaffected: %v", err)
	}
}

func TestSignupLimit(t *testing.T) {
	ctx := context.Background()
	f := newFakeStore()
	c := &clock{time.Unix(1_800_000_000, 0)}
	s := newTestService(f, c, Paddle{})
	signup := func(ip string, i int) error {
		_, _, err := s.Signup(ctx, ip, fmt.Sprint("u", i, "@example.com"), "longenough")
		return err
	}

	for range signupIPLimit + 1 {
		if _, _, err := s.Signup(ctx, testIP, "x@example.com", "short"); !errors.Is(err, ErrWeakPassword) {
			t.Fatal(err)
		}
	}
	for i := range signupIPLimit {
		if err := signup(testIP, i); err != nil {
			t.Fatalf("signup %d: %v", i, err)
		}
	}
	if err := signup(testIP, 100); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("over the limit: %v", err)
	}
	if err := signup("203.0.113.9", 101); err != nil {
		t.Errorf("another IP: %v", err)
	}
	c.add(signupWindow)
	if err := signup(testIP, 102); err != nil {
		t.Errorf("next day: %v", err)
	}
}
