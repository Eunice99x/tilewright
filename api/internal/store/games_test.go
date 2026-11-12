package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestGamesIntegration(t *testing.T) {
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

	newUser := func(t testing.TB, name string) User {
		t.Helper()
		u, err := s.CreateUser(ctx, fmt.Sprintf("%s-%d@example.com", name, time.Now().UnixNano()), "hash")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Exec(`DELETE FROM users WHERE id = $1`, u.ID) })
		return u
	}

	t.Run("solo: one in progress, saves with versions, best list", func(t *testing.T) {
		u := newUser(t, "solo")
		g, err := s.CreateGame(ctx, u.ID, "solo", `{"turn": 0}`, nil)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := s.CreateGame(ctx, u.ID, "solo", `{}`, nil); !errors.Is(err, ErrGameInProgress) {
			t.Errorf("second solo game: got %v want %v", err, ErrGameInProgress)
		}
		if got, err := s.ActiveSoloGame(ctx, u.ID); err != nil || got.ID != g.ID {
			t.Errorf("active: got %v %v", got.ID, err)
		}
		if _, err := s.PlayerGame(ctx, u.ID, "not-a-uuid"); !errors.Is(err, ErrNotFound) {
			t.Errorf("bad id: got %v want %v", err, ErrNotFound)
		}

		stale := g
		g.State, g.Score, g.Over = `{"turn": 3}`, 57, true
		saved, err := s.SaveGame(ctx, g)
		if err != nil || saved.Version != g.Version+1 {
			t.Fatalf("save: %v, version %d", err, saved.Version)
		}
		if _, err := s.SaveGame(ctx, stale); !errors.Is(err, ErrGameChanged) {
			t.Errorf("stale save: got %v want %v", err, ErrGameChanged)
		}

		best, err := s.BestGames(ctx, u.ID, 10)
		if err != nil || len(best) != 1 || best[0].Score != 57 || best[0].Turns != 3 {
			t.Errorf("best: got %+v %v", best, err)
		}
	})

	t.Run("versus: invite, join, seats and opponents", func(t *testing.T) {
		host, friend, third := newUser(t, "host"), newUser(t, "friend"), newUser(t, "third")
		code := fmt.Sprintf("c%d", time.Now().UnixNano())
		g, err := s.CreateGame(ctx, host.ID, "versus", `{"turn": 0}`, &code)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateGame(ctx, host.ID, "versus", `{}`, nil); err != nil {
			t.Errorf("a second versus game should be fine: %v", err)
		}

		found, err := s.GameByInvite(ctx, code)
		if err != nil || found.ID != g.ID {
			t.Fatalf("by invite: got %v %v", found.ID, err)
		}

		if err := s.AddPlayer(ctx, g.ID, friend.ID, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.AddPlayer(ctx, g.ID, third.ID, 1); !errors.Is(err, ErrGameFull) {
			t.Errorf("third player: got %v want %v", err, ErrGameFull)
		}

		mine, err := s.PlayerGame(ctx, friend.ID, g.ID)
		if err != nil || mine.Seat != 1 || mine.Opponent != host.Email {
			t.Errorf("friend's view: got seat %d opponent %q %v", mine.Seat, mine.Opponent, err)
		}
		if _, err := s.PlayerGame(ctx, third.ID, g.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("outsider: got %v want %v", err, ErrNotFound)
		}

		list, err := s.PlayerGames(ctx, host.ID, 10)
		if err != nil || len(list) != 2 {
			t.Fatalf("host's list: got %d games %v", len(list), err)
		}
	})
}
