package game

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed words.txt
var wordList string

type Dictionary interface {
	Has(word string) bool
}

type WordSet map[string]struct{}

func NewWordSet(words ...string) WordSet {
	set := make(WordSet, len(words))
	for _, w := range words {
		set[strings.ToLower(w)] = struct{}{}
	}
	return set
}

func (w WordSet) Has(word string) bool {
	_, ok := w[strings.ToLower(word)]
	return ok
}

var (
	loadWords sync.Once
	words     WordSet
)

func Words() WordSet {
	loadWords.Do(func() {
		words = NewWordSet(strings.Fields(wordList)...)
	})
	return words
}
