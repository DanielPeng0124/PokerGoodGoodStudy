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
	Away   bool   `json:"away,omitempty"`
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
	Seated        bool   `json:"seated"`
}

type TurnTimer struct {
	SeatIndex        int       `json:"seatIndex"`
	UserID           string    `json:"userId"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RemainingSeconds int64     `json:"remainingSeconds"`
	ExtensionsUsed   int       `json:"extensionsUsed"`
	ExtensionMax     int       `json:"extensionMax"`
	DurationSeconds  int64     `json:"durationSeconds"`
}

const (
	turnDurationSeconds = int64(10)
	maxTurnExtensions   = 3
)

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
	lastDealerSeat         int // -1 means no previous hand
	// Captured at the start of a hand (before posting blinds), used to calculate
	// the delta when the hand ends.
	handStartStacks map[int]int64

	// Computed when a hand ends, keyed by userId.
	lastHandSummaryByUser map[string]HandSummary
	handHistory           []HandRecord
	totalBuyInsByUser     map[string]int64
	offTableStacksByUser  map[string]int64
	lastStackByUser       map[string]int64
	lastNameByUser        map[string]string
	lastSeatByUser        map[string]int
	turnDeadline          time.Time
	turnExtensionUses     map[string]int
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
	return &Room{
		ID:                   uuid.NewString(),
		OwnerID:              ownerID,
		Settings:             settings,
		Seats:                map[int]*Seat{},
		Engine:               game.NewEngine(),
		nextHandNumber:       1,
		lastDealerSeat:       -1,
		handHistory:          []HandRecord{},
		totalBuyInsByUser:    map[string]int64{},
		offTableStacksByUser: map[string]int64{},
		lastStackByUser:      map[string]int64{},
		lastNameByUser:       map[string]string{},
		lastSeatByUser:       map[string]int{},
		turnExtensionUses:    map[string]int{},
	}
}

func (r *Room) Sit(userID, name string, seat int, buyIn int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupFinishedHandLocked()
	r.removeInactiveBustedSeatsLocked()
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
	r.noteBuyInLocked(userID, name, seat, buyIn)
	return nil
}

func (r *Room) SetAway(userID string, away bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	seatIndex, seat := r.findSeatByUserLocked(userID)
	if seat == nil {
		return errors.New("user is not seated")
	}
	if !away {
		if seat.Stack <= 0 {
			return errors.New("no chips; quit seat and sit again")
		}
		seat.Away = false
		r.notePlayerLocked(userID, seat.Name, seatIndex, seat.Stack)
		return nil
	}
	if !r.seatInActiveHandLocked(seatIndex) {
		r.releaseSeatLocked(seatIndex, seat)
		return nil
	}
	if seat.Stack <= 0 {
		return errors.New("no chips; quit seat and sit again")
	}
	seat.Away = true
	r.notePlayerLocked(userID, seat.Name, seatIndex, seat.Stack)
	return nil
}

func (r *Room) LeaveSeat(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	seatIndex, seat := r.findSeatByUserLocked(userID)
	if seat == nil {
		return errors.New("user is not seated")
	}
	if r.seatInActiveHandLocked(seatIndex) {
		return errors.New("cannot quit seat during an active hand")
	}
	r.releaseSeatLocked(seatIndex, seat)
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
		if seat == nil || seat.Away || seat.Stack <= 0 {
			continue
		}
		players[s] = &game.PlayerState{UserID: seat.UserID, Name: seat.Name, SeatIndex: s, Stack: seat.Stack, Status: game.StatusActive}
		r.handStartStacks[s] = seat.Stack
	}
	dealer, sb, bb, err := pickButton(players, r.lastDealerSeat)
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
	r.lastDealerSeat = dealer
	r.Game = g
	r.Paused = false
	r.Ending = false
	r.turnExtensionUses = map[string]int{}
	r.resetTurnTimerLocked()
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
	r.afterActionLocked()
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
	r.afterActionLocked()
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
	if paused {
		r.clearTurnTimerLocked()
	} else {
		r.resetTurnTimerLocked()
	}
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
		r.clearTurnTimerLocked()
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
		r.clearTurnTimerLocked()
		return nil
	}
	if r.Game.Phase == game.PhaseFinished {
		r.recordFinishedHandLocked()
		r.Game = nil
		r.Paused = false
		r.Ending = false
		r.handStartStacks = nil
		r.clearTurnTimerLocked()
		return nil
	}
	r.Ending = true
	return nil
}

func (r *Room) AddTurnTime(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Game == nil || r.Game.Phase == game.PhaseFinished {
		return errors.New("hand is not active")
	}
	if r.Paused {
		return errors.New("game is paused")
	}
	p := r.Game.Players[r.Game.CurrentTurn]
	if p == nil || p.UserID != userID {
		return errors.New("not your turn")
	}
	if r.turnExtensionUses == nil {
		r.turnExtensionUses = map[string]int{}
	}
	if r.turnExtensionUses[userID] >= maxTurnExtensions {
		return errors.New("no extra time left")
	}
	r.turnExtensionUses[userID]++
	now := time.Now()
	if r.turnDeadline.Before(now) {
		r.turnDeadline = now
	}
	r.turnDeadline = r.turnDeadline.Add(time.Duration(turnDurationSeconds) * time.Second)
	return nil
}

func (r *Room) AutoActExpired(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Game == nil || r.Game.Phase == game.PhaseFinished || r.Paused || r.turnDeadline.IsZero() || now.Before(r.turnDeadline) {
		return false
	}
	if err := r.Engine.AutoActCurrent(r.Game); err != nil {
		r.resetTurnTimerLocked()
		return false
	}
	r.afterActionLocked()
	return true
}

func (r *Room) afterActionLocked() {
	r.syncStacksLocked()
	r.maybeAutoStartNextHandLocked()
	r.resetTurnTimerLocked()
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
			r.notePlayerLocked(seat.UserID, seat.Name, s, r.offTableStackLocked(seat.UserID)+seat.Stack)
		} else if p != nil {
			r.notePlayerLocked(p.UserID, p.Name, s, r.totalStackForPlayerLocked(p, 0))
		}
	}
	if r.Game.Phase == game.PhaseFinished {
		r.recordFinishedHandLocked()
		r.removeBustedSeatsLocked()
		r.removeAwaySeatsLocked()
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
		r.clearTurnTimerLocked()
		return
	}
	if r.Paused {
		return
	}

	players := map[int]*game.PlayerState{}
	for s, seat := range r.Seats {
		if seat != nil && !seat.Away && seat.Stack > 0 {
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

	dealer, sb, bb, err := pickButton(players, r.lastDealerSeat)
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
	r.lastDealerSeat = dealer
	r.Game = g
	r.turnExtensionUses = map[string]int{}
	r.resetTurnTimerLocked()
}

func (r *Room) recordFinishedHandLocked() {
	if r.Game == nil || r.Game.Phase != game.PhaseFinished || r.Game.HandNumber == 0 {
		return
	}
	if r.lastRecordedHandNumber == r.Game.HandNumber {
		r.removeBustedSeatsLocked()
		r.removeAwaySeatsLocked()
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
		r.notePlayerLocked(p.UserID, p.Name, p.SeatIndex, r.totalStackForPlayerLocked(p, 0))
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
	r.removeBustedSeatsLocked()
	r.removeAwaySeatsLocked()
}

func (r *Room) Snapshot(forUser string) map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupFinishedHandLocked()
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
		gameMap := map[string]any{"handNumber": r.Game.HandNumber, "startedAt": r.Game.StartedAt, "phase": r.Game.Phase, "dealerSeat": r.Game.DealerSeat, "smallBlind": r.Game.SmallBlind, "bigBlind": r.Game.BigBlind, "minRaise": r.Game.MinRaise, "currentTurn": r.Game.CurrentTurn, "currentBet": r.Game.CurrentBet, "pot": r.Game.Pot, "community": r.Game.Community, "players": players, "winners": r.Game.Winners, "log": r.Game.Log}
		if !r.turnDeadline.IsZero() {
			gameMap["turnDeadline"] = r.turnDeadline
		}
		resp["game"] = gameMap
	}

	if timer := r.turnTimerLocked(time.Now()); timer != nil {
		resp["turnTimer"] = timer
	}

	if r.lastHandSummaryByUser != nil {
		if s, ok := r.lastHandSummaryByUser[forUser]; ok {
			resp["lastHandSummary"] = s
		}
	}
	return resp
}

func (r *Room) cleanupFinishedHandLocked() {
	if r.Game != nil && r.Game.Phase == game.PhaseFinished {
		r.recordFinishedHandLocked()
		r.removeBustedSeatsLocked()
		r.removeAwaySeatsLocked()
		r.clearTurnTimerLocked()
	}
}

func (r *Room) resetTurnTimerLocked() {
	if r.Game == nil || r.Game.Phase == game.PhaseFinished || r.Paused {
		r.clearTurnTimerLocked()
		return
	}
	if r.Game.Players[r.Game.CurrentTurn] == nil {
		r.clearTurnTimerLocked()
		return
	}
	r.turnDeadline = time.Now().Add(time.Duration(turnDurationSeconds) * time.Second)
}

func (r *Room) clearTurnTimerLocked() {
	r.turnDeadline = time.Time{}
}

func (r *Room) turnTimerLocked(now time.Time) *TurnTimer {
	if r.Game == nil || r.Game.Phase == game.PhaseFinished || r.Paused || r.turnDeadline.IsZero() {
		return nil
	}
	p := r.Game.Players[r.Game.CurrentTurn]
	if p == nil {
		return nil
	}
	remaining := int64(0)
	if now.Before(r.turnDeadline) {
		remaining = int64((r.turnDeadline.Sub(now) + time.Second - 1) / time.Second)
	}
	return &TurnTimer{
		SeatIndex:        p.SeatIndex,
		UserID:           p.UserID,
		ExpiresAt:        r.turnDeadline,
		RemainingSeconds: remaining,
		ExtensionsUsed:   r.turnExtensionUses[p.UserID],
		ExtensionMax:     maxTurnExtensions,
		DurationSeconds:  turnDurationSeconds,
	}
}

func (r *Room) requireOwnerLocked(userID string) error {
	if userID != r.OwnerID {
		return errors.New("only room owner can do this")
	}
	return nil
}

func (r *Room) findSeatByUserLocked(userID string) (int, *Seat) {
	for seatIndex, seat := range r.Seats {
		if seat != nil && seat.UserID == userID {
			return seatIndex, seat
		}
	}
	return 0, nil
}

func (r *Room) noteBuyInLocked(userID, name string, seatIndex int, buyIn int64) {
	if r.totalBuyInsByUser == nil {
		r.totalBuyInsByUser = map[string]int64{}
	}
	if r.offTableStacksByUser == nil {
		r.offTableStacksByUser = map[string]int64{}
	}
	r.totalBuyInsByUser[userID] += buyIn
	r.notePlayerLocked(userID, name, seatIndex, r.offTableStacksByUser[userID]+buyIn)
}

func (r *Room) offTableStackLocked(userID string) int64 {
	if r.offTableStacksByUser == nil {
		return 0
	}
	return r.offTableStacksByUser[userID]
}

func (r *Room) totalStackForPlayerLocked(p *game.PlayerState, committed int64) int64 {
	if p == nil {
		return 0
	}
	offTable := r.offTableStackLocked(p.UserID)
	if seat := r.Seats[p.SeatIndex]; seat != nil && seat.UserID == p.UserID {
		return offTable + p.Stack + committed
	}
	if offTable > 0 {
		return offTable
	}
	return p.Stack + committed
}

func (r *Room) notePlayerLocked(userID, name string, seatIndex int, stack int64) {
	if userID == "" {
		return
	}
	if r.lastStackByUser == nil {
		r.lastStackByUser = map[string]int64{}
	}
	if r.lastNameByUser == nil {
		r.lastNameByUser = map[string]string{}
	}
	if r.lastSeatByUser == nil {
		r.lastSeatByUser = map[string]int{}
	}
	r.lastStackByUser[userID] = stack
	if strings.TrimSpace(name) != "" {
		r.lastNameByUser[userID] = name
	}
	r.lastSeatByUser[userID] = seatIndex
}

func (r *Room) removeBustedSeatsLocked() {
	for seatIndex, seat := range r.Seats {
		if seat != nil && seat.Stack <= 0 {
			r.releaseSeatLocked(seatIndex, seat)
		}
	}
}

func (r *Room) removeAwaySeatsLocked() {
	for seatIndex, seat := range r.Seats {
		if seat != nil && seat.Away && !r.seatInActiveHandLocked(seatIndex) {
			r.releaseSeatLocked(seatIndex, seat)
		}
	}
}

func (r *Room) removeInactiveBustedSeatsLocked() {
	for seatIndex, seat := range r.Seats {
		if seat == nil || seat.Stack > 0 {
			continue
		}
		if r.seatInActiveHandLocked(seatIndex) {
			continue
		}
		r.releaseSeatLocked(seatIndex, seat)
	}
}

func (r *Room) seatInActiveHandLocked(seatIndex int) bool {
	if r.Game == nil || r.Game.Phase == game.PhaseFinished {
		return false
	}
	_, inHand := r.Game.Players[seatIndex]
	return inHand
}

func (r *Room) releaseSeatLocked(seatIndex int, seat *Seat) {
	if seat == nil {
		return
	}
	if r.offTableStacksByUser == nil {
		r.offTableStacksByUser = map[string]int64{}
	}
	if seat.Stack > 0 {
		r.offTableStacksByUser[seat.UserID] += seat.Stack
	}
	r.notePlayerLocked(seat.UserID, seat.Name, seatIndex, r.offTableStacksByUser[seat.UserID])
	delete(r.Seats, seatIndex)
}

func (r *Room) ledgerLocked() []LedgerEntry {
	byUser := map[string]*LedgerEntry{}
	for userID, buyIn := range r.totalBuyInsByUser {
		byUser[userID] = &LedgerEntry{
			UserID:       userID,
			Name:         r.lastNameByUser[userID],
			SeatIndex:    r.lastSeatByUser[userID],
			BuyIn:        buyIn,
			CurrentStack: r.lastStackByUser[userID],
		}
	}

	for _, hand := range r.handHistory {
		for _, player := range hand.Players {
			row := byUser[player.UserID]
			if row == nil {
				row = &LedgerEntry{
					UserID:    player.UserID,
					Name:      player.Name,
					SeatIndex: player.SeatIndex,
				}
				byUser[player.UserID] = row
			}
			if row.Name == "" {
				row.Name = player.Name
			}
			row.SeatIndex = player.SeatIndex
			row.HandsPlayed++
			row.LastHandDelta = player.Delta
			if containsInt(hand.Winners, player.SeatIndex) {
				row.HandsWon++
			}
		}
	}

	for _, seatIndex := range sortedSeatIndexes(r.Seats) {
		seat := r.Seats[seatIndex]
		if seat == nil || seat.UserID == "" {
			continue
		}
		row := byUser[seat.UserID]
		if row == nil {
			row = &LedgerEntry{UserID: seat.UserID}
			byUser[seat.UserID] = row
		}
		row.Name = seat.Name
		row.SeatIndex = seat.Index
		row.BuyIn = r.totalBuyInsByUser[seat.UserID]
		if row.BuyIn == 0 {
			row.BuyIn = seat.BuyIn
		}
		row.CurrentStack = r.offTableStackLocked(seat.UserID) + seat.Stack
		row.Seated = true
	}

	if r.Game != nil && r.Game.Phase != game.PhaseFinished {
		for _, p := range r.Game.Players {
			if p == nil {
				continue
			}
			if row := byUser[p.UserID]; row != nil {
				row.CurrentStack = r.totalStackForPlayerLocked(p, p.TotalBet)
			}
		}
	}

	rows := make([]LedgerEntry, 0, len(byUser))
	for _, row := range byUser {
		row.Net = row.CurrentStack - row.BuyIn
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

func pickButton(players map[int]*game.PlayerState, lastDealer int) (dealer, sb, bb int, err error) {
	seats := game.ActiveSeatIndexes(players)
	if len(seats) < 2 {
		return 0, 0, 0, errors.New("need at least two players")
	}

	// Find the next dealer seat clockwise after lastDealer.
	// lastDealer == -1 means first hand: start at seats[0].
	dealerIdx := 0
	if lastDealer >= 0 {
		for i, s := range seats {
			if s > lastDealer {
				dealerIdx = i
				break
			}
		}
		// If all active seats are <= lastDealer, wrap around to seats[0].
	}

	n := len(seats)
	dealer = seats[dealerIdx]
	if n == 2 {
		return dealer, dealer, seats[(dealerIdx+1)%n], nil
	}
	return dealer, seats[(dealerIdx+1)%n], seats[(dealerIdx+2)%n], nil
}
