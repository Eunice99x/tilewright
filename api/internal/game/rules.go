package game

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed rules.json
var rulesJSON []byte

const (
	Size   = 11
	Centre = Size / 2
	empty  = '.'
)

var (
	RackSize        int
	MaxTurns        int
	ScorelessRounds int
	AllTilesBonus   int

	Layout [Size]string

	letterValues [26]int
	letterCounts [26]int
)

func init() {
	var r struct {
		RackSize      int      `json:"rack_size"`
		MaxTurns      int      `json:"max_turns"`
		Scoreless     int      `json:"scoreless_rounds"`
		AllTilesBonus int      `json:"all_tiles_bonus"`
		Layout        []string `json:"layout"`
		Tiles         map[string]struct {
			Value int `json:"value"`
			Count int `json:"count"`
		} `json:"tiles"`
	}
	if err := json.Unmarshal(rulesJSON, &r); err != nil {
		panic(fmt.Sprintf("game: rules.json: %v", err))
	}
	if len(r.Layout) != Size || len(r.Tiles) != 26 || r.RackSize < 1 || r.MaxTurns < 1 || r.Scoreless < 1 {
		panic("game: rules.json needs an 11-row layout, all 26 letters, a rack size and a turn count")
	}
	for i, row := range r.Layout {
		if len(row) != Size {
			panic(fmt.Sprintf("game: rules.json layout row %d is not %d squares", i, Size))
		}
		Layout[i] = row
	}
	for letter, t := range r.Tiles {
		if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
			panic(fmt.Sprintf("game: rules.json: bad letter %q", letter))
		}
		letterValues[letter[0]-'A'] = t.Value
		letterCounts[letter[0]-'A'] = t.Count
	}
	RackSize, MaxTurns, ScorelessRounds, AllTilesBonus = r.RackSize, r.MaxTurns, r.Scoreless, r.AllTilesBonus
}

func value(letter byte) int {
	return letterValues[letter-'A']
}

type RuleError string

func (e RuleError) Error() string {
	return string(e)
}

const (
	ErrGameOver     = RuleError("game_over")
	ErrNoTiles      = RuleError("no_tiles")
	ErrBadLetter    = RuleError("bad_letter")
	ErrNotInRack    = RuleError("not_in_rack")
	ErrOffBoard     = RuleError("off_board")
	ErrSquareTaken  = RuleError("square_taken")
	ErrNotInLine    = RuleError("not_in_line")
	ErrGap          = RuleError("gap_in_word")
	ErrStartSquare  = RuleError("must_cover_start")
	ErrNotConnected = RuleError("not_connected")
	ErrTooShort     = RuleError("word_too_short")
	ErrBagTooSmall  = RuleError("bag_too_small")
	ErrNotYourTurn  = RuleError("not_your_turn")
	ErrSoloOnly     = RuleError("solo_only")
	ErrVersusOnly   = RuleError("versus_only")
)

type NotWordError struct {
	Words []string
}

func (e *NotWordError) Error() string {
	return "not_a_word"
}
