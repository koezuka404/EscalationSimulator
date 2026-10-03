package domain

import (
	"errors"
	"time"
)

const WaitMinutesCap = 99

var (
	ErrInvalidSeverity  = errors.New("緊急度は1〜4です")
	ErrInvalidPlanScore = errors.New("プラン点数は1〜3です")
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
