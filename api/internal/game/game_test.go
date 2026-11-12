package game

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func noShuffle(int, func(i, j int)) {}

func newTestGame(rack, bag string) *Game {
	g := &Game{Seats: []Seat{{Rack: rack}}, Bag: bag, Moves: []Move{}, Resigned: -1}
	for r := range g.Board {
		g.Board[r] = strings.Repeat(".", Size)
	}
	return g
}

func put(g *Game, row, col int, word string) {
	b := []byte(g.Board[row])
	copy(b[col:], word)
	g.Board[row] = string(b)
}

func across(row, col int, letters string) []Placement {
	var ps []Placement
	for i, l := range letters {
		ps = append(ps, Placement{Row: row, Col: col + i, Letter: string(l)})
	}
	return ps
}

func down(row, col int, letters string) []Placement {
	var ps []Placement
	for i, l := range letters {
		ps = append(ps, Placement{Row: row + i, Col: col, Letter: string(l)})
	}
	return ps
}

func TestNew(t *testing.T) {
	g := New(1, noShuffle)

	if len(g.Seats[0].Rack) != RackSize {
		t.Errorf("got rack of %d tiles, want %d", len(g.Seats[0].Rack), RackSize)
	}

	if got := len(g.Seats[0].Rack) + len(g.Bag); got != 80 {
		t.Errorf("got %d tiles in total, want 80", got)
	}

	if g.Board[Centre][Centre] != '.' {
		t.Error("board should start empty")
	}
}

func TestPlay(t *testing.T) {
	dict := NewWordSet("cat", "to", "at", "toaster", "aa", "cats")

	assertPoints := func(t testing.TB, move Move, err error, want int) {
		t.Helper()
		if err != nil {
			t.Fatalf("didn't want an error but got %v", err)
		}
		if move.Points != want {
			t.Errorf("got %d points want %d", move.Points, want)
		}
	}

	assertError := func(t testing.TB, got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Errorf("got error %v want %v", got, want)
		}
	}

	t.Run("first word scores with bonus squares", func(t *testing.T) {
		g := newTestGame("CATXXXX", "")

		move, err := g.Play(0, across(Centre, 3, "CAT"), dict)

		assertPoints(t, move, err, 8)
		if g.Board[Centre] != "...CAT....." {
			t.Errorf("CAT not on the board: %q", g.Board[Centre])
		}
		if g.Seats[0].Score != 8 || g.Turn != 1 {
			t.Errorf("got score %d turn %d, want 8 and 1", g.Seats[0].Score, g.Turn)
		}
	})

	t.Run("first word must cover the start square", func(t *testing.T) {
		g := newTestGame("CATXXXX", "")
		_, err := g.Play(0, across(0, 0, "CAT"), dict)

		assertError(t, err, ErrStartSquare)
	})

	t.Run("first word needs two tiles", func(t *testing.T) {
		g := newTestGame("AXXXXXX", "")
		_, err := g.Play(0, across(Centre, Centre, "A"), dict)

		assertError(t, err, ErrTooShort)
	})

	t.Run("letters must come from the rack", func(t *testing.T) {
		g := newTestGame("CAXXXXX", "")
		_, err := g.Play(0, across(Centre, 3, "CAT"), dict)

		assertError(t, err, ErrNotInRack)
	})

	t.Run("tiles must be in one line", func(t *testing.T) {
		g := newTestGame("CATXXXX", "")
		ps := []Placement{{Centre, Centre, "C"}, {Centre + 1, Centre + 1, "A"}}
		_, err := g.Play(0, ps, dict)

		assertError(t, err, ErrNotInLine)
	})

	t.Run("no gaps inside a word", func(t *testing.T) {
		g := newTestGame("CATXXXX", "")
		ps := []Placement{{Centre, Centre, "C"}, {Centre, Centre + 2, "A"}}
		_, err := g.Play(0, ps, dict)

		assertError(t, err, ErrGap)
	})

	t.Run("unknown words are listed and nothing moves", func(t *testing.T) {
		g := newTestGame("XQZXXXX", "")
		before := *g
		before.Seats = []Seat{g.Seats[0]}
		_, err := g.Play(0, across(Centre, 4, "XQZ"), dict)

		var notWord *NotWordError
		if !errors.As(err, &notWord) {
			t.Fatalf("got %v, want a NotWordError", err)
		}
		if !reflect.DeepEqual(notWord.Words, []string{"XQZ"}) {
			t.Errorf("got %v want [XQZ]", notWord.Words)
		}
		if g.Board != before.Board || g.Seats[0].Rack != before.Seats[0].Rack {
			t.Error("the board or rack changed after a bad move")
		}
	})

	t.Run("later words must touch the board", func(t *testing.T) {
		g := newTestGame("TOXXXXX", "")
		put(g, Centre, 3, "CAT")
		_, err := g.Play(0, across(0, 0, "TO"), dict)

		assertError(t, err, ErrNotConnected)
	})

	t.Run("can't place on a taken square", func(t *testing.T) {
		g := newTestGame("TOXXXXX", "")
		put(g, Centre, 3, "CAT")
		_, err := g.Play(0, across(Centre, 4, "TO"), dict)

		assertError(t, err, ErrSquareTaken)
	})

	t.Run("cross words count too", func(t *testing.T) {
		g := newTestGame("TOXXXXX", "")
		put(g, Centre, 3, "CAT")

		move, err := g.Play(0, across(Centre+1, 4, "TO"), dict)

		assertPoints(t, move, err, 8)
		if !reflect.DeepEqual(move.Words, []string{"TO", "AT", "TO"}) {
			t.Errorf("got words %v", move.Words)
		}
	})

	t.Run("one tile can extend a word", func(t *testing.T) {
		g := newTestGame("SXXXXXX", "")
		put(g, Centre, 3, "CAT")

		move, err := g.Play(0, across(Centre, 6, "S"), dict)

		assertPoints(t, move, err, 6)
	})

	t.Run("words can go down", func(t *testing.T) {
		g := newTestGame("CATXXXX", "")

		move, err := g.Play(0, down(3, Centre, "CAT"), dict)

		assertPoints(t, move, err, 8)
	})

	t.Run("using all seven tiles adds the bonus", func(t *testing.T) {
		g := newTestGame("TOASTER", "")

		move, err := g.Play(0, across(Centre, 2, "TOASTER"), dict)

		assertPoints(t, move, err, 9+AllTilesBonus)
	})

	t.Run("draws back up to seven tiles", func(t *testing.T) {
		g := newTestGame("CATXXXX", "ABCDEFGH")
		g.Play(0, across(Centre, 3, "CAT"), dict)

		if g.Seats[0].Rack != "XXXXABC" {
			t.Errorf("got rack %q want %q", g.Seats[0].Rack, "XXXXABC")
		}
		if g.Bag != "DEFGH" {
			t.Errorf("got bag %q want %q", g.Bag, "DEFGH")
		}
	})

	t.Run("the game ends after the last turn", func(t *testing.T) {
		g := newTestGame("CATXXXX", "")
		g.Turn = MaxTurns - 1
		g.Play(0, across(Centre, 3, "CAT"), dict)

		if !g.Over {
			t.Fatal("game should be over")
		}

		_, err := g.Play(0, across(Centre+1, 3, "AT"), dict)
		assertError(t, err, ErrGameOver)
	})
}

