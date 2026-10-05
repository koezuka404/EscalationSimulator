package domain

import (
	"errors"
	"time"
)

const WaitMinutesCap = 99

var (
	ErrInvalidSeverity  = errors.New("緊急度は1、2、3、4のどれかを選んでください")
	ErrInvalidPlanScore = errors.New("プランの点数は1、2、3のどれかにしてください")
)

//緊急度、プラン、超過、待ち時間から点数を返す
func PriorityScore(severity, planScore int, createdAt, now time.Time, slaMinutes int, assigned bool) (int, error) {
	if severity < 1 || severity > 4 {
		return 0, ErrInvalidSeverity
	}
	if planScore < 1 || planScore > 3 {
		return 0, ErrInvalidPlanScore
	}
	if slaMinutes < 1 {
		return 0, ErrInvalidSLAMinutes
	}

	waited := elapsedMinutes(createdAt, now)
	overdue := 0
	if !assigned && waited >= slaMinutes {
		overdue = 1
	}
	if waited > WaitMinutesCap {
		waited = WaitMinutesCap
	}
	return severity*1_000_000 + overdue*10_000 + planScore*100 + waited, nil
}

//点数が同じとき、作った時刻が早い方を先にする
func SameScoreFirst(createdA time.Time, idA string, createdB time.Time, idB string) bool {
	if createdA.Before(createdB) {
		return true
	}
	if createdB.Before(createdA) {
		return false
	}
	return idA < idB
}

//点数が高い方を先にし、同じ点数は作った時刻が早い方を先にする
func WaitingFirst(scoreA int, createdA time.Time, idA string, scoreB int, createdB time.Time, idB string) bool {
	if scoreA != scoreB {
		return scoreA > scoreB
	}
	return SameScoreFirst(createdA, idA, createdB, idB)
}

//待ち時間と、約束時間までの残り、超過を返す
func SLAProgress(createdAt, now time.Time, slaMinutes int, assigned bool) (waited, remaining, overdueMinutes int, overdue bool) {
	waited = elapsedMinutes(createdAt, now)
	if slaMinutes < 1 {
		return waited, 0, 0, false
	}
	if !assigned && waited >= slaMinutes {
		return waited, 0, waited - slaMinutes, true
	}
	return waited, slaMinutes - waited, 0, false
}

func elapsedMinutes(createdAt, now time.Time) int {
	if !now.After(createdAt) {
		return 0
	}
	minutes := int(now.Sub(createdAt).Minutes())
	if minutes < 0 {
		return 0
	}
	return minutes
}
