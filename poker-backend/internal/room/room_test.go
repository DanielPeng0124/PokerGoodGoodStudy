package room

import (
	"strings"
	"testing"

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

	seen := map[int]bool{}
	for i := 0; i < 6; i++ {
		dealer := rm.Game.DealerSeat
		hand := rm.Game.HandNumber
		seen[dealer] = true
		// fold until hand ends; whoever's turn it is folds
		for rm.Game != nil && rm.Game.HandNumber == hand {
			actor := rm.Game.CurrentTurn
			actorID := rm.Game.Players[actor].UserID
			if err := rm.Action(actorID, game.Action{Type: game.ActionFold}); err != nil {
				t.Fatalf("hand %d fold: %v", hand, err)
			}
		}
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

	dealers := []int{rm.Game.DealerSeat}

	// Simulate "10 seconds up, auto-fold SB" four times via SkipCurrentTurn.
	for i := 0; i < 4; i++ {
		hand := rm.Game.HandNumber
		// Keep skipping until this hand ends.
		for rm.Game != nil && rm.Game.HandNumber == hand {
			if err := rm.SkipCurrentTurn(); err != nil {
				t.Fatalf("SkipCurrentTurn hand %d: %v", hand, err)
			}
		}
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