func TestSwap(t *testing.T) {

	t.Run("swaps tiles and uses a turn", func(t *testing.T) {
		g := newTestGame("QXZAEIO", "RSTUVWY")
		err := g.Swap(0, "QXZ", noShuffle)

		if err != nil {
			t.Fatal(err)
		}
		if g.Seats[0].Rack != "AEIORST" {
			t.Errorf("got rack %q want %q", g.Seats[0].Rack, "AEIORST")
		}
		if g.Bag != "UVWYQXZ" {
			t.Errorf("got bag %q want %q", g.Bag, "UVWYQXZ")
		}
		if g.Turn != 1 {
			t.Errorf("got turn %d want 1", g.Turn)
		}
	})

	t.Run("not enough tiles in the bag", func(t *testing.T) {
		g := newTestGame("QXZAEIO", "RS")
		err := g.Swap(0, "QXZ", noShuffle)

		if !errors.Is(err, ErrBagTooSmall) {
			t.Errorf("got %v want %v", err, ErrBagTooSmall)
		}
	})

	t.Run("letters must be on the rack", func(t *testing.T) {
		g := newTestGame("QXZAEIO", "RSTUVWY")
		err := g.Swap(0, "QQ", noShuffle)

		if !errors.Is(err, ErrNotInRack) {
			t.Errorf("got %v want %v", err, ErrNotInRack)
		}
	})
}

func TestEnd(t *testing.T) {
	g := newTestGame("CATXXXX", "")
	g.Seats[0].Score = 42

	if err := g.End(0); err != nil {
		t.Fatal(err)
	}
	if !g.Over || g.Seats[0].Score != 42 {
		t.Errorf("got over=%v score=%d, want true and 42", g.Over, g.Seats[0].Score)
	}
	if err := g.End(0); !errors.Is(err, ErrGameOver) {
		t.Errorf("ending twice: got %v want %v", err, ErrGameOver)
	}
}

func TestWords(t *testing.T) {
	dict := Words()

	for _, w := range []string{"quiz", "CAT", "toaster", "ox"} {
		if !dict.Has(w) {
			t.Errorf("%q should be a word", w)
		}
	}
	for _, w := range []string{"xqz", "catt", ""} {
		if dict.Has(w) {
			t.Errorf("%q shouldn't be a word", w)
		}
	}
}

