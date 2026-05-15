package room

import (
	"strings"
	"testing"
	"time"

	"poker-backend/internal/game"
	"poker-backend/internal/model"
)

func TestSitRejectsDuplicateName(t *testing.T) {
	rm := New("user-1", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("user-1", "Alice", 0, 1000); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	err := rm.Sit("user-2", " alice ", 1, 1000)
	if err == nil {
		t.Fatal("duplicate name was allowed")
	}
	if !strings.Contains(err.Error(), "name already taken") {
		t.Fatalf("error = %q, want name already taken", err.Error())
	}
}

func TestNewRoomDefaultsToOneTwoBlinds(t *testing.T) {
	rm := New("user-1", model.RoomSettings{})
	if rm.Settings.SmallBlind != 1 {
		t.Fatalf("small blind = %d, want 1", rm.Settings.SmallBlind)
	}
	if rm.Settings.BigBlind != 2 {
		t.Fatalf("big blind = %d, want 2", rm.Settings.BigBlind)
	}
}

func TestOnlyOwnerCanControlGame(t *testing.T) {
	rm := New("owner", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 1,
		BigBlind:   2,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("owner", "Owner", 0, 1000); err != nil {
		t.Fatalf("sit owner: %v", err)
	}
	if err := rm.Sit("guest", "Guest", 1, 1000); err != nil {
		t.Fatalf("sit guest: %v", err)
	}
	if err := rm.Start("guest"); err == nil {
		t.Fatal("non-owner started game")
	}
	if err := rm.Start("owner"); err != nil {
		t.Fatalf("owner start: %v", err)
	}
	if err := rm.Pause("guest", true); err == nil {
		t.Fatal("non-owner paused game")
	}
	if err := rm.Pause("owner", true); err != nil {
		t.Fatalf("owner pause: %v", err)
	}
	if err := rm.Action("owner", game.Action{Type: game.ActionFold}); err == nil {
		t.Fatal("action while paused was allowed")
	}
	if err := rm.SkipCurrentTurn(); err == nil {
		t.Fatal("auto action while paused was allowed")
	}
	if err := rm.Pause("owner", false); err != nil {
		t.Fatalf("owner resume: %v", err)
	}
	if err := rm.End("guest", rm.Game.HandNumber); err == nil {
		t.Fatal("non-owner ended game")
	}
	if err := rm.End("owner", rm.Game.HandNumber); err != nil {
		t.Fatalf("owner end: %v", err)
	}
	if !rm.Ending {
		t.Fatal("game was not marked to end after hand")
	}
	if rm.Game == nil {
		t.Fatal("active hand was cancelled immediately")
	}
	if err := rm.Action("owner", game.Action{Type: game.ActionFold}); err != nil {
		t.Fatalf("finish ending hand: %v", err)
	}
	if rm.Game != nil {
		t.Fatal("game still active after ending hand finished")
	}
	if rm.Ending {
		t.Fatal("ending flag still set after hand finished")
	}
	if err := rm.Start("owner"); err != nil {
		t.Fatalf("owner restart after end: %v", err)
	}
	if rm.Game == nil {
		t.Fatal("game did not restart after end")
	}
}

func TestEndAfterHandRequestStopsJustAutoStartedNextHand(t *testing.T) {
	rm := New("owner", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 1,
		BigBlind:   2,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("owner", "Owner", 0, 1000); err != nil {
		t.Fatalf("sit owner: %v", err)
	}
	if err := rm.Sit("guest", "Guest", 1, 1000); err != nil {
		t.Fatalf("sit guest: %v", err)
	}
	if err := rm.Start("owner"); err != nil {
		t.Fatalf("owner start: %v", err)
	}
	requestedHand := rm.Game.HandNumber
	if err := rm.Action("owner", game.Action{Type: game.ActionFold}); err != nil {
		t.Fatalf("finish hand before end request arrives: %v", err)
	}
	// Simulate the 2-second display delay elapsing.
	rm.StartDelayedHandIfReady(time.Now().Add(5 * time.Second))
	if rm.Game == nil || rm.Game.HandNumber <= requestedHand {
		t.Fatalf("next hand was not auto-started, game=%#v", rm.Game)
	}
	if err := rm.End("owner", requestedHand); err != nil {
		t.Fatalf("owner end stale hand request: %v", err)
	}
	if rm.Game != nil {
		t.Fatal("just auto-started next hand was not stopped")
	}
	if rm.Ending {
		t.Fatal("ending flag still set after stopping auto-started hand")
	}
}

func TestFinishedHandRecordsHistoryAndLedger(t *testing.T) {
	rm := New("user-1", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("user-1", "Alice", 0, 1000); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	if err := rm.Sit("user-2", "Bob", 1, 1000); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Start("user-1"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := rm.Action("user-1", game.Action{Type: game.ActionFold}); err != nil {
		t.Fatalf("fold: %v", err)
	}

	snapshot := rm.Snapshot("user-1")
	history, ok := snapshot["handHistory"].([]HandRecord)
	if !ok {
		t.Fatalf("hand history missing or wrong type: %#v", snapshot["handHistory"])
	}
	if len(history) != 1 {
		t.Fatalf("hand history length = %d, want 1", len(history))
	}
	if history[0].Pot != 15 {
		t.Fatalf("recorded pot = %d, want 15", history[0].Pot)
	}
	if len(history[0].Winners) != 1 || history[0].Winners[0] != 1 {
		t.Fatalf("winners = %#v, want seat 1", history[0].Winners)
	}

	ledger, ok := snapshot["ledger"].([]LedgerEntry)
	if !ok {
		t.Fatalf("ledger missing or wrong type: %#v", snapshot["ledger"])
	}
	rows := map[string]LedgerEntry{}
	for _, row := range ledger {
		rows[row.UserID] = row
	}
	if rows["user-1"].Net != -5 {
		t.Fatalf("alice net = %d, want -5", rows["user-1"].Net)
	}
	if rows["user-2"].Net != 5 {
		t.Fatalf("bob net = %d, want 5", rows["user-2"].Net)
	}

	summary, ok := snapshot["lastHandSummary"].(HandSummary)
	if !ok {
		t.Fatalf("last hand summary missing or wrong type: %#v", snapshot["lastHandSummary"])
	}
	if summary.Delta != -5 {
		t.Fatalf("alice last hand delta = %d, want -5", summary.Delta)
	}
}

func TestDealerRotatesEachHand(t *testing.T) {
	rm := New("u0", model.RoomSettings{
		MaxSeats:   3,
		SmallBlind: 1,
		BigBlind:   2,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})
	if err := rm.Sit("u0", "P0", 0, 500); err != nil {
		t.Fatalf("sit 0: %v", err)
	}
	if err := rm.Sit("u1", "P1", 1, 500); err != nil {
		t.Fatalf("sit 1: %v", err)
	}
	if err := rm.Sit("u2", "P2", 2, 500); err != nil {
		t.Fatalf("sit 2: %v", err)
	}
	if err := rm.Start("u0"); err != nil {
		t.Fatalf("start: %v", err)
	}

	future := time.Now().Add(10 * time.Second)
	seen := map[int]bool{}
	for i := 0; i < 6; i++ {
		dealer := rm.Game.DealerSeat
		hand := rm.Game.HandNumber
		seen[dealer] = true
		// fold until hand ends; whoever's turn it is folds
		for rm.Game != nil && rm.Game.HandNumber == hand && rm.Game.Phase != "finished" {
			actor := rm.Game.CurrentTurn
			actorID := rm.Game.Players[actor].UserID
			if err := rm.Action(actorID, game.Action{Type: game.ActionFold}); err != nil {
				t.Fatalf("hand %d fold: %v", hand, err)
			}
		}
		// Skip the 2-second display delay.
		rm.StartDelayedHandIfReady(future)
		if rm.Game == nil {
			t.Fatalf("game ended unexpectedly after hand %d", hand)
		}
	}

	// After 6 hands with 3 players all 3 seats must have been dealer at least once.
	for seat := 0; seat < 3; seat++ {
		if !seen[seat] {
			t.Errorf("seat %d was never dealer in 6 hands; dealer sequence may be stuck", seat)
		}
	}
}

func TestDealerRotatesAfterSkipTurn(t *testing.T) {
	rm := New("u0", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 1,
		BigBlind:   2,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})
	if err := rm.Sit("u0", "P0", 0, 500); err != nil {
		t.Fatalf("sit 0: %v", err)
	}
	if err := rm.Sit("u1", "P1", 1, 500); err != nil {
		t.Fatalf("sit 1: %v", err)
	}
	if err := rm.Start("u0"); err != nil {
		t.Fatalf("start: %v", err)
	}

	future := time.Now().Add(10 * time.Second)
	dealers := []int{rm.Game.DealerSeat}

	// Simulate "10 seconds up, auto-fold SB" four times via SkipCurrentTurn.
	for i := 0; i < 4; i++ {
		hand := rm.Game.HandNumber
		// Keep skipping until this hand ends.
		for rm.Game != nil && rm.Game.HandNumber == hand && rm.Game.Phase != "finished" {
			if err := rm.SkipCurrentTurn(); err != nil {
				t.Fatalf("SkipCurrentTurn hand %d: %v", hand, err)
			}
		}
		// Skip the 2-second display delay.
		rm.StartDelayedHandIfReady(future)
		if rm.Game == nil {
			t.Fatalf("game ended unexpectedly after hand %d", hand)
		}
		dealers = append(dealers, rm.Game.DealerSeat)
	}

	// Dealers must alternate: 0,1,0,1,0
	for i := 1; i < len(dealers); i++ {
		if dealers[i] == dealers[i-1] {
			t.Errorf("dealer did not change between hand %d and %d: both seat %d (sequence: %v)", i, i+1, dealers[i], dealers)
		}
	}
}

func TestSnapshotOnlyShowsCardsToMatchingUser(t *testing.T) {
	rm := New("user-1", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("user-1", "Alice", 0, 1000); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	if err := rm.Sit("user-2", "Bob", 1, 1000); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Start("user-1"); err != nil {
		t.Fatalf("start: %v", err)
	}

	aliceSnapshot := rm.Snapshot("user-1")
	aliceGame := aliceSnapshot["game"].(map[string]any)
	alicePlayers := aliceGame["players"].(map[int]game.PublicPlayerState)
	if got := len(alicePlayers[0].Cards); got != 2 {
		t.Fatalf("alice sees %d own cards, want 2", got)
	}
	if got := len(alicePlayers[1].Cards); got != 0 {
		t.Fatalf("alice sees %d bob cards, want 0", got)
	}

	otherSnapshot := rm.Snapshot("different-browser-user")
	otherGame := otherSnapshot["game"].(map[string]any)
	otherPlayers := otherGame["players"].(map[int]game.PublicPlayerState)
	if got := len(otherPlayers[0].Cards); got != 0 {
		t.Fatalf("different user sees %d alice cards, want 0", got)
	}
}

func TestBustedSeatIsVacatedAndBuyInsAccumulate(t *testing.T) {
	rm := New("alice", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("alice", "Alice", 0, 100); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 100); err != nil {
		t.Fatalf("sit bob: %v", err)
	}

	rm.handStartStacks = map[int]int64{0: 100, 1: 100}
	rm.Game = &game.GameState{
		HandNumber: 1,
		StartedAt:  time.Now(),
		Phase:      game.PhaseFinished,
		DealerSeat: 0,
		SmallBlind: 5,
		BigBlind:   10,
		Players:    map[int]*game.PlayerState{0: {UserID: "alice", Name: "Alice", SeatIndex: 0, Stack: 0, Status: game.StatusFolded}, 1: {UserID: "bob", Name: "Bob", SeatIndex: 1, Stack: 200, Status: game.StatusActive}},
		Winners:    []int{1},
		Community:  []game.Card{},
		Log:        []game.LogEntry{},
	}
	rm.syncStacksLocked()
	rm.recordFinishedHandLocked()

	if _, ok := rm.Seats[0]; ok {
		t.Fatal("busted alice seat was not vacated")
	}
	if _, ok := rm.Seats[1]; !ok {
		t.Fatal("bob seat was unexpectedly vacated")
	}
	if err := rm.Sit("alice", "Alice", 0, 150); err != nil {
		t.Fatalf("alice re-sit after bust: %v", err)
	}

	rows := map[string]LedgerEntry{}
	for _, row := range rm.ledgerLocked() {
		rows[row.UserID] = row
	}
	if rows["alice"].BuyIn != 250 {
		t.Fatalf("alice buy-in = %d, want 250", rows["alice"].BuyIn)
	}
	if rows["alice"].CurrentStack != 150 {
		t.Fatalf("alice current stack = %d, want 150", rows["alice"].CurrentStack)
	}
	if rows["alice"].Net != -100 {
		t.Fatalf("alice net = %d, want -100", rows["alice"].Net)
	}
}

func TestLedgerKeepsLatestStackAfterRebuyAndSeatRelease(t *testing.T) {
	rm := New("alice", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("alice", "Alice", 0, 700); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 700); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	rm.handStartStacks = map[int]int64{0: 700, 1: 700}
	rm.Game = &game.GameState{
		HandNumber: 1,
		StartedAt:  time.Now(),
		Phase:      game.PhaseFinished,
		DealerSeat: 0,
		SmallBlind: 5,
		BigBlind:   10,
		Players:    map[int]*game.PlayerState{0: {UserID: "alice", Name: "Alice", SeatIndex: 0, Stack: 0, Status: game.StatusAllIn}, 1: {UserID: "bob", Name: "Bob", SeatIndex: 1, Stack: 1400, Status: game.StatusActive}},
		Winners:    []int{1},
	}
	rm.syncStacksLocked()
	rm.recordFinishedHandLocked()

	if err := rm.Sit("alice", "Alice", 0, 700); err != nil {
		t.Fatalf("alice re-sit: %v", err)
	}
	if err := rm.SetAway("alice", true); err != nil {
		t.Fatalf("alice release seat: %v", err)
	}

	rows := map[string]LedgerEntry{}
	for _, row := range rm.ledgerLocked() {
		rows[row.UserID] = row
	}
	if rows["alice"].BuyIn != 1400 {
		t.Fatalf("alice buy-in = %d, want 1400", rows["alice"].BuyIn)
	}
	if rows["alice"].CurrentStack != 700 {
		t.Fatalf("alice current stack = %d, want 700", rows["alice"].CurrentStack)
	}
	if rows["alice"].Net != -700 {
		t.Fatalf("alice net = %d, want -700", rows["alice"].Net)
	}
	if rows["alice"].BuyIn+rows["alice"].Net != rows["alice"].CurrentStack {
		t.Fatalf("ledger invariant failed: buy-in %d + net %d != stack %d", rows["alice"].BuyIn, rows["alice"].Net, rows["alice"].CurrentStack)
	}
}

func TestBustedSeatIsVacatedAfterRealActionFlow(t *testing.T) {
	rm := New("alice", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("alice", "Alice", 0, 100); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 100); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Start("alice"); err != nil {
		t.Fatalf("start: %v", err)
	}
	rm.Game.Players[0].Cards = []game.Card{{Rank: 4, Suit: game.Spades}, {Rank: 4, Suit: game.Hearts}}
	rm.Game.Players[1].Cards = []game.Card{{Rank: 14, Suit: game.Spades}, {Rank: 14, Suit: game.Hearts}}
	rm.Game.Deck = []game.Card{
		{Rank: 2, Suit: game.Spades},
		{Rank: 2, Suit: game.Clubs}, {Rank: 7, Suit: game.Diamonds}, {Rank: 9, Suit: game.Hearts},
		{Rank: 3, Suit: game.Spades},
		{Rank: 11, Suit: game.Spades},
		{Rank: 5, Suit: game.Spades},
		{Rank: 3, Suit: game.Clubs},
	}

	if err := rm.Action("alice", game.Action{Type: game.ActionAllIn}); err != nil {
		t.Fatalf("alice all-in: %v", err)
	}
	if err := rm.Action("bob", game.Action{Type: game.ActionCall}); err != nil {
		t.Fatalf("bob call: %v", err)
	}

	if _, ok := rm.Seats[0]; ok {
		t.Fatal("busted alice seat was not vacated after real action flow")
	}
	if err := rm.Sit("alice", "Alice", 0, 150); err != nil {
		t.Fatalf("alice re-sit after real bust: %v", err)
	}
}

func TestSnapshotVacatesAlreadyRecordedBustedSeat(t *testing.T) {
	rm := New("alice", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("alice", "Alice", 0, 100); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 100); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	rm.Game = &game.GameState{
		HandNumber: 1,
		StartedAt:  time.Now(),
		Phase:      game.PhaseFinished,
		DealerSeat: 0,
		Players:    map[int]*game.PlayerState{0: {UserID: "alice", Name: "Alice", SeatIndex: 0, Stack: 0, Status: game.StatusAllIn}, 1: {UserID: "bob", Name: "Bob", SeatIndex: 1, Stack: 200, Status: game.StatusActive}},
		Winners:    []int{1},
	}
	rm.lastRecordedHandNumber = 1
	rm.Seats[0].Stack = 0
	rm.Seats[1].Stack = 200

	rm.Snapshot("alice")

	if _, ok := rm.Seats[0]; ok {
		t.Fatal("snapshot did not vacate already-recorded busted seat")
	}
	if err := rm.Sit("alice", "Alice", 0, 150); err != nil {
		t.Fatalf("alice re-sit after snapshot cleanup: %v", err)
	}
}

func TestSitCleansStaleBustedSeatBeforeValidation(t *testing.T) {
	rm := New("alice", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("alice", "Alice", 0, 100); err != nil {
		t.Fatalf("sit alice: %v", err)
	}
	rm.Seats[0].Stack = 0

	if err := rm.Sit("alice", "Alice", 0, 150); err != nil {
		t.Fatalf("alice re-sit after stale bust: %v", err)
	}
	if rm.Seats[0].Stack != 150 {
		t.Fatalf("alice stack = %d, want 150", rm.Seats[0].Stack)
	}
}

func TestAwaySeatIsReleasedWhenOutOfHand(t *testing.T) {
	rm := New("owner", model.RoomSettings{
		MaxSeats:   4,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("owner", "Owner", 0, 1000); err != nil {
		t.Fatalf("sit owner: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 1000); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Sit("carol", "Carol", 2, 1000); err != nil {
		t.Fatalf("sit carol: %v", err)
	}
	if err := rm.SetAway("bob", true); err != nil {
		t.Fatalf("set bob away: %v", err)
	}
	if _, ok := rm.Seats[1]; ok {
		t.Fatal("bob seat was not vacated")
	}
	if err := rm.Sit("dave", "Dave", 1, 500); err != nil {
		t.Fatalf("dave sit in released seat: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 3, 500); err != nil {
		t.Fatalf("bob sit again elsewhere: %v", err)
	}

	rows := map[string]LedgerEntry{}
	for _, row := range rm.ledgerLocked() {
		rows[row.UserID] = row
	}
	if rows["bob"].BuyIn != 1500 {
		t.Fatalf("bob buy-in = %d, want 1500", rows["bob"].BuyIn)
	}
	if rows["bob"].CurrentStack != 1500 {
		t.Fatalf("bob current stack = %d, want 1500", rows["bob"].CurrentStack)
	}
	if rows["bob"].Net != 0 {
		t.Fatalf("bob net = %d, want 0", rows["bob"].Net)
	}
}

func TestAwayDuringActiveHandReleasesSeatAfterHand(t *testing.T) {
	rm := New("owner", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})

	if err := rm.Sit("owner", "Owner", 0, 1000); err != nil {
		t.Fatalf("sit owner: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 1000); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Start("owner"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := rm.SetAway("bob", true); err != nil {
		t.Fatalf("set bob away during hand: %v", err)
	}
	if rm.Seats[1] == nil || !rm.Seats[1].Away {
		t.Fatal("bob was not marked away during active hand")
	}
	if err := rm.End("owner", 0); err != nil {
		t.Fatalf("end after current hand: %v", err)
	}
	if err := rm.Action("owner", game.Action{Type: game.ActionFold}); err != nil {
		t.Fatalf("finish hand: %v", err)
	}
	if rm.Game != nil {
		t.Fatal("game was not ended after the hand")
	}
	if _, ok := rm.Seats[1]; ok {
		t.Fatal("away seat was not released after hand")
	}

	rows := map[string]LedgerEntry{}
	var netSum int64
	for _, row := range rm.ledgerLocked() {
		rows[row.UserID] = row
		netSum += row.Net
	}
	if netSum != 0 {
		t.Fatalf("ledger net sum = %d, want 0; rows = %#v", netSum, rows)
	}
	if rows["owner"].Net != -5 {
		t.Fatalf("owner net = %d, want -5", rows["owner"].Net)
	}
	if rows["bob"].Net != 5 {
		t.Fatalf("bob net = %d, want 5", rows["bob"].Net)
	}
	if rows["bob"].CurrentStack != 1005 {
		t.Fatalf("bob current stack = %d, want 1005", rows["bob"].CurrentStack)
	}
	if rows["bob"].Seated {
		t.Fatal("bob ledger row still shows seated after away release")
	}

	if err := rm.Sit("guest", "Guest", 1, 500); err != nil {
		t.Fatalf("guest sit in released seat: %v", err)
	}
}

func TestTurnTimerAutoCheckOrFold(t *testing.T) {
	rm := New("owner", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})
	if err := rm.Sit("owner", "Owner", 0, 1000); err != nil {
		t.Fatalf("sit owner: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 1000); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Start("owner"); err != nil {
		t.Fatalf("start: %v", err)
	}
	rm.Ending = true
	rm.turnDeadline = time.Now().Add(-time.Second)

	if !rm.AutoActExpired(time.Now()) {
		t.Fatal("expired timer did not auto-act")
	}
	if rm.Game != nil {
		t.Fatal("game did not end after timed-out fold")
	}
	rows := map[string]LedgerEntry{}
	var netSum int64
	for _, row := range rm.ledgerLocked() {
		rows[row.UserID] = row
		netSum += row.Net
	}
	if rows["owner"].Net != -5 {
		t.Fatalf("owner net = %d, want -5", rows["owner"].Net)
	}
	if rows["bob"].Net != 5 {
		t.Fatalf("bob net = %d, want 5", rows["bob"].Net)
	}
	if netSum != 0 {
		t.Fatalf("ledger net sum = %d, want 0", netSum)
	}
}

func TestAddTurnTimeIsLimitedToThreeUsesPerHand(t *testing.T) {
	rm := New("owner", model.RoomSettings{
		MaxSeats:   2,
		SmallBlind: 5,
		BigBlind:   10,
		MinBuyIn:   100,
		MaxBuyIn:   2000,
	})
	if err := rm.Sit("owner", "Owner", 0, 1000); err != nil {
		t.Fatalf("sit owner: %v", err)
	}
	if err := rm.Sit("bob", "Bob", 1, 1000); err != nil {
		t.Fatalf("sit bob: %v", err)
	}
	if err := rm.Start("owner"); err != nil {
		t.Fatalf("start: %v", err)
	}
	before := rm.turnDeadline
	for i := 0; i < maxTurnExtensions; i++ {
		if err := rm.AddTurnTime("owner"); err != nil {
			t.Fatalf("add time %d: %v", i+1, err)
		}
	}
	if !rm.turnDeadline.After(before) {
		t.Fatal("turn deadline did not extend")
	}
	if err := rm.AddTurnTime("owner"); err == nil {
		t.Fatal("fourth add time succeeded")
	}
	timer := rm.turnTimerLocked(time.Now())
	if timer == nil {
		t.Fatal("turn timer missing")
	}
	if timer.ExtensionsUsed != maxTurnExtensions {
		t.Fatalf("extensions used = %d, want %d", timer.ExtensionsUsed, maxTurnExtensions)
	}
}
