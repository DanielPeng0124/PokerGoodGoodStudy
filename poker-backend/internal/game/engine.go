package game

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) StartHand(players map[int]*PlayerState, dealer, sbSeat, bbSeat int, sb, bb int64, handNumber int) (*GameState, error) {
	active := ActiveSeatIndexes(players)
	if len(active) < 2 {
		return nil, errors.New("need at least two players")
	}
	deck := NewDeck()
	Shuffle(deck)
	g := &GameState{
		HandNumber: handNumber,
		StartedAt:  time.Now(),
		Phase:      PhasePreflop,
		DealerSeat: dealer,
		SmallBlind: sb,
		BigBlind:   bb,
		CurrentBet: bb,
		MinRaise:   bb,
		Players:    players,
		Deck:       deck,
		Community:  []Card{},
		Log:        []LogEntry{},
	}
	for _, p := range players {
		if p != nil && p.Stack > 0 {
			p.Status = StatusActive
			p.Bet = 0
			p.TotalBet = 0
			p.Cards = nil
			p.Acted = false
		}
	}
	addLog(g, LogEntry{Type: "hand_started", Message: fmt.Sprintf("Hand #%d started", handNumber)})
	if paid := postBlind(g, sbSeat, sb); paid > 0 {
		addSeatLog(g, "small_blind", sbSeat, paid, fmt.Sprintf("%s posts small blind %d", displayName(g, sbSeat), paid))
	}
	if paid := postBlind(g, bbSeat, bb); paid > 0 {
		addSeatLog(g, "big_blind", bbSeat, paid, fmt.Sprintf("%s posts big blind %d", displayName(g, bbSeat), paid))
	}
	for i := 0; i < 2; i++ {
		for _, seat := range ActiveSeatIndexes(players) {
			drawTo(g, seat)
		}
	}
	addLog(g, LogEntry{Type: "deal", Message: fmt.Sprintf("Cards dealt to %d players", len(active))})
	g.CurrentTurn = firstPreflopActor(g, bbSeat)
	if !needsAction(g, g.CurrentTurn) {
		g.CurrentTurn = nextToAct(g, bbSeat)
	}
	return g, nil
}

func (e *Engine) ApplyAction(g *GameState, userID string, a Action) error {
	if g == nil || g.Phase == PhaseFinished {
		return errors.New("hand is not active")
	}
	sanitizeTurn(g)
	p := g.Players[g.CurrentTurn]
	if p == nil || p.UserID != userID {
		return errors.New("not your turn")
	}
	return e.applyCurrentPlayerAction(g, p, a)
}

// AutoActCurrent is intentionally useful for an MVP/local game: it unblocks a disconnected
// or forgotten seat by checking when possible, otherwise folding. Remove or protect it before production.
func (e *Engine) AutoActCurrent(g *GameState) error {
	if g == nil || g.Phase == PhaseFinished {
		return errors.New("hand is not active")
	}
	sanitizeTurn(g)
	p := g.Players[g.CurrentTurn]
	if p == nil || p.Status != StatusActive {
		return errors.New("no active player to auto-act")
	}
	if g.CurrentBet-p.Bet <= 0 {
		return e.applyCurrentPlayerAction(g, p, Action{Type: ActionCheck})
	}
	return e.applyCurrentPlayerAction(g, p, Action{Type: ActionFold})
}

