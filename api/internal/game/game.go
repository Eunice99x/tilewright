package game

import (
	"bytes"
	"strings"
)

type Game struct {
	Board     [Size]string `json:"board"`
	Bag       string       `json:"bag"`
	Seats     []Seat       `json:"seats"`
	ToMove    int          `json:"to_move"`
	Turn      int          `json:"turn"`
	Scoreless int          `json:"scoreless"`
	Moves     []Move       `json:"moves"`
	Over      bool         `json:"over"`
	Resigned  int          `json:"resigned"`
}

type Seat struct {
	Rack  string `json:"rack"`
	Score int    `json:"score"`
}

type Move struct {
	Seat   int      `json:"seat"`
	Kind   string   `json:"kind"`
	Words  []string `json:"words,omitempty"`
	Tiles  int      `json:"tiles,omitempty"`
	Points int      `json:"points"`
}

type Placement struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Letter string `json:"letter"`
}

type Shuffler func(n int, swap func(i, j int))

func New(players int, shuffle Shuffler) *Game {
	players = min(max(players, 1), 2)
	bag := make([]byte, 0, 80)
	for i, n := range letterCounts {
		for range n {
			bag = append(bag, byte('A'+i))
		}
	}
	shuffle(len(bag), func(i, j int) { bag[i], bag[j] = bag[j], bag[i] })

	g := &Game{Bag: string(bag), Seats: make([]Seat, players), Moves: []Move{}, Resigned: -1}
	for r := range g.Board {
		g.Board[r] = strings.Repeat(string(empty), Size)
	}
	for s := range g.Seats {
		g.draw(s)
	}
	return g
}

func (g *Game) Solo() bool {
	return len(g.Seats) == 1
}

func (g *Game) Play(seat int, ps []Placement, dict Dictionary) (Move, error) {
	if err := g.canMove(seat); err != nil {
		return Move{}, err
	}
	if len(ps) == 0 || len(ps) > RackSize {
		return Move{}, ErrNoTiles
	}

	board := g.grid()
	rack := []byte(g.Seats[seat].Rack)
	isNew := map[[2]int]bool{}

	for _, p := range ps {
		letter, ok := upper(p.Letter)
		if !ok {
			return Move{}, ErrBadLetter
		}
		if p.Row < 0 || p.Row >= Size || p.Col < 0 || p.Col >= Size {
			return Move{}, ErrOffBoard
		}
		if board[p.Row][p.Col] != empty {
			return Move{}, ErrSquareTaken
		}
		i := bytes.IndexByte(rack, letter)
		if i < 0 {
			return Move{}, ErrNotInRack
		}
		rack = append(rack[:i], rack[i+1:]...)
		board[p.Row][p.Col] = letter
		isNew[[2]int{p.Row, p.Col}] = true
	}

	across, down := true, true
	minR, maxR, minC, maxC := ps[0].Row, ps[0].Row, ps[0].Col, ps[0].Col
	for _, p := range ps[1:] {
		if p.Row != ps[0].Row {
			across = false
		}
		if p.Col != ps[0].Col {
			down = false
		}
		minR, maxR = min(minR, p.Row), max(maxR, p.Row)
		minC, maxC = min(minC, p.Col), max(maxC, p.Col)
	}
	if !across && !down {
		return Move{}, ErrNotInLine
	}

	for r := minR; r <= maxR; r++ {
		for c := minC; c <= maxC; c++ {
			if board[r][c] == empty {
				return Move{}, ErrGap
			}
		}
	}

	if g.Board[Centre][Centre] == empty {
		if !isNew[[2]int{Centre, Centre}] {
			return Move{}, ErrStartSquare
		}
		if len(ps) < 2 {
			return Move{}, ErrTooShort
		}
	} else if !g.touches(ps) {
		return Move{}, ErrNotConnected
	}

	var found []string
	points := 0
	addWord := func(r, c, dr, dc int) {
		word, pts := wordAt(board, isNew, r, c, dr, dc)
		if len(word) >= 2 {
			found = append(found, word)
			points += pts
		}
	}

	switch {
	case len(ps) == 1:
		addWord(ps[0].Row, ps[0].Col, 0, 1)
		addWord(ps[0].Row, ps[0].Col, 1, 0)
	case across:
		addWord(ps[0].Row, ps[0].Col, 0, 1)
		for _, p := range ps {
			addWord(p.Row, p.Col, 1, 0)
		}
	default:
		addWord(ps[0].Row, ps[0].Col, 1, 0)
		for _, p := range ps {
			addWord(p.Row, p.Col, 0, 1)
		}
	}
	if len(found) == 0 {
		return Move{}, ErrTooShort
	}

	var bad []string
	for _, w := range found {
		if !dict.Has(w) {
			bad = append(bad, w)
		}
	}
	if len(bad) > 0 {
		return Move{}, &NotWordError{Words: bad}
	}

	if len(ps) == RackSize {
		points += AllTilesBonus
	}

	for r := range g.Board {
		g.Board[r] = string(board[r])
	}
	g.Seats[seat].Rack = string(rack)
	g.draw(seat)
	g.Seats[seat].Score += points
	g.Scoreless = 0
	move := Move{Seat: seat, Kind: "play", Words: found, Points: points}
	g.Moves = append(g.Moves, move)
	g.afterTurn(seat)
	return move, nil
}

