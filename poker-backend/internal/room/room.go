package room

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"poker-backend/internal/game"
	"poker-backend/internal/model"
)

type Seat struct {
	Index  int    `json:"index"`
	UserID string `json:"userId,omitempty"`
	Name   string `json:"name,omitempty"`
	Stack  int64  `json:"stack"`
	BuyIn  int64  `json:"buyIn"`
}

type HandSummary struct {
	StartStack int64 `json:"startStack"`
	EndStack   int64 `json:"endStack"`
	Delta      int64 `json:"delta"`
}

type HandPlayerRecord struct {
	UserID     string            `json:"userId"`
	Name       string            `json:"name"`
	SeatIndex  int               `json:"seatIndex"`
	StartStack int64             `json:"startStack"`
	EndStack   int64             `json:"endStack"`
	Delta      int64             `json:"delta"`
	TotalBet   int64             `json:"totalBet"`
	Status     game.PlayerStatus `json:"status"`
	Cards      []game.Card       `json:"cards,omitempty"`
}

type HandRecord struct {
	Number     int                `json:"number"`
	StartedAt  time.Time          `json:"startedAt"`
	EndedAt    time.Time          `json:"endedAt"`
	DealerSeat int                `json:"dealerSeat"`
	SmallBlind int64              `json:"smallBlind"`
	BigBlind   int64              `json:"bigBlind"`
	Pot        int64              `json:"pot"`
	Community  []game.Card        `json:"community"`
	Winners    []int              `json:"winners"`
	Players    []HandPlayerRecord `json:"players"`
	Log        []game.LogEntry    `json:"log"`
}

type LedgerEntry struct {
	UserID        string `json:"userId"`
	Name          string `json:"name"`
	SeatIndex     int    `json:"seatIndex"`
	BuyIn         int64  `json:"buyIn"`
	CurrentStack  int64  `json:"currentStack"`
	Net           int64  `json:"net"`
	LastHandDelta int64  `json:"lastHandDelta"`
	HandsPlayed   int    `json:"handsPlayed"`
	HandsWon      int    `json:"handsWon"`
}

type Room struct {
	mu       sync.Mutex
	ID       string             `json:"id"`
	OwnerID  string             `json:"ownerId"`
	Settings model.RoomSettings `json:"settings"`
	Seats    map[int]*Seat      `json:"seats"`
	Game     *game.GameState    `json:"game,omitempty"`
	Engine   *game.Engine       `json:"-"`
	Paused   bool               `json:"paused"`
	Ending   bool               `json:"endingAfterHand"`

	nextHandNumber         int
	lastRecordedHandNumber int
	// Captured at the start of a hand (before posting blinds), used to calculate
	// the delta when the hand ends.
	handStartStacks map[int]int64
	// Computed when a hand ends, keyed by userId.
	lastHandSummaryByUser map[string]HandSummary
	handHistory           []HandRecord
}

func New(ownerID string, settings model.RoomSettings) *Room {
	if settings.MaxSeats == 0 {
		settings.MaxSeats = 9
	}
	if settings.SmallBlind == 0 {
		settings.SmallBlind = 1
	}
	if settings.BigBlind == 0 {
		settings.BigBlind = 2
	}
	if settings.MinBuyIn == 0 {
		settings.MinBuyIn = 200
	}
	if settings.MaxBuyIn == 0 {
		settings.MaxBuyIn = 2000
	}
	return &Room{ID: uuid.NewString(), OwnerID: ownerID, Settings: settings, Seats: map[int]*Seat{}, Engine: game.NewEngine(), nextHandNumber: 1, handHistory: []HandRecord{}}
}

func (r *Room) Sit(userID, name string, seat int, buyIn int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	if seat < 0 || seat >= r.Settings.MaxSeats {
		return errors.New("invalid seat")
	}
	if _, ok := r.Seats[seat]; ok {
		return errors.New("seat taken")
	}
	for _, existing := range r.Seats {
		if existing.UserID == userID {
			return errors.New("user already seated")
		}
		if strings.EqualFold(strings.TrimSpace(existing.Name), name) {
			return errors.New("name already taken")
		}
	}
	if buyIn < r.Settings.MinBuyIn || buyIn > r.Settings.MaxBuyIn {
		return errors.New("invalid buy-in")
	}
	r.Seats[seat] = &Seat{Index: seat, UserID: userID, Name: name, Stack: buyIn, BuyIn: buyIn}
	return nil
}

