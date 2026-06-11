package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestServerMessageIncludesEmptyNPCSnapshots(t *testing.T) {
	payload, err := json.Marshal(ServerMessage{
		Type: "snapshot",
		NPCs: []NPCSnapshot{},
	})
	if err != nil {
		t.Fatalf("marshal server message: %v", err)
	}
	if !strings.Contains(string(payload), `"npcs":[]`) {
		t.Fatalf("payload = %s, want authoritative empty npcs field", payload)
	}
}
