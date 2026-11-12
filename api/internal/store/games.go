package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"

	"github.com/lib/pq"
)

var (
	ErrGameChanged    = errors.New("game_changed")
	ErrGameInProgress = errors.New("game_in_progress")
	ErrGameFull       = errors.New("game_full")
)

type Game struct {
	ID         string    `db:"id"`
	UserID     string    `db:"user_id"`
	Mode       string    `db:"mode"`
	State      string    `db:"state"`
	Score      int       `db:"score"`
	Over       bool      `db:"over"`
	InviteCode *string   `db:"invite_code"`
	Version    int       `db:"version"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type PlayerGame struct {
	Game
	Seat     int    `db:"seat"`
	Opponent string `db:"opponent"`
}

type GameSummary struct {
	ID         string    `db:"id" json:"id"`
	Score      int       `db:"score" json:"score"`
	Turns      int       `db:"turns" json:"turns"`
	FinishedAt time.Time `db:"finished_at" json:"finished_at"`
}

const gameCols = `g.id, g.user_id, g.mode, g.state, g.score, g.over, g.invite_code, g.version, g.created_at, g.updated_at`

func (s *Store) CreateGuest(ctx context.Context, email, name string) (User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, `INSERT INTO users (email, password_hash, display_name) VALUES ($1, '', $2) RETURNING `+userCols, email, name)
	if isUniqueViolation(err) {
		return u, ErrEmailTaken
	}
	return u, err
}

func (s *Store) CreateGame(ctx context.Context, userID, mode, state string, inviteCode *string) (Game, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return Game{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	var g Game
	err = tx.GetContext(ctx, &g, `INSERT INTO games AS g (user_id, mode, state, invite_code) VALUES ($1, $2, $3::jsonb, $4) RETURNING `+gameCols,
		userID, mode, state, inviteCode)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Constraint == "games_one_solo" {
			return Game{}, ErrGameInProgress
		}
		return Game{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO game_players (game_id, user_id, seat) VALUES ($1, $2, 0)`, g.ID, userID); err != nil {
		return Game{}, err
	}
	return g, tx.Commit()
}

func (s *Store) ActiveSoloGame(ctx context.Context, userID string) (Game, error) {
	var g Game
	err := s.db.GetContext(ctx, &g, `SELECT `+gameCols+` FROM games g WHERE g.user_id = $1 AND g.mode = 'solo' AND NOT g.over`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

func (s *Store) GameByInvite(ctx context.Context, code string) (Game, error) {
	var g Game
	err := s.db.GetContext(ctx, &g, `SELECT `+gameCols+` FROM games g WHERE g.invite_code = $1`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return g, ErrNotFound
	}
	return g, err
}

func (s *Store) AddPlayer(ctx context.Context, gameID, userID string, seat int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO game_players (game_id, user_id, seat) VALUES ($1, $2, $3)`, gameID, userID, seat)
	if isUniqueViolation(err) {
		return ErrGameFull
	}
	if err == nil {
		_, err = s.db.ExecContext(ctx, `UPDATE games SET updated_at = now() WHERE id = $1`, gameID)
	}
	return err
}

func (s *Store) PlayerGame(ctx context.Context, userID, id string) (PlayerGame, error) {
	if !uuidPattern.MatchString(id) {
		return PlayerGame{}, ErrNotFound
	}
	var rows []PlayerGame
	if err := s.db.SelectContext(ctx, &rows, playerGamesQuery+` AND g.id = $2`, userID, id); err != nil {
		return PlayerGame{}, err
	}
	if len(rows) == 0 {
		return PlayerGame{}, ErrNotFound
	}
	return rows[0], nil
}

func (s *Store) PlayerGames(ctx context.Context, userID string, limit int) ([]PlayerGame, error) {
	rows := []PlayerGame{}
	err := s.db.SelectContext(ctx, &rows, playerGamesQuery+` ORDER BY g.over, g.updated_at DESC LIMIT $2`, userID, limit)
	return rows, err
}

const playerGamesQuery = `SELECT ` + gameCols + `, me.seat,
	COALESCE((SELECT COALESCE(NULLIF(u.display_name, ''), u.email) FROM game_players op JOIN users u ON u.id = op.user_id
		WHERE op.game_id = g.id AND op.user_id <> me.user_id LIMIT 1), '') AS opponent
	FROM game_players me JOIN games g ON g.id = me.game_id
	WHERE me.user_id = $1`

func (s *Store) SaveGame(ctx context.Context, g Game) (Game, error) {
	var saved Game
	err := s.db.GetContext(ctx, &saved, `UPDATE games AS g SET state = $1::jsonb, score = $2, over = $3, version = g.version + 1, updated_at = now()
		WHERE g.id = $4 AND g.version = $5 RETURNING `+gameCols,
		g.State, g.Score, g.Over, g.ID, g.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrGameChanged
	}
	return saved, err
}

func (s *Store) BestGames(ctx context.Context, userID string, limit int) ([]GameSummary, error) {
	out := []GameSummary{}
	err := s.db.SelectContext(ctx, &out, `SELECT id, score, (state->>'turn')::int AS turns, updated_at AS finished_at
		FROM games WHERE user_id = $1 AND mode = 'solo' AND over ORDER BY score DESC, updated_at DESC LIMIT $2`, userID, limit)
	return out, err
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
