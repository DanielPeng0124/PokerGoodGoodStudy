package game

import (
	"crypto/rand"
	"encoding/binary"
	mrand "math/rand"
)

type Suit string
const (
	Spades Suit = "s"
	Hearts Suit = "h"
	Diamonds Suit = "d"
	Clubs Suit = "c"
)

type Card struct {
	Rank int  `json:"rank"` // 2..14
	Suit Suit `json:"suit"`
}

func (c Card) String() string {
	r := map[int]string{11:"J",12:"Q",13:"K",14:"A"}
	rs, ok := r[c.Rank]
	if !ok { rs = string(rune('0' + c.Rank)) }
	if c.Rank == 10 { rs = "T" }
	return rs + string(c.Suit)
}

func NewDeck() []Card {
	suits := []Suit{Spades, Hearts, Diamonds, Clubs}
	deck := make([]Card, 0, 52)
	for _, s := range suits {
		for r := 2; r <= 14; r++ {
			deck = append(deck, Card{Rank: r, Suit: s})
		}
	}
	return deck
}

func Shuffle(deck []Card) {
	var b [8]byte
	_, _ = rand.Read(b[:])
	seed := int64(binary.LittleEndian.Uint64(b[:]))
	r := mrand.New(mrand.NewSource(seed))
	r.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
}
