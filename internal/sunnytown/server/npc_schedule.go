package server

import (
	"sort"
	"time"
)

type npcSchedulePhase string

const (
	npcSchedulePhaseMorning npcSchedulePhase = "morning"
	npcSchedulePhaseDay     npcSchedulePhase = "day"
	npcSchedulePhaseEvening npcSchedulePhase = "evening"
	npcSchedulePhaseNight   npcSchedulePhase = "night"

	npcScheduleMajorPressure = 55.0
	npcScheduleMinorPressure = 30.0

	defaultNPCScheduleDayLength = 24 * time.Minute
)

func npcSchedulePhaseAt(now time.Time, dayLength time.Duration) npcSchedulePhase {
	if dayLength <= 0 {
		dayLength = defaultNPCScheduleDayLength
	}
	dayNanos := dayLength.Nanoseconds()
	elapsedNanos := now.UTC().UnixNano() % dayNanos
	if elapsedNanos < 0 {
		elapsedNanos += dayNanos
	}
	elapsedRatio := float64(elapsedNanos) / float64(dayNanos)
	switch {
	case elapsedRatio < 0.25:
		return npcSchedulePhaseMorning
	case elapsedRatio < 0.60:
		return npcSchedulePhaseDay
	case elapsedRatio < 0.80:
		return npcSchedulePhaseEvening
	default:
		return npcSchedulePhaseNight
	}
}

func (npc *liveNPC) scheduleDrivePressure(drive npcDrive, now time.Time, dayLength time.Duration) float64 {
	if npc == nil {
		return 0
	}
	switch npcSchedulePhaseAt(now, dayLength) {
	case npcSchedulePhaseMorning:
		if drive == npcDriveHunger && npc.hasScheduleAnchor(npcDriveHunger) {
			return npcScheduleMinorPressure
		}
	case npcSchedulePhaseDay:
		if drive == npcDriveWork && npc.hasScheduleAnchor(npcDriveWork) {
			return npcScheduleMajorPressure
		}
	case npcSchedulePhaseEvening:
		if drive == npcDriveSocial && npc.hasScheduleAnchor(npcDriveSocial) {
			return npcScheduleMajorPressure
		}
	case npcSchedulePhaseNight:
		if drive == npcDriveEnergy && npc.hasScheduleAnchor(npcDriveEnergy) {
			return npcScheduleMajorPressure
		}
	}
	return 0
}

func (npc *liveNPC) hasScheduleAnchor(drive npcDrive) bool {
	if npc == nil {
		return false
	}
	for _, anchor := range npc.anchors.forDrive(drive) {
		if anchor != nil && anchor.isStrongRoutineAnchor() {
			return true
		}
	}
	return false
}

func (anchor *npcLocationAnchor) isStrongRoutineAnchor() bool {
	if anchor == nil {
		return false
	}
	return anchor.Source == npcAnchorSourceOwner || anchor.Source == npcAnchorSourceRole
}

func (npc *liveNPC) driveSelectionValue(drive npcDrive, now time.Time, dayLength time.Duration) float64 {
	value := npc.driveValue(drive)
	pressure := npc.scheduleDrivePressure(drive, now, dayLength)
	if pressure <= 0 {
		return value
	}
	if value < npcEmergencyDriveThreshold {
		return value
	}
	return value - pressure
}

func (npc *liveNPC) drivesByUrgency(now time.Time, dayLength time.Duration) []npcDrive {
	drives := append([]npcDrive(nil), allNPCDrives...)
	sortNPCDrivesBySelectionValue(npc, drives, now, dayLength)
	return drives
}

func sortNPCDrivesBySelectionValue(npc *liveNPC, drives []npcDrive, now time.Time, dayLength time.Duration) {
	sort.SliceStable(drives, func(i int, j int) bool {
		leftValue := npc.driveSelectionValue(drives[i], now, dayLength)
		rightValue := npc.driveSelectionValue(drives[j], now, dayLength)
		if leftValue != rightValue {
			return leftValue < rightValue
		}
		return npc.driveValue(drives[i]) < npc.driveValue(drives[j])
	})
}
