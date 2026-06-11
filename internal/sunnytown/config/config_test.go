package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsNPCDayLengthToTwentyFourMinutes(t *testing.T) {
	t.Setenv("SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES", "")

	cfg := Load()

	if cfg.NPCDayLength != 24*time.Minute {
		t.Fatalf("NPCDayLength = %s, want 24m", cfg.NPCDayLength)
	}
}

func TestLoadParsesNPCDayLengthMinutes(t *testing.T) {
	t.Setenv("SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES", "8")

	cfg := Load()

	if cfg.NPCDayLength != 8*time.Minute {
		t.Fatalf("NPCDayLength = %s, want 8m", cfg.NPCDayLength)
	}
}

func TestLoadFallsBackForInvalidNPCDayLength(t *testing.T) {
	for _, value := range []string{"0", "-5", "nope"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("SUNNY_TOWN_NPC_DAY_LENGTH_MINUTES", value)

			cfg := Load()

			if cfg.NPCDayLength != 24*time.Minute {
				t.Fatalf("NPCDayLength = %s, want 24m", cfg.NPCDayLength)
			}
		})
	}
}
