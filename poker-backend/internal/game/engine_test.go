package game

import "testing"

func TestShowdownCapsShortStackWinAndAwardsSidePot(t *testing.T) {
	g := &GameState{
		Phase: PhaseShowdown,
		Pot:   450,
		Community: []Card{
			card(2, Clubs),
			card(7, Diamonds),
			card(9, Hearts),
			card(11, Spades),
			card(3, Clubs),
		},
		Players: map[int]*PlayerState{
			0: {UserID: "short", Name: "Short", SeatIndex: 0, Stack: 0, TotalBet: 50, Status: StatusAllIn, Cards: []Card{card(14, Spades), card(14, Hearts)}},
			1: {UserID: "middle", Name: "Middle", SeatIndex: 1, Stack: 0, TotalBet: 200, Status: StatusAllIn, Cards: []Card{card(13, Spades), card(12, Hearts)}},
			2: {UserID: "deep", Name: "Deep", SeatIndex: 2, Stack: 0, TotalBet: 200, Status: StatusAllIn, Cards: []Card{card(5, Spades), card(5, Hearts)}},
		},
	}

	showdown(g)

	if g.Players[0].Stack != 150 {
		t.Fatalf("short stack = %d, want main pot 150", g.Players[0].Stack)
	}
	if g.Players[1].Stack != 0 {
		t.Fatalf("middle stack = %d, want 0", g.Players[1].Stack)
	}
	if g.Players[2].Stack != 300 {
		t.Fatalf("deep stack = %d, want side pot 300", g.Players[2].Stack)
	}
	if g.Pot != 0 {
		t.Fatalf("pot = %d, want 0", g.Pot)
	}
	assertSeats(t, g.Winners, []int{0, 2})
}

func TestShowdownReturnsUnmatchedSidePot(t *testing.T) {
	g := &GameState{
		Phase: PhaseShowdown,
		Pot:   250,
		Community: []Card{
			card(2, Clubs),
			card(7, Diamonds),
			card(9, Hearts),
			card(11, Spades),
			card(3, Clubs),
		},
		Players: map[int]*PlayerState{
			0: {UserID: "short", Name: "Short", SeatIndex: 0, Stack: 0, TotalBet: 50, Status: StatusAllIn, Cards: []Card{card(14, Spades), card(14, Hearts)}},
			1: {UserID: "deep", Name: "Deep", SeatIndex: 1, Stack: 0, TotalBet: 200, Status: StatusActive, Cards: []Card{card(13, Spades), card(12, Hearts)}},
		},
	}

	showdown(g)

	if g.Players[0].Stack != 100 {
		t.Fatalf("short stack = %d, want capped main pot 100", g.Players[0].Stack)
	}
	if g.Players[1].Stack != 150 {
		t.Fatalf("deep stack = %d, want unmatched 150 returned", g.Players[1].Stack)
	}
	if g.Pot != 0 {
		t.Fatalf("pot = %d, want 0", g.Pot)
	}
	assertSeats(t, g.Winners, []int{0})
	if got := g.Log[len(g.Log)-1].Type; got != "return" {
		t.Fatalf("last log type = %q, want return", got)
	}
}

func TestHeadsUpAllInRunsOutAndCapsWinToOpponentStack(t *testing.T) {
	g := &GameState{
		Phase:       PhasePreflop,
		CurrentTurn: 0,
		CurrentBet:  1000,
		BigBlind:    10,
		MinRaise:    10,
		Pot:         1000,
		Community:   []Card{},
		Log:         []LogEntry{},
		Deck: []Card{
			card(2, Spades),
			card(2, Clubs), card(7, Diamonds), card(9, Hearts),
			card(3, Spades),
			card(11, Spades),
			card(5, Spades),
			card(3, Clubs),
		},
		Players: map[int]*PlayerState{
			0: {UserID: "short", Name: "Short", SeatIndex: 0, Stack: 200, Bet: 0, TotalBet: 0, Status: StatusActive, Cards: []Card{card(4, Spades), card(4, Hearts)}},
			1: {UserID: "deep", Name: "Deep", SeatIndex: 1, Stack: 500, Bet: 1000, TotalBet: 1000, Status: StatusActive, Acted: true, Cards: []Card{card(14, Spades), card(14, Hearts)}},
		},
	}

	if err := NewEngine().ApplyAction(g, "short", Action{Type: ActionCall}); err != nil {
		t.Fatalf("short call: %v", err)
	}

	if g.Phase != PhaseFinished {
		t.Fatalf("phase = %s, want finished", g.Phase)
	}
	if g.Players[0].Stack != 0 {
		t.Fatalf("short stack = %d, want 0", g.Players[0].Stack)
	}
	if g.Players[1].Stack != 1700 {
		t.Fatalf("deep stack = %d, want 1700; net win should be capped at 200", g.Players[1].Stack)
	}
	if g.Pot != 0 {
		t.Fatalf("pot = %d, want 0", g.Pot)
	}
	assertSeats(t, g.Winners, []int{1})
}

func TestAwardSingleReturnsUnmatchedChips(t *testing.T) {
	g := &GameState{
		Phase: PhasePreflop,
		Pot:   1200,
		Log:   []LogEntry{},
		Players: map[int]*PlayerState{
			0: {UserID: "short", Name: "Short", SeatIndex: 0, Stack: 0, TotalBet: 200, Status: StatusFolded},
			1: {UserID: "deep", Name: "Deep", SeatIndex: 1, Stack: 500, TotalBet: 1000, Status: StatusActive},
		},
	}

	awardSingle(g)

	if g.Players[1].Stack != 1700 {
		t.Fatalf("deep stack = %d, want 1700; net win should be capped at 200", g.Players[1].Stack)
	}
	if g.Pot != 0 {
		t.Fatalf("pot = %d, want 0", g.Pot)
	}
	assertSeats(t, g.Winners, []int{1})
	if got := g.Log[len(g.Log)-1].Type; got != "return" {
		t.Fatalf("last log type = %q, want return", got)
	}
}

func card(rank int, suit Suit) Card {
	return Card{Rank: rank, Suit: suit}
}

func assertSeats(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("seats = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("seats = %#v, want %#v", got, want)
		}
	}
}