func (r *Room) Start(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.requireOwnerLocked(userID); err != nil {
		return err
	}
	if r.Game != nil && r.Game.Phase != game.PhaseFinished {
		return errors.New("game already started")
	}
	players := map[int]*game.PlayerState{}
	r.handStartStacks = map[int]int64{}
	for s, seat := range r.Seats {
		players[s] = &game.PlayerState{UserID: seat.UserID, Name: seat.Name, SeatIndex: s, Stack: seat.Stack, Status: game.StatusActive}
		r.handStartStacks[s] = seat.Stack
	}
	dealer, sb, bb, err := pickButton(players)
	if err != nil {
		return err
	}
	handNumber := r.nextHandNumber
	r.nextHandNumber++
	g, err := r.Engine.StartHand(players, dealer, sb, bb, r.Settings.SmallBlind, r.Settings.BigBlind, handNumber)
	if err != nil {
		r.nextHandNumber--
		return err
	}
	r.Game = g
	r.Paused = false
	r.Ending = false
	return nil
}

func (r *Room) Action(userID string, a game.Action) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Game == nil {
		return errors.New("game not started")
	}
	if r.Paused {
		return errors.New("game is paused")
	}
	if err := r.Engine.ApplyAction(r.Game, userID, a); err != nil {
		return err
	}
	r.syncStacksLocked()
	r.maybeAutoStartNextHandLocked()
	return nil
}

func (r *Room) SkipCurrentTurn() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Game == nil {
		return errors.New("game not started")
	}
	if r.Paused {
		return errors.New("game is paused")
	}
	if err := r.Engine.AutoActCurrent(r.Game); err != nil {
		return err
	}
	r.syncStacksLocked()
	r.maybeAutoStartNextHandLocked()
	return nil
}

func (r *Room) Pause(userID string, paused bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.requireOwnerLocked(userID); err != nil {
		return err
	}
	if r.Game == nil {
		return errors.New("game not started")
	}
	r.Paused = paused
	return nil
}

func (r *Room) End(userID string, requestedHandNumber int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.requireOwnerLocked(userID); err != nil {
		return err
	}
	if r.Game == nil {
		r.Paused = false
		r.Ending = false
		r.handStartStacks = nil
		return nil
	}
	if requestedHandNumber > 0 && r.Game.HandNumber > requestedHandNumber && r.lastRecordedHandNumber >= requestedHandNumber {
		if handHasPlayerAction(r.Game) {
			r.Ending = true
			return nil
		}
		r.Game = nil
		r.Paused = false
		r.Ending = false
		r.handStartStacks = nil
		return nil
	}
	if r.Game.Phase == game.PhaseFinished {
		r.recordFinishedHandLocked()
		r.Game = nil
		r.Paused = false
		r.Ending = false
		r.handStartStacks = nil
		return nil
	}
	r.Ending = true
	return nil
}

func handHasPlayerAction(g *game.GameState) bool {
	if g == nil {
		return false
	}
	for _, entry := range g.Log {
		switch entry.Type {
		case "fold", "check", "call", "bet", "raise", "all_in":
			return true
		}
	}
	return false
}

func (r *Room) syncStacksLocked() {
	for s, p := range r.Game.Players {
		if seat := r.Seats[s]; seat != nil {
			seat.Stack = p.Stack
		}
	}
}

