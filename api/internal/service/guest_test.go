package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGuests(t *testing.T) {
	ctx := context.Background()

	t.Run("a guest gets a session and keeps their name", func(t *testing.T) {
		f := newFakeStore()
		svc := New(f, Config{})

		token, u, err := svc.GuestSignup(ctx, "1.2.3.4", "  Sam   the  Great ")
		if err != nil {
			t.Fatal(err)
		}
		if token == "" || !u.Guest || u.Name != "Sam the Great" {
			t.Errorf("got token %q user %+v", token, u)
		}
		if !strings.HasSuffix(u.Email, "@guest.invalid") {
			t.Errorf("got email %q", u.Email)
		}
	})

	t.Run("bad names are refused", func(t *testing.T) {
		svc := New(newFakeStore(), Config{})
		for _, name := range []string{"", "   ", "sam@example.com", strings.Repeat("x", 25), "bell\ahere", "nul\x00"} {
			if _, _, err := svc.GuestSignup(ctx, "1.2.3.4", name); !errors.Is(err, ErrInvalid) {
				t.Errorf("%q: got %v want %v", name, err, ErrInvalid)
			}
		}
	})

	t.Run("one IP can't make endless guests", func(t *testing.T) {
		svc := New(newFakeStore(), Config{})
		var err error
		for i := 0; i <= guestIPLimit; i++ {
			_, _, err = svc.GuestSignup(ctx, "1.2.3.4", "Sam")
		}
		if !errors.Is(err, ErrTooManyAttempts) {
			t.Errorf("got %v want %v", err, ErrTooManyAttempts)
		}
	})

	t.Run("other players see the guest's name", func(t *testing.T) {
		f := newFakeStore()
		svc := New(f, Config{Shuffle: func(int, func(i, j int)) {}})
		host, _ := f.CreateUser(ctx, "host@example.com", "hash")
		g, _ := svc.StartVersus(ctx, host)

		_, guest, _ := svc.GuestSignup(ctx, "1.2.3.4", "Sam")
		gu := f.users[guest.ID]
		joined, err := svc.JoinGame(ctx, gu, g.InviteCode)
		if err != nil {
			t.Fatal(err)
		}
		if joined.Opponent != "host" {
			t.Errorf("guest sees %q", joined.Opponent)
		}
		hostView, _ := svc.Game(ctx, host, g.ID)
		if hostView.Opponent != "Sam" {
			t.Errorf("host sees %q", hostView.Opponent)
		}
	})
}
