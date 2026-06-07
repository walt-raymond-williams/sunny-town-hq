package pet

import (
	"testing"

	petv1 "hq/proto/hq/pet/v1"
)

func TestClampStat(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{name: "below range", value: -3, want: 0},
		{name: "in range", value: 42, want: 42},
		{name: "above range", value: 125, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClampStat(tt.value); got != tt.want {
				t.Fatalf("ClampStat(%d) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeGameResult(t *testing.T) {
	tests := []struct {
		name      string
		score     int
		stars     int
		wantScore int
		wantStars int
	}{
		{name: "negative values", score: -5, stars: -2, wantScore: 0, wantStars: 0},
		{name: "stars cap", score: 150, stars: 150, wantScore: 100, wantStars: 100},
		{name: "score capped to collected stars", score: 12, stars: 7, wantScore: 7, wantStars: 7},
		{name: "valid result", score: 7, stars: 9, wantScore: 7, wantStars: 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScore, gotStars := NormalizeGameResult(tt.score, tt.stars)
			if gotScore != tt.wantScore || gotStars != tt.wantStars {
				t.Fatalf("NormalizeGameResult(%d, %d) = (%d, %d), want (%d, %d)", tt.score, tt.stars, gotScore, gotStars, tt.wantScore, tt.wantStars)
			}
		})
	}
}

func TestGameHappinessDelta(t *testing.T) {
	tests := []struct {
		score int
		want  int
	}{
		{score: 0, want: 0},
		{score: 7, want: 7},
		{score: 9, want: 8},
		{score: 10, want: 20},
		{score: 20, want: 20},
	}

	for _, tt := range tests {
		if got := GameHappinessDelta(tt.score); got != tt.want {
			t.Fatalf("GameHappinessDelta(%d) = %d, want %d", tt.score, got, tt.want)
		}
	}
}

func TestProtoMood(t *testing.T) {
	tests := []struct {
		mood string
		want petv1.PetMood
	}{
		{mood: "sleeping", want: petv1.PetMood_PET_MOOD_SLEEPING},
		{mood: "hungry", want: petv1.PetMood_PET_MOOD_HUNGRY},
		{mood: "sad", want: petv1.PetMood_PET_MOOD_SAD},
		{mood: "happy", want: petv1.PetMood_PET_MOOD_HAPPY},
		{mood: "unknown", want: petv1.PetMood_PET_MOOD_IDLE},
	}

	for _, tt := range tests {
		if got := ProtoMood(tt.mood); got != tt.want {
			t.Fatalf("ProtoMood(%q) = %v, want %v", tt.mood, got, tt.want)
		}
	}
}