func newVersusGame(rack0, rack1, bag string) *Game {
	g := newTestGame(rack0, bag)
	g.Seats = append(g.Seats, Seat{Rack: rack1})
	return g
}

func TestVersus(t *testing.T) {
	dict := NewWordSet("cat", "at", "to", "cats")

	assertError := func(t testing.TB, got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Errorf("got error %v want %v", got, want)
		}
	}

	t.Run("deals two racks and seat 0 starts", func(t *testing.T) {
		g := New(2, noShuffle)

		if len(g.Seats) != 2 || len(g.Seats[0].Rack) != RackSize || len(g.Seats[1].Rack) != RackSize {
			t.Fatalf("got seats %+v", g.Seats)
		}
		if got := len(g.Bag) + 2*RackSize; got != 80 {
			t.Errorf("got %d tiles in total, want 80", got)
		}
		if g.ToMove != 0 {
			t.Errorf("got seat %d to move, want 0", g.ToMove)
		}
	})

	t.Run("players take turns", func(t *testing.T) {
		g := newVersusGame("CATXXXX", "TOXXXXX", "EEEEEEEE")

		_, err := g.Play(1, across(Centre, 3, "TO"), dict)
		assertError(t, err, ErrNotYourTurn)

		if _, err := g.Play(0, across(Centre, 3, "CAT"), dict); err != nil {
			t.Fatal(err)
		}
		if g.ToMove != 1 {
			t.Errorf("got seat %d to move, want 1", g.ToMove)
		}

		_, err = g.Play(0, across(Centre+1, 4, "AT"), dict)
		assertError(t, err, ErrNotYourTurn)

		move, err := g.Play(1, across(Centre+1, 4, "TO"), dict)
		if err != nil {
			t.Fatal(err)
		}
		if g.Seats[1].Score != move.Points || g.Seats[0].Score != 8 {
			t.Errorf("got scores %d and %d", g.Seats[0].Score, g.Seats[1].Score)
		}
		if g.ToMove != 0 {
			t.Errorf("got seat %d to move, want 0", g.ToMove)
		}
	})

	t.Run("going out takes the other rack's points", func(t *testing.T) {

		g := newVersusGame("CAT", "QX", "")
		g.Play(0, across(Centre, 3, "CAT"), dict)

		if !g.Over {
			t.Fatal("game should be over")
		}
		if g.Seats[0].Score != 8+21 || g.Seats[1].Score != -21 {
			t.Errorf("got scores %d and %d, want 29 and -21", g.Seats[0].Score, g.Seats[1].Score)
		}
		if g.Winner() != 0 {
			t.Errorf("got winner %d want 0", g.Winner())
		}
	})

	t.Run("three scoreless rounds end the game", func(t *testing.T) {
		g := newVersusGame("QXXXXXX", "AXXXXXX", "")
		g.Seats[0].Score, g.Seats[1].Score = 10, 10

		for i := 0; i < ScorelessRounds*2; i++ {
			if err := g.Pass(g.ToMove); err != nil {
				t.Fatal(err)
			}
		}

		if !g.Over {
			t.Fatal("game should be over")
		}

		if g.Seats[0].Score != 10-66 || g.Seats[1].Score != 10-55 {
			t.Errorf("got scores %d and %d", g.Seats[0].Score, g.Seats[1].Score)
		}
		if g.Winner() != 1 {
			t.Errorf("got winner %d want 1", g.Winner())
		}
	})

	t.Run("resigning loses, even when ahead", func(t *testing.T) {
		g := newVersusGame("CATXXXX", "TOXXXXX", "")
		g.Seats[0].Score = 100

		if err := g.Resign(0); err != nil {
			t.Fatal(err)
		}
		if g.Winner() != 1 {
			t.Errorf("got winner %d want 1", g.Winner())
		}
		assertError(t, g.Resign(1), ErrGameOver)
	})

	t.Run("a draw has no winner", func(t *testing.T) {
		g := newVersusGame("", "", "")
		g.Over = true

		if g.Winner() != -1 {
			t.Errorf("got winner %d want -1", g.Winner())
		}
	})

	t.Run("solo and versus only moves", func(t *testing.T) {
		assertError(t, newVersusGame("A", "B", "").End(0), ErrSoloOnly)
		assertError(t, newTestGame("A", "").Resign(0), ErrVersusOnly)
	})

	t.Run("a player only sees their own rack", func(t *testing.T) {
		g := newVersusGame("CATXXXX", "TOXXXXX", "")

		if v := g.View(1); v.Rack != "TOXXXXX" || len(v.Scores) != 2 {
			t.Errorf("seat 1 got %+v", v)
		}
		if v := g.View(5); v.Rack != "" {
			t.Errorf("a stranger saw rack %q", v.Rack)
		}
	})
}
