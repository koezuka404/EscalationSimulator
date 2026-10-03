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

// PriorityScore は待ち順の点数を返す。保存先も順番メモリも知らない。
// 待ち時間の加点だけ 99 分で頭打ちにする。期限超過は、頭打ちする前の経過分で判定する。
// すでに担当が付いている件は超過にしない。
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

// SameScoreFirst は点数が同じとき、a を b より先にするかを返す。
// 作った時刻が早い方を先にする。時刻も同じなら ID が小さい方を先にする。
// 保存先も順番メモリも知らない。
func SameScoreFirst(createdA time.Time, idA string, createdB time.Time, idB string) bool {
	if createdA.Before(createdB) {
		return true
	}
	if createdB.Before(createdA) {
		return false
	}
	return idA < idB
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
