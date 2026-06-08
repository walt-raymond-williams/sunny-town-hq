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
)

func npcSchedulePhaseAt(now time.Time) npcSchedulePhase {
	hour := now.UTC().Hour()
	switch {
	case hour >= 6 && hour < 10:
		return npcSchedulePhaseMorning
	case hour >= 10 && hour < 17:
		return npcSchedulePhaseDay
	case hour >= 17 && hour < 21:
		return npcSchedulePhaseEvening
	default:
		return npcSchedulePhaseNight
	}
}

func (npc *liveNPC) scheduleDrivePressure(drive npcDrive, now time.Time) float64 {
	if npc == nil {
		return 0
	}
	switch npcSchedulePhaseAt(now) {
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

func (npc *liveNPC) driveSelectionValue(drive npcDrive, now time.Time) float64 {
	value := npc.driveValue(drive)
	pressure := npc.scheduleDrivePressure(drive, now)
	if pressure <= 0 {
		return value
	}
	if value < npcEmergencyDriveThreshold {
		return value
	}
	return value - pressure
}

func (npc *liveNPC) drivesByUrgency(now time.Time) []npcDrive {
	drives := append([]npcDrive(nil), allNPCDrives...)
	sortNPCDrivesBySelectionValue(npc, drives, now)
	return drives
}

func sortNPCDrivesBySelectionValue(npc *liveNPC, drives []npcDrive, now time.Time) {
	sort.SliceStable(drives, func(i int, j int) bool {
		leftValue := npc.driveSelectionValue(drives[i], now)
		rightValue := npc.driveSelectionValue(drives[j], now)
		if leftValue != rightValue {
			return leftValue < rightValue
		}
		return npc.driveValue(drives[i]) < npc.driveValue(drives[j])
	})
}
