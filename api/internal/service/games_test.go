package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"saas-starter/api/internal/game"
	"saas-starter/api/internal/store"
)

func TestGames(t *testing.T) {
	ctx := context.Background()

	newGameService := func(t testing.TB) (*Service, *fakeStore, store.User) {
		t.Helper()
		f := newFakeStore()
		u, err := f.CreateUser(ctx, "player@example.com", "hash")
		if err != nil {
			t.Fatal(err)
		}
		svc := New(f, Config{Words: game.NewWordSet("aa"), Shuffle: func(int, func(i, j int)) {}})
		return svc, f, u
	}

	aa := []game.Placement{{Row: 5, Col: 5, Letter: "A"}, {Row: 5, Col: 6, Letter: "A"}}

	assertError := func(t testing.TB, got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Errorf("got error %v want %v", got, want)
		}
	}

	t.Run("solo start deals a rack and hides the bag", func(t *testing.T) {
		svc, _, u := newGameService(t)
		g, err := svc.StartGame(ctx, u)

		if err != nil {
			t.Fatal(err)
		}
		if g.Rack != "AAAAAAA" || g.Mode != "solo" {
			t.Errorf("got rack %q mode %q", g.Rack, g.Mode)
		}
		if g.BagLeft != 80-game.RackSize {
			t.Errorf("got %d in the bag want %d", g.BagLeft, 80-game.RackSize)
		}
	})

	t.Run("solo start again returns the game in progress", func(t *testing.T) {
		svc, _, u := newGameService(t)
		first, _ := svc.StartGame(ctx, u)
		second, _ := svc.StartGame(ctx, u)

		if first.ID != second.ID {
			t.Errorf("got a second game %s, want %s again", second.ID, first.ID)
		}
	})

	t.Run("a play is saved", func(t *testing.T) {
		svc, _, u := newGameService(t)
		g, _ := svc.StartGame(ctx, u)

		played, err := svc.PlayTiles(ctx, u, g.ID, aa)
		if err != nil {
			t.Fatal(err)
		}
		if played.Scores[0] != 2 || played.Last == nil || played.Last.Points != 2 {
			t.Errorf("got scores %v last %+v, want 2 points", played.Scores, played.Last)
		}

		again, _ := svc.Game(ctx, u, g.ID)
		if !strings.Contains(again.Board[5], "AA") {
			t.Errorf("the play wasn't saved: row 5 is %q", again.Board[5])
		}
	})

	t.Run("a bad word saves nothing", func(t *testing.T) {
		svc, _, u := newGameService(t)
		g, _ := svc.StartGame(ctx, u)

		_, err := svc.PlayTiles(ctx, u, g.ID, append(aa, game.Placement{Row: 5, Col: 7, Letter: "A"}))

		var notWord *game.NotWordError
		if !errors.As(err, &notWord) {
			t.Fatalf("got %v, want a NotWordError", err)
		}
		again, _ := svc.Game(ctx, u, g.ID)
		if again.Turn != 0 {
			t.Errorf("got turn %d want 0", again.Turn)
		}
	})

	t.Run("other players can't see or play your game", func(t *testing.T) {
		svc, f, u := newGameService(t)
		g, _ := svc.StartGame(ctx, u)
		other, _ := f.CreateUser(ctx, "other@example.com", "hash")

		_, err := svc.Game(ctx, other, g.ID)
		assertError(t, err, ErrNotFound)

		_, err = svc.EndGame(ctx, other, g.ID)
		assertError(t, err, ErrNotFound)
	})

	t.Run("a stale save is refused", func(t *testing.T) {
		svc, f, u := newGameService(t)
		g, _ := svc.StartGame(ctx, u)
		row, _ := f.PlayerGame(ctx, u.ID, g.ID)
		svc.EndGame(ctx, u, g.ID)

		_, err := f.SaveGame(ctx, row.Game)
		assertError(t, err, ErrGameChanged)
	})

	t.Run("ended solo games make the best list, and a new one can start", func(t *testing.T) {
		svc, _, u := newGameService(t)
		g, _ := svc.StartGame(ctx, u)
		svc.PlayTiles(ctx, u, g.ID, aa)
		ended, err := svc.EndGame(ctx, u, g.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !ended.Over {
			t.Error("game should be over")
		}

		best, _ := svc.BestGames(ctx, u)
		if len(best) != 1 || best[0].Score != 2 {
			t.Errorf("got best %+v, want one game with 2 points", best)
		}

		next, _ := svc.StartGame(ctx, u)
		if next.ID == g.ID {
			t.Error("start after the end should deal a new game")
		}
	})
}

