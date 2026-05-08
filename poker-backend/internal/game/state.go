package game

import "time"

type Phase string

const (
	PhaseWaiting  Phase = "waiting"
	PhasePreflop  Phase = "preflop"
	PhaseFlop     Phase = "flop"
	PhaseTurn     Phase = "turn"
	PhaseRiver    Phase = "river"
	PhaseShowdown Phase = "showdown"
	PhaseFinished Phase = "finished"
)

type PlayerStatus string

const (
	StatusEmpty      PlayerStatus = "empty"
	StatusActive     PlayerStatus = "active"
	StatusFolded     PlayerStatus = "folded"
	StatusAllIn      PlayerStatus = "all_in"
	StatusSittingOut PlayerStatus = "sitting_out"
)

type ActionType string

const (
	ActionFold  ActionType = "fold"
	ActionCheck ActionType = "check"
	ActionCall  ActionType = "call"
	ActionBet   ActionType = "bet"
	ActionRaise ActionType = "raise"
	ActionAllIn ActionType = "all_in"
)

type Action struct {
	Type   ActionType `json:"type"`
	Amount int64      `json:"amount,omitempty"`
}

type LogEntry struct {
	Seq       int    `json:"seq"`
	Phase     Phase  `json:"phase"`
	Type      string `json:"type"`
	SeatIndex *int   `json:"seatIndex,omitempty"`
	Name      string `json:"name,omitempty"`
	Amount    int64  `json:"amount,omitempty"`
	Pot       int64  `json:"pot"`
	Cards     []Card `json:"cards,omitempty"`
	Message   string `json:"message"`
}

type PlayerState struct {
	UserID    string       `json:"userId"`
	Name      string       `json:"name"`
	SeatIndex int          `json:"seatIndex"`
	Stack     int64        `json:"stack"`
	Bet       int64        `json:"bet"`
	TotalBet  int64        `json:"totalBet"`
	Cards     []Card       `json:"-"`
	Status    PlayerStatus `json:"status"`
	Acted     bool         `json:"acted"`
}

type PublicPlayerState struct {
	UserID    string       `json:"userId"`
	Name      string       `json:"name"`
	SeatIndex int          `json:"seatIndex"`
	Stack     int64        `json:"stack"`
	Bet       int64        `json:"bet"`
	TotalBet  int64        `json:"totalBet"`
	Status    PlayerStatus `json:"status"`
	Acted     bool         `json:"acted"`
	Cards     []Card       `json:"cards,omitempty"`
}

type GameState struct {
	HandNumber  int                  `json:"handNumber"`
	StartedAt   time.Time            `json:"startedAt"`
	Phase       Phase                `json:"phase"`
	DealerSeat  int                  `json:"dealerSeat"`
	SmallBlind  int64                `json:"smallBlind"`
	BigBlind    int64                `json:"bigBlind"`
	CurrentTurn int                  `json:"currentTurn"`
	CurrentBet  int64                `json:"currentBet"`
	MinRaise    int64                `json:"minRaise"`
	Pot         int64                `json:"pot"`
	Community   []Card               `json:"community"`
	Players     map[int]*PlayerState `json:"-"`
	Deck        []Card               `json:"-"`
	Winners     []int                `json:"winners,omitempty"`
	Log         []LogEntry           `json:"log,omitempty"`
}