func (r *Room) maybeAutoStartNextHandLocked() {
	// Auto-start next hand when the current hand is finished and there are
	// more than 2 players with stack > 0 (as requested).
	if r.Game == nil || r.Game.Phase != game.PhaseFinished {
		return
	}

	r.recordFinishedHandLocked()
	if r.Ending {
		r.Game = nil
		r.Paused = false
		r.Ending = false
		r.handStartStacks = nil
		return
	}
	if r.Paused {
		return
	}

	players := map[int]*game.PlayerState{}
	for s, seat := range r.Seats {
		if seat != nil && seat.Stack > 0 {
			players[s] = &game.PlayerState{
				UserID:    seat.UserID,
				Name:      seat.Name,
				SeatIndex: s,
				Stack:     seat.Stack,
				Status:    game.StatusActive,
			}
		}
	}
	if len(players) < 2 {
		return
	}

	// Capture start-of-hand stacks for the next hand (before posting blinds).
	r.handStartStacks = map[int]int64{}
	for seatIndex, p := range players {
		if p != nil {
			r.handStartStacks[seatIndex] = p.Stack
		}
	}

	dealer, sb, bb, err := pickButton(players)
	if err != nil {
		return
	}
	handNumber := r.nextHandNumber
	r.nextHandNumber++
	g, err := r.Engine.StartHand(players, dealer, sb, bb, r.Settings.SmallBlind, r.Settings.BigBlind, handNumber)
	if err != nil {
		r.nextHandNumber--
		return
	}
	r.Game = g
}

func (r *Room) recordFinishedHandLocked() {
	if r.Game == nil || r.Game.Phase != game.PhaseFinished || r.Game.HandNumber == 0 {
		return
	}
	if r.lastRecordedHandNumber == r.Game.HandNumber {
		return
	}

	summaryByUser := map[string]HandSummary{}
	players := make([]HandPlayerRecord, 0, len(r.Game.Players))
	for _, seatIndex := range sortedPlayerSeats(r.Game.Players) {
		p := r.Game.Players[seatIndex]
		if p == nil {
			continue
		}
		start := p.Stack
		if r.handStartStacks != nil {
			if v, ok := r.handStartStacks[seatIndex]; ok {
				start = v
			}
		}
		end := p.Stack
		delta := end - start
		players = append(players, HandPlayerRecord{
			UserID:     p.UserID,
			Name:       p.Name,
			SeatIndex:  p.SeatIndex,
			StartStack: start,
			EndStack:   end,
			Delta:      delta,
			TotalBet:   p.TotalBet,
			Status:     p.Status,
			Cards:      copyCards(p.Cards),
		})
		summaryByUser[p.UserID] = HandSummary{
			StartStack: start,
			EndStack:   end,
			Delta:      delta,
		}
	}

	r.handHistory = append(r.handHistory, HandRecord{
		Number:     r.Game.HandNumber,
		StartedAt:  r.Game.StartedAt,
		EndedAt:    time.Now(),
		DealerSeat: r.Game.DealerSeat,
		SmallBlind: r.Game.SmallBlind,
		BigBlind:   r.Game.BigBlind,
		Pot:        totalCommitted(r.Game),
		Community:  copyCards(r.Game.Community),
		Winners:    copyInts(r.Game.Winners),
		Players:    players,
		Log:        copyLog(r.Game.Log),
	})
	r.lastHandSummaryByUser = summaryByUser
	r.lastRecordedHandNumber = r.Game.HandNumber
}

func (r *Room) Snapshot(forUser string) map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	resp := map[string]any{"id": r.ID, "ownerId": r.OwnerID, "settings": r.Settings, "seats": r.Seats, "paused": r.Paused, "endingAfterHand": r.Ending, "handHistory": r.handHistory, "ledger": r.ledgerLocked()}
	if r.Game != nil {
		players := map[int]game.PublicPlayerState{}
		for s, p := range r.Game.Players {
			pub := game.PublicPlayerState{UserID: p.UserID, Name: p.Name, SeatIndex: p.SeatIndex, Stack: p.Stack, Bet: p.Bet, TotalBet: p.TotalBet, Status: p.Status, Acted: p.Acted}
			if p.UserID == forUser || r.Game.Phase == game.PhaseFinished {
				pub.Cards = p.Cards
			}
			players[s] = pub
		}
		resp["game"] = map[string]any{"handNumber": r.Game.HandNumber, "startedAt": r.Game.StartedAt, "phase": r.Game.Phase, "dealerSeat": r.Game.DealerSeat, "smallBlind": r.Game.SmallBlind, "bigBlind": r.Game.BigBlind, "minRaise": r.Game.MinRaise, "currentTurn": r.Game.CurrentTurn, "currentBet": r.Game.CurrentBet, "pot": r.Game.Pot, "community": r.Game.Community, "players": players, "winners": r.Game.Winners, "log": r.Game.Log}
	}

	if r.lastHandSummaryByUser != nil {
		if s, ok := r.lastHandSummaryByUser[forUser]; ok {
			resp["lastHandSummary"] = s
		}
	}
	return resp
}

