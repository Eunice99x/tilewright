package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"saas-starter/api/internal/game"
	"saas-starter/api/internal/store"
)

var (
	ErrGameChanged    = store.ErrGameChanged
	ErrGameInProgress = store.ErrGameInProgress
	ErrGameFull       = store.ErrGameFull
)

const (
	bestGamesShown = 10
	gamesListed    = 50
)

type GameView struct {
	ID         string `json:"id"`
	Mode       string `json:"mode"`
	Seat       int    `json:"seat"`
	Opponent   string `json:"opponent,omitempty"`
	Waiting    bool   `json:"waiting"`
	InviteCode string `json:"invite_code,omitempty"`
	game.View
	Last *game.Move `json:"last,omitempty"`
}

type GameListItem struct {
	ID         string    `json:"id"`
	Mode       string    `json:"mode"`
	Seat       int       `json:"seat"`
	Opponent   string    `json:"opponent,omitempty"`
	Waiting    bool      `json:"waiting"`
	InviteCode string    `json:"invite_code,omitempty"`
	Scores     []int     `json:"scores"`
	ToMove     int       `json:"to_move"`
	Turn       int       `json:"turn"`
	Over       bool      `json:"over"`
	Winner     int       `json:"winner"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Service) StartGame(ctx context.Context, u store.User) (GameView, error) {
	row, err := s.store.ActiveSoloGame(ctx, u.ID)
	if errors.Is(err, store.ErrNotFound) {
		row, err = s.newGame(ctx, u, "solo", nil)
		if errors.Is(err, store.ErrGameInProgress) {
			row, err = s.store.ActiveSoloGame(ctx, u.ID)
		}
	}
	if err != nil {
		return GameView{}, err
	}
	return s.Game(ctx, u, row.ID)
}

func (s *Service) StartVersus(ctx context.Context, u store.User) (GameView, error) {
	code, err := inviteCode()
	if err != nil {
		return GameView{}, err
	}
	row, err := s.newGame(ctx, u, "versus", &code)
	if err != nil {
		return GameView{}, err
	}
	return s.Game(ctx, u, row.ID)
}

func (s *Service) JoinGame(ctx context.Context, u store.User, code string) (GameView, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" || len(code) > 32 {
		return GameView{}, ErrNotFound
	}
	row, err := s.store.GameByInvite(ctx, code)
	if err != nil {
		return GameView{}, err
	}
	if _, err := s.store.PlayerGame(ctx, u.ID, row.ID); err == nil {
		return s.Game(ctx, u, row.ID)
	}
	if row.Over {
		return GameView{}, game.ErrGameOver
	}
	if err := s.store.AddPlayer(ctx, row.ID, u.ID, 1); err != nil {
		return GameView{}, err
	}
	return s.Game(ctx, u, row.ID)
}

func (s *Service) Game(ctx context.Context, u store.User, id string) (GameView, error) {
	row, err := s.store.PlayerGame(ctx, u.ID, id)
	if err != nil {
		return GameView{}, err
	}
	g, err := decodeGame(row.Game)
	if err != nil {
		return GameView{}, err
	}
	return viewOf(row, g, nil), nil
}

func (s *Service) Games(ctx context.Context, u store.User) ([]GameListItem, error) {
	rows, err := s.store.PlayerGames(ctx, u.ID, gamesListed)
	if err != nil {
		return nil, err
	}
	out := make([]GameListItem, 0, len(rows))
	for _, row := range rows {
		g, err := decodeGame(row.Game)
		if err != nil {
			return nil, err
		}
		v := viewOf(row, g, nil)
		out = append(out, GameListItem{
			ID: v.ID, Mode: v.Mode, Seat: v.Seat, Opponent: v.Opponent, Waiting: v.Waiting, InviteCode: v.InviteCode,
			Scores: v.Scores, ToMove: v.ToMove, Turn: v.Turn, Over: v.Over, Winner: v.Winner, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Service) PlayTiles(ctx context.Context, u store.User, id string, ps []game.Placement) (GameView, error) {
	return s.updateGame(ctx, u, id, func(g *game.Game, seat int) error {
		_, err := g.Play(seat, ps, s.words())
		return err
	})
}

func (s *Service) SwapTiles(ctx context.Context, u store.User, id, letters string) (GameView, error) {
	return s.updateGame(ctx, u, id, func(g *game.Game, seat int) error {
		return g.Swap(seat, letters, s.shuffle)
	})
}

func (s *Service) PassTurn(ctx context.Context, u store.User, id string) (GameView, error) {
	return s.updateGame(ctx, u, id, func(g *game.Game, seat int) error {
		return g.Pass(seat)
	})
}

func (s *Service) EndGame(ctx context.Context, u store.User, id string) (GameView, error) {
	return s.updateGame(ctx, u, id, func(g *game.Game, seat int) error {
		if g.Solo() {
			return g.End(seat)
		}
		return g.Resign(seat)
	})
}

func (s *Service) BestGames(ctx context.Context, u store.User) ([]store.GameSummary, error) {
	return s.store.BestGames(ctx, u.ID, bestGamesShown)
}

func (s *Service) newGame(ctx context.Context, u store.User, mode string, code *string) (store.Game, error) {
	players := 1
	if mode == "versus" {
		players = 2
	}
	state, err := json.Marshal(game.New(players, s.shuffle))
	if err != nil {
		return store.Game{}, err
	}
	return s.store.CreateGame(ctx, u.ID, mode, string(state), code)
}

func (s *Service) updateGame(ctx context.Context, u store.User, id string, move func(g *game.Game, seat int) error) (GameView, error) {
	row, err := s.store.PlayerGame(ctx, u.ID, id)
	if err != nil {
		return GameView{}, err
	}
	g, err := decodeGame(row.Game)
	if err != nil {
		return GameView{}, err
	}
	movesBefore := len(g.Moves)
	if err := move(g, row.Seat); err != nil {
		return GameView{}, err
	}

	state, err := json.Marshal(g)
	if err != nil {
		return GameView{}, err
	}
	row.State, row.Score, row.Over = string(state), g.Seats[0].Score, g.Over
	saved, err := s.store.SaveGame(ctx, row.Game)
	if err != nil {
		return GameView{}, err
	}
	row.Game = saved

	var last *game.Move
	for i := len(g.Moves) - 1; i >= movesBefore; i-- {
		if g.Moves[i].Kind != "left" {
			last = &g.Moves[i]
			break
		}
	}
	return viewOf(row, g, last), nil
}

func viewOf(row store.PlayerGame, g *game.Game, last *game.Move) GameView {
	v := GameView{ID: row.ID, Mode: row.Mode, Seat: row.Seat, Opponent: displayName(row.Opponent), View: g.View(row.Seat), Last: last}
	if row.Mode == "versus" && row.Opponent == "" {
		v.Waiting = true
		if row.Seat == 0 && row.InviteCode != nil {
			v.InviteCode = *row.InviteCode
		}
	}
	return v
}

func displayName(email string) string {
	name, _, _ := strings.Cut(email, "@")
	return name
}

func decodeGame(row store.Game) (*game.Game, error) {
	var g game.Game
	if err := json.Unmarshal([]byte(row.State), &g); err != nil {
		return nil, fmt.Errorf("game %s: bad state: %w", row.ID, err)
	}
	if len(g.Seats) == 0 {
		return nil, fmt.Errorf("game %s: no seats", row.ID)
	}
	return &g, nil
}

func inviteCode() (string, error) {
	b := make([]byte, 7)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))[:10], nil
}

func (s *Service) words() game.Dictionary {
	if s.dict != nil {
		return s.dict
	}
	return game.Words()
}