func TestVersusGames(t *testing.T) {
	ctx := context.Background()

	setup := func(t testing.TB) (*Service, *fakeStore, store.User, store.User) {
		t.Helper()
		f := newFakeStore()
		host, _ := f.CreateUser(ctx, "host@example.com", "hash")
		friend, _ := f.CreateUser(ctx, "friend@example.com", "hash")
		svc := New(f, Config{Words: game.NewWordSet("aa"), Shuffle: func(int, func(i, j int)) {}})
		return svc, f, host, friend
	}

	assertError := func(t testing.TB, got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Errorf("got error %v want %v", got, want)
		}
	}

	t.Run("a new versus game waits with an invite code", func(t *testing.T) {
		svc, _, host, _ := setup(t)
		g, err := svc.StartVersus(ctx, host)

		if err != nil {
			t.Fatal(err)
		}
		if !g.Waiting || len(g.InviteCode) != 10 || g.Mode != "versus" || len(g.Scores) != 2 {
			t.Errorf("got %+v", g)
		}
	})

	t.Run("the friend joins with the code and sees only their own rack", func(t *testing.T) {
		svc, _, host, friend := setup(t)
		g, _ := svc.StartVersus(ctx, host)

		joined, err := svc.JoinGame(ctx, friend, strings.ToUpper(g.InviteCode))
		if err != nil {
			t.Fatal(err)
		}
		if joined.ID != g.ID || joined.Seat != 1 || joined.Waiting || joined.Opponent != "host" {
			t.Errorf("got %+v", joined)
		}
		if joined.Rack == g.Rack {
			t.Error("the friend got the host's rack")
		}

		again, _ := svc.Game(ctx, host, g.ID)
		if again.Waiting || again.InviteCode != "" || again.Opponent != "friend" {
			t.Errorf("host sees waiting=%v code=%q opponent=%q", again.Waiting, again.InviteCode, again.Opponent)
		}
	})

	t.Run("opening the link again, or your own link, just opens the game", func(t *testing.T) {
		svc, _, host, friend := setup(t)
		g, _ := svc.StartVersus(ctx, host)
		svc.JoinGame(ctx, friend, g.InviteCode)

		if again, err := svc.JoinGame(ctx, friend, g.InviteCode); err != nil || again.Seat != 1 {
			t.Errorf("friend again: %v seat %d", err, again.Seat)
		}
		if own, err := svc.JoinGame(ctx, host, g.InviteCode); err != nil || own.Seat != 0 {
			t.Errorf("host: %v seat %d", err, own.Seat)
		}
	})

	t.Run("a third player can't join", func(t *testing.T) {
		svc, f, host, friend := setup(t)
		g, _ := svc.StartVersus(ctx, host)
		svc.JoinGame(ctx, friend, g.InviteCode)
		third, _ := f.CreateUser(ctx, "third@example.com", "hash")

		_, err := svc.JoinGame(ctx, third, g.InviteCode)
		assertError(t, err, ErrGameFull)
	})

	t.Run("a wrong code is not found", func(t *testing.T) {
		svc, _, _, friend := setup(t)
		_, err := svc.JoinGame(ctx, friend, "nope")
		assertError(t, err, ErrNotFound)
	})

	t.Run("players take turns, and resigning ends it", func(t *testing.T) {
		svc, _, host, friend := setup(t)
		g, _ := svc.StartVersus(ctx, host)
		svc.JoinGame(ctx, friend, g.InviteCode)

		_, err := svc.PassTurn(ctx, friend, g.ID)
		assertError(t, err, game.ErrNotYourTurn)

		if _, err := svc.PassTurn(ctx, host, g.ID); err != nil {
			t.Fatal(err)
		}
		if v, err := svc.PassTurn(ctx, friend, g.ID); err != nil || v.ToMove != 0 {
			t.Fatalf("friend's pass: %v, to move %d", err, v.ToMove)
		}

		over, err := svc.EndGame(ctx, friend, g.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !over.Over || over.Winner != 0 {
			t.Errorf("got over=%v winner=%d, want the host to win", over.Over, over.Winner)
		}
	})

	t.Run("the games list shows both games, the one in progress first", func(t *testing.T) {
		svc, _, host, friend := setup(t)
		solo, _ := svc.StartGame(ctx, host)
		svc.EndGame(ctx, host, solo.ID)
		versus, _ := svc.StartVersus(ctx, host)
		svc.JoinGame(ctx, friend, versus.InviteCode)

		list, err := svc.Games(ctx, host)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 2 || list[0].ID != versus.ID || list[0].Opponent != "friend" || !list[1].Over {
			t.Errorf("got %+v", list)
		}

		theirs, _ := svc.Games(ctx, friend)
		if len(theirs) != 1 || theirs[0].Seat != 1 {
			t.Errorf("friend got %+v", theirs)
		}
	})
}
