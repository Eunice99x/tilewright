package service

import (
	"context"
	"slices"
	"time"

	"saas-starter/api/internal/store"
)

type fakeGames struct {
	rows  map[string]store.Game
	seats []fakeSeat
}

type fakeSeat struct {
	gameID, userID string
	seat           int
}

func (f *fakeStore) games() *fakeGames {
	if f.gameRows == nil {
		f.gameRows = &fakeGames{rows: map[string]store.Game{}}
	}
	return f.gameRows
}

func (f *fakeStore) CreateGuest(_ context.Context, email, name string) (store.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := store.User{ID: f.id(), Email: email, Plan: "free", DisplayName: name, CreatedAt: time.Now()}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) CreateGame(_ context.Context, userID, mode, state string, code *string) (store.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if mode == "solo" {
		for _, g := range f.games().rows {
			if g.UserID == userID && g.Mode == "solo" && !g.Over {
				return store.Game{}, store.ErrGameInProgress
			}
		}
	}
	now := time.Now()
	g := store.Game{ID: f.id(), UserID: userID, Mode: mode, State: state, InviteCode: code, CreatedAt: now, UpdatedAt: now}
	f.games().rows[g.ID] = g
	f.games().seats = append(f.games().seats, fakeSeat{g.ID, userID, 0})
	return g, nil
}

func (f *fakeStore) ActiveSoloGame(_ context.Context, userID string) (store.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, g := range f.games().rows {
		if g.UserID == userID && g.Mode == "solo" && !g.Over {
			return g, nil
		}
	}
	return store.Game{}, store.ErrNotFound
}

func (f *fakeStore) GameByInvite(_ context.Context, code string) (store.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, g := range f.games().rows {
		if g.InviteCode != nil && *g.InviteCode == code {
			return g, nil
		}
	}
	return store.Game{}, store.ErrNotFound
}

func (f *fakeStore) AddPlayer(_ context.Context, gameID, userID string, seat int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.games().seats {
		if s.gameID == gameID && (s.seat == seat || s.userID == userID) {
			return store.ErrGameFull
		}
	}
	f.games().seats = append(f.games().seats, fakeSeat{gameID, userID, seat})
	return nil
}

func (f *fakeStore) PlayerGame(_ context.Context, userID, id string) (store.PlayerGame, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.games().seats {
		if s.gameID == id && s.userID == userID {
			return f.playerGame(s), nil
		}
	}
	return store.PlayerGame{}, store.ErrNotFound
}

func (f *fakeStore) PlayerGames(_ context.Context, userID string, limit int) ([]store.PlayerGame, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []store.PlayerGame{}
	for _, s := range f.games().seats {
		if s.userID == userID {
			out = append(out, f.playerGame(s))
		}
	}
	slices.SortFunc(out, func(a, b store.PlayerGame) int {
		if a.Over != b.Over {
			if a.Over {
				return 1
			}
			return -1
		}
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})
	return out[:min(limit, len(out))], nil
}

func (f *fakeStore) playerGame(s fakeSeat) store.PlayerGame {
	pg := store.PlayerGame{Game: f.games().rows[s.gameID], Seat: s.seat}
	for _, o := range f.games().seats {
		if o.gameID == s.gameID && o.userID != s.userID {
			if u := f.users[o.userID]; u.DisplayName != "" {
				pg.Opponent = u.DisplayName
			} else {
				pg.Opponent = u.Email
			}
		}
	}
	return pg
}

func (f *fakeStore) SaveGame(_ context.Context, g store.Game) (store.Game, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.games().rows[g.ID]
	if !ok || cur.Version != g.Version {
		return store.Game{}, store.ErrGameChanged
	}
	g.Version++
	g.UpdatedAt = time.Now()
	f.games().rows[g.ID] = g
	return g, nil
}

func (f *fakeStore) BestGames(_ context.Context, userID string, limit int) ([]store.GameSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []store.GameSummary{}
	for _, g := range f.games().rows {
		if g.UserID == userID && g.Mode == "solo" && g.Over {
			out = append(out, store.GameSummary{ID: g.ID, Score: g.Score, FinishedAt: g.UpdatedAt})
		}
	}
	slices.SortFunc(out, func(a, b store.GameSummary) int { return b.Score - a.Score })
	return out[:min(limit, len(out))], nil
}
