package model

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoomSettings struct {
	MaxSeats        int   `json:"maxSeats"`
	SmallBlind      int64 `json:"smallBlind"`
	BigBlind        int64 `json:"bigBlind"`
	MinBuyIn        int64 `json:"minBuyIn"`
	MaxBuyIn        int64 `json:"maxBuyIn"`
	TurnTimeoutSecs int   `json:"turnTimeoutSecs"` // 0 = no timeout
}