func (r *Room) requireOwnerLocked(userID string) error {
	if userID != r.OwnerID {
		return errors.New("only room owner can do this")
	}
	return nil
}

func (r *Room) ledgerLocked() []LedgerEntry {
	byUser := map[string]*LedgerEntry{}
	for _, seatIndex := range sortedSeatIndexes(r.Seats) {
		seat := r.Seats[seatIndex]
		if seat == nil || seat.UserID == "" {
			continue
		}
		byUser[seat.UserID] = &LedgerEntry{
			UserID:       seat.UserID,
			Name:         seat.Name,
			SeatIndex:    seat.Index,
			BuyIn:        seat.BuyIn,
			CurrentStack: seat.Stack,
		}
	}

	for _, hand := range r.handHistory {
		for _, player := range hand.Players {
			row := byUser[player.UserID]
			if row == nil {
				row = &LedgerEntry{
					UserID:       player.UserID,
					Name:         player.Name,
					SeatIndex:    player.SeatIndex,
					CurrentStack: player.EndStack,
				}
				byUser[player.UserID] = row
			}
			row.HandsPlayed++
			row.Net += player.Delta
			row.LastHandDelta = player.Delta
			if containsInt(hand.Winners, player.SeatIndex) {
				row.HandsWon++
			}
		}
	}

	if r.Game != nil {
		for _, p := range r.Game.Players {
			if p == nil {
				continue
			}
			if row := byUser[p.UserID]; row != nil {
				row.CurrentStack = p.Stack
			}
		}
	}

	rows := make([]LedgerEntry, 0, len(byUser))
	for _, row := range byUser {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SeatIndex == rows[j].SeatIndex {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].SeatIndex < rows[j].SeatIndex
	})
	return rows
}

func sortedSeatIndexes(seats map[int]*Seat) []int {
	indexes := make([]int, 0, len(seats))
	for seatIndex := range seats {
		indexes = append(indexes, seatIndex)
	}
	sort.Ints(indexes)
	return indexes
}

func sortedPlayerSeats(players map[int]*game.PlayerState) []int {
	indexes := make([]int, 0, len(players))
	for seatIndex := range players {
		indexes = append(indexes, seatIndex)
	}
	sort.Ints(indexes)
	return indexes
}

func totalCommitted(g *game.GameState) int64 {
	var total int64
	for _, p := range g.Players {
		if p != nil {
			total += p.TotalBet
		}
	}
	return total
}

func copyCards(cards []game.Card) []game.Card {
	if len(cards) == 0 {
		return []game.Card{}
	}
	out := make([]game.Card, len(cards))
	copy(out, cards)
	return out
}

func copyInts(values []int) []int {
	if len(values) == 0 {
		return []int{}
	}
	out := make([]int, len(values))
	copy(out, values)
	return out
}

func copyLog(entries []game.LogEntry) []game.LogEntry {
	if len(entries) == 0 {
		return []game.LogEntry{}
	}
	out := make([]game.LogEntry, len(entries))
	copy(out, entries)
	return out
}

func containsInt(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func pickButton(players map[int]*game.PlayerState) (dealer, sb, bb int, err error) {
	seats := game.ActiveSeatIndexes(players)
	if len(seats) < 2 {
		return 0, 0, 0, errors.New("need at least two players")
	}
	if len(seats) == 2 {
		return seats[0], seats[0], seats[1], nil
	}
	// 3+ players: dealer is seats[0], then SB/BB go clockwise (simplified MVP rule).
	return seats[0], seats[1], seats[2], nil
}