func (e *Engine) applyCurrentPlayerAction(g *GameState, p *PlayerState, a Action) error {
	if p.Status != StatusActive {
		return errors.New("player not active")
	}
	toCall := g.CurrentBet - p.Bet
	seat := p.SeatIndex
	switch a.Type {
	case ActionFold:
		p.Status = StatusFolded
		p.Acted = true
		addSeatLog(g, "fold", seat, 0, fmt.Sprintf("%s folds", p.Name))
	case ActionCheck:
		if toCall != 0 {
			return errors.New("cannot check")
		}
		p.Acted = true
		addSeatLog(g, "check", seat, 0, fmt.Sprintf("%s checks", p.Name))
	case ActionCall:
		paid := pay(g, p, min(toCall, p.Stack))
		p.Acted = true
		if paid < toCall {
			addSeatLog(g, "call", seat, paid, fmt.Sprintf("%s calls all-in for %d", p.Name, paid))
		} else {
			addSeatLog(g, "call", seat, paid, fmt.Sprintf("%s calls %d", p.Name, paid))
		}
	case ActionBet:
		if g.CurrentBet != 0 {
			return errors.New("use raise")
		}
		if a.Amount < g.BigBlind {
			return errors.New("bet too small")
		}
		paid := pay(g, p, a.Amount)
		g.CurrentBet = p.Bet
		g.MinRaise = a.Amount
		markOthersUnacted(g, p.SeatIndex)
		p.Acted = true
		addSeatLog(g, "bet", seat, paid, fmt.Sprintf("%s bets %d", p.Name, paid))
	case ActionRaise:
		newBet := a.Amount
		if newBet <= g.CurrentBet {
			return errors.New("raise must be greater than current bet")
		}
		if newBet < g.CurrentBet+g.MinRaise && newBet-p.Bet < p.Stack {
			return errors.New("raise too small")
		}
		pay(g, p, newBet-p.Bet)
		if p.Bet > g.CurrentBet {
			diff := p.Bet - g.CurrentBet
			if diff >= g.MinRaise {
				markOthersUnacted(g, p.SeatIndex)
				g.MinRaise = diff
			}
			g.CurrentBet = p.Bet
		}
		p.Acted = true
		addSeatLog(g, "raise", seat, p.Bet, fmt.Sprintf("%s raises to %d", p.Name, p.Bet))
	case ActionAllIn:
		oldCurrentBet := g.CurrentBet
		paid := pay(g, p, p.Stack)
		p.Acted = true
		if p.Bet > oldCurrentBet {
			diff := p.Bet - oldCurrentBet
			if diff >= g.MinRaise {
				markOthersUnacted(g, p.SeatIndex)
				g.MinRaise = diff
			}
			g.CurrentBet = p.Bet
		}
		addSeatLog(g, "all_in", seat, paid, fmt.Sprintf("%s goes all-in for %d", p.Name, paid))
	default:
		return errors.New("unknown action")
	}
	return e.advance(g)
}

func (e *Engine) advance(g *GameState) error {
	if remaining(g) == 1 {
		g.Phase = PhaseFinished
		awardSingle(g)
		return nil
	}
	if onlyAllInOrFolded(g) || bettingRoundComplete(g) {
		return nextPhase(g)
	}
	g.CurrentTurn = nextToAct(g, g.CurrentTurn)
	return nil
}

func nextPhase(g *GameState) error {
	for _, p := range g.Players {
		if p != nil {
			p.Bet = 0
			p.Acted = false
		}
	}
	g.CurrentBet = 0
	g.MinRaise = g.BigBlind
	switch g.Phase {
	case PhasePreflop:
		burn(g)
		start := len(g.Community)
		drawCommunity(g, 3)
		g.Phase = PhaseFlop
		addLog(g, LogEntry{Type: "flop", Cards: copyCards(g.Community[start:]), Message: "Flop: " + cardsText(g.Community[start:])})
	case PhaseFlop:
		burn(g)
		start := len(g.Community)
		drawCommunity(g, 1)
		g.Phase = PhaseTurn
		addLog(g, LogEntry{Type: "turn", Cards: copyCards(g.Community[start:]), Message: "Turn: " + cardsText(g.Community[start:])})
	case PhaseTurn:
		burn(g)
		start := len(g.Community)
		drawCommunity(g, 1)
		g.Phase = PhaseRiver
		addLog(g, LogEntry{Type: "river", Cards: copyCards(g.Community[start:]), Message: "River: " + cardsText(g.Community[start:])})
	case PhaseRiver:
		g.Phase = PhaseShowdown
		addLog(g, LogEntry{Type: "showdown", Message: "Showdown"})
		showdown(g)
		g.Phase = PhaseFinished
		return nil
	}
	if onlyAllInOrFolded(g) {
		return nextPhase(g)
	}
	g.CurrentTurn = nextToAct(g, g.DealerSeat)
	return nil
}

func showdown(g *GameState) {
	best := int64(-1)
	winners := []int{}
	for seat, p := range g.Players {
		if p.Status == StatusActive || p.Status == StatusAllIn {
			score := EvaluateBest(append(append([]Card{}, p.Cards...), g.Community...))
			if score > best {
				best = score
				winners = []int{seat}
			} else if score == best {
				winners = append(winners, seat)
			}
		}
	}
	if len(winners) == 0 {
		return
	}
	share := g.Pot / int64(len(winners))
	for _, s := range winners {
		g.Players[s].Stack += share
		addSeatLog(g, "win", s, share, fmt.Sprintf("%s wins %d", displayName(g, s), share))
	}
	g.Winners = winners
	g.Pot = 0
}

func awardSingle(g *GameState) {
	for _, p := range g.Players {
		if p.Status == StatusActive || p.Status == StatusAllIn {
			amount := g.Pot
			p.Stack += amount
			g.Winners = []int{p.SeatIndex}
			addSeatLog(g, "win", p.SeatIndex, amount, fmt.Sprintf("%s wins %d", p.Name, amount))
			g.Pot = 0
			return
		}
	}
}

