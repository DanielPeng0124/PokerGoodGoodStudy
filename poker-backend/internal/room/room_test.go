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
	if err := rm.Start(); err != nil {
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
	if err := rm.Start(); err != nil {
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
