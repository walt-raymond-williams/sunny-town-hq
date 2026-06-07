package pet

import (
	"time"

	petv1 "hq/proto/hq/pet/v1"
)

const (
	DailyDecayPoints      = 144.0
	SleepRecoveryDuration = 10 * time.Minute
	DecayTickInterval     = 5 * time.Minute
	GameTargetScore       = 10
	GameEnergyCost        = 5
	GameMaxStarReward     = 100
)

func ClampStat(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func NormalizeGameResult(score int, starsCollected int) (int, int) {
	if score < 0 {
		score = 0
	}
	if starsCollected < 0 {
		starsCollected = 0
	}
	if starsCollected > GameMaxStarReward {
		starsCollected = GameMaxStarReward
	}
	if score > starsCollected {
		score = starsCollected
	}
	return score, starsCollected
}

func GameHappinessDelta(score int) int {
	if score >= GameTargetScore {
		delta := score * 2
		if delta > 20 {
			return 20
		}
		return delta
	}
	if score > 8 {
		return 8
	}
	return score
}

func ProtoMood(mood string) petv1.PetMood {
	switch mood {
	case "sleeping":
		return petv1.PetMood_PET_MOOD_SLEEPING
	case "hungry":
		return petv1.PetMood_PET_MOOD_HUNGRY
	case "sad":
		return petv1.PetMood_PET_MOOD_SAD
	case "happy":
		return petv1.PetMood_PET_MOOD_HAPPY
	default:
		return petv1.PetMood_PET_MOOD_IDLE
	}
}