func postBlind(g *GameState, seat int, amt int64) int64 {
	if p := g.Players[seat]; p != nil {
		return pay(g, p, min(amt, p.Stack))
	}
	return 0
}

func pay(g *GameState, p *PlayerState, amt int64) int64 {
	if amt <= 0 || p.Stack <= 0 {
		return 0
	}
	if amt >= p.Stack {
		amt = p.Stack
		p.Status = StatusAllIn
	}
	p.Stack -= amt
	p.Bet += amt
	p.TotalBet += amt
	g.Pot += amt
	return amt
}

func drawTo(g *GameState, seat int) {
	if g.Players[seat] == nil || len(g.Deck) == 0 {
		return
	}
	g.Players[seat].Cards = append(g.Players[seat].Cards, g.Deck[0])
	g.Deck = g.Deck[1:]
}

func burn(g *GameState) {
	if len(g.Deck) > 0 {
		g.Deck = g.Deck[1:]
	}
}

func drawCommunity(g *GameState, n int) {
	for i := 0; i < n && len(g.Deck) > 0; i++ {
		g.Community = append(g.Community, g.Deck[0])
		g.Deck = g.Deck[1:]
	}
}

func ActiveSeatIndexes(players map[int]*PlayerState) []int {
	seats := []int{}
	for s, p := range players {
		if p != nil && p.Stack > 0 {
			seats = append(seats, s)
		}
	}
	sort.Ints(seats)
	return seats
}

func firstPreflopActor(g *GameState, bbSeat int) int {
	seats := ActiveSeatIndexes(g.Players)
	if len(seats) == 2 {
		return nextToAct(g, bbSeat)
	}
	return nextToAct(g, bbSeat)
}

func nextToAct(g *GameState, from int) int {
	seats := ActiveSeatIndexes(g.Players)
	if len(seats) == 0 {
		return g.CurrentTurn
	}
	for offset := 1; offset <= len(seats); offset++ {
		idx := indexOfOrInsertion(seats, from)
		s := seats[(idx+offset)%len(seats)]
		if needsAction(g, s) {
			return s
		}
	}
	for _, s := range seats {
		if p := g.Players[s]; p != nil && p.Status == StatusActive {
			return s
		}
	}
	return seats[0]
}

func indexOfOrInsertion(seats []int, from int) int {
	for i, s := range seats {
		if s == from {
			return i
		}
		if s > from {
			return i - 1
		}
	}
	return len(seats) - 1
}

func needsAction(g *GameState, seat int) bool {
	p := g.Players[seat]
	return p != nil && p.Status == StatusActive && (!p.Acted || p.Bet < g.CurrentBet)
}

func sanitizeTurn(g *GameState) {
	if g == nil || g.Phase == PhaseFinished {
		return
	}
	if !needsAction(g, g.CurrentTurn) {
		g.CurrentTurn = nextToAct(g, g.CurrentTurn)
	}
}

func markOthersUnacted(g *GameState, except int) {
	for s, p := range g.Players {
		if s != except && p.Status == StatusActive {
			p.Acted = false
		}
	}
}

func bettingRoundComplete(g *GameState) bool {
	for _, p := range g.Players {
		if p != nil && p.Status == StatusActive && (!p.Acted || p.Bet < g.CurrentBet) {
			return false
		}
	}
	return true
}

func remaining(g *GameState) int {
	n := 0
	for _, p := range g.Players {
		if p.Status == StatusActive || p.Status == StatusAllIn {
			n++
		}
	}
	return n
}

func onlyAllInOrFolded(g *GameState) bool {
	active := 0
	for _, p := range g.Players {
		if p.Status == StatusActive {
			active++
		}
	}
	return active == 0
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func addSeatLog(g *GameState, typ string, seat int, amount int64, message string) {
	addLog(g, LogEntry{
		Type:      typ,
		SeatIndex: seatPtr(seat),
		Name:      displayName(g, seat),
		Amount:    amount,
		Message:   message,
	})
}

func addLog(g *GameState, entry LogEntry) {
	if g == nil {
		return
	}
	entry.Seq = len(g.Log) + 1
	if entry.Phase == "" {
		entry.Phase = g.Phase
	}
	entry.Pot = g.Pot
	g.Log = append(g.Log, entry)
}

func displayName(g *GameState, seat int) string {
	if p := g.Players[seat]; p != nil && p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("Seat %d", seat+1)
}

func seatPtr(seat int) *int {
	return &seat
}

func copyCards(cards []Card) []Card {
	if len(cards) == 0 {
		return nil
	}
	out := make([]Card, len(cards))
	copy(out, cards)
	return out
}

func cardsText(cards []Card) string {
	labels := make([]string, 0, len(cards))
	for _, card := range cards {
		labels = append(labels, card.String())
	}
	return strings.Join(labels, " ")
}