func (g *Game) Swap(seat int, letters string, shuffle Shuffler) error {
	if err := g.canMove(seat); err != nil {
		return err
	}
	letters = strings.ToUpper(letters)
	n := len(letters)
	if n == 0 || n > RackSize {
		return ErrNoTiles
	}
	if len(g.Bag) < n {
		return ErrBagTooSmall
	}

	rack := []byte(g.Seats[seat].Rack)
	for i := 0; i < n; i++ {
		j := bytes.IndexByte(rack, letters[i])
		if j < 0 {
			return ErrNotInRack
		}
		rack = append(rack[:j], rack[j+1:]...)
	}

	drawn := g.Bag[:n]
	bag := []byte(g.Bag[n:] + letters)
	shuffle(len(bag), func(i, j int) { bag[i], bag[j] = bag[j], bag[i] })

	g.Seats[seat].Rack = string(rack) + drawn
	g.Bag = string(bag)
	g.Scoreless++
	g.Moves = append(g.Moves, Move{Seat: seat, Kind: "swap", Tiles: n})
	g.afterTurn(seat)
	return nil
}

func (g *Game) Pass(seat int) error {
	if err := g.canMove(seat); err != nil {
		return err
	}
	g.Scoreless++
	g.Moves = append(g.Moves, Move{Seat: seat, Kind: "pass"})
	g.afterTurn(seat)
	return nil
}

func (g *Game) End(seat int) error {
	if !g.Solo() {
		return ErrSoloOnly
	}
	if err := g.canMove(seat); err != nil {
		return err
	}
	g.Over = true
	g.Moves = append(g.Moves, Move{Seat: seat, Kind: "end"})
	return nil
}

func (g *Game) Resign(seat int) error {
	if g.Solo() {
		return ErrVersusOnly
	}
	if g.Over {
		return ErrGameOver
	}
	if seat < 0 || seat >= len(g.Seats) {
		return ErrNotYourTurn
	}
	g.Over = true
	g.Resigned = seat
	g.Moves = append(g.Moves, Move{Seat: seat, Kind: "resign"})
	return nil
}

func (g *Game) Winner() int {
	if !g.Over || g.Solo() {
		return -1
	}
	if g.Resigned >= 0 {
		return 1 - g.Resigned
	}
	switch a, b := g.Seats[0].Score, g.Seats[1].Score; {
	case a > b:
		return 0
	case b > a:
		return 1
	}
	return -1
}

type View struct {
	Board     []string `json:"board"`
	Rack      string   `json:"rack"`
	BagLeft   int      `json:"bag_left"`
	Scores    []int    `json:"scores"`
	ToMove    int      `json:"to_move"`
	Turn      int      `json:"turn"`
	TurnsLeft int      `json:"turns_left"`
	Over      bool     `json:"over"`
	Winner    int      `json:"winner"`
	Resigned  int      `json:"resigned"`
	Moves     []Move   `json:"moves"`
}

func (g *Game) View(seat int) View {
	moves := g.Moves
	if moves == nil {
		moves = []Move{}
	}
	scores := make([]int, len(g.Seats))
	for i, s := range g.Seats {
		scores[i] = s.Score
	}
	v := View{
		Board:    g.Board[:],
		BagLeft:  len(g.Bag),
		Scores:   scores,
		ToMove:   g.ToMove,
		Turn:     g.Turn,
		Over:     g.Over,
		Winner:   g.Winner(),
		Resigned: g.Resigned,
		Moves:    moves,
	}
	if seat >= 0 && seat < len(g.Seats) {
		v.Rack = g.Seats[seat].Rack
	}
	if g.Solo() {
		v.TurnsLeft = MaxTurns - g.Turn
	}
	return v
}

func (g *Game) canMove(seat int) error {
	if g.Over {
		return ErrGameOver
	}
	if seat != g.ToMove || seat < 0 || seat >= len(g.Seats) {
		return ErrNotYourTurn
	}
	return nil
}

func (g *Game) afterTurn(seat int) {
	g.Turn++
	out := len(g.Bag) == 0 && len(g.Seats[seat].Rack) == 0
	switch {
	case g.Solo():
		g.Over = g.Turn >= MaxTurns || out
	case out:
		g.finish(seat)
	case g.Scoreless >= ScorelessRounds*len(g.Seats):
		g.finish(-1)
	}
	if !g.Over {
		g.ToMove = (g.ToMove + 1) % len(g.Seats)
	}
}

func (g *Game) finish(out int) {
	for i, s := range g.Seats {
		if i == out {
			continue
		}
		left := 0
		for j := 0; j < len(s.Rack); j++ {
			left += value(s.Rack[j])
		}
		if left == 0 {
			continue
		}
		g.Seats[i].Score -= left
		g.Moves = append(g.Moves, Move{Seat: i, Kind: "left", Points: -left})
		if out >= 0 {
			g.Seats[out].Score += left
			g.Moves = append(g.Moves, Move{Seat: out, Kind: "left", Points: left})
		}
	}
	g.Over = true
}

func (g *Game) draw(seat int) {
	n := min(RackSize-len(g.Seats[seat].Rack), len(g.Bag))
	g.Seats[seat].Rack += g.Bag[:n]
	g.Bag = g.Bag[n:]
}

func (g *Game) touches(ps []Placement) bool {
	for _, p := range ps {
		for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			r, c := p.Row+d[0], p.Col+d[1]
			if r >= 0 && r < Size && c >= 0 && c < Size && g.Board[r][c] != empty {
				return true
			}
		}
	}
	return false
}

func (g *Game) grid() [][]byte {
	b := make([][]byte, Size)
	for r := range b {
		b[r] = []byte(g.Board[r])
	}
	return b
}

func wordAt(board [][]byte, isNew map[[2]int]bool, r, c, dr, dc int) (string, int) {
	for r-dr >= 0 && c-dc >= 0 && board[r-dr][c-dc] != empty {
		r, c = r-dr, c-dc
	}

	var word []byte
	sum, mult := 0, 1
	for r < Size && c < Size && board[r][c] != empty {
		letter := board[r][c]
		v := value(letter)
		if isNew[[2]int{r, c}] {
			switch Layout[r][c] {
			case 'd':
				v *= 2
			case 't':
				v *= 3
			case 'D':
				mult *= 2
			case 'T':
				mult *= 3
			}
		}
		word = append(word, letter)
		sum += v
		r, c = r+dr, c+dc
	}
	return string(word), sum * mult
}

func upper(s string) (byte, bool) {
	if len(s) != 1 {
		return 0, false
	}
	l := s[0]
	if l >= 'a' && l <= 'z' {
		l -= 'a' - 'A'
	}
	return l, l >= 'A' && l <= 'Z'
}
