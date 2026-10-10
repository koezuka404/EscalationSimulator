package entity

import (
	"errors"
	"time"
)

const (
	DemoRunning   = "running"
	DemoStopped   = "stopped"
	DemoCompleted = "completed"
	DemoEven      = "even"
	DemoIncident  = "incident"
)

var (
	ErrDemoRunning             = errors.New("デモはすでに実行中です")
	ErrDemoNotRunning          = errors.New("実行中のデモはありません")
	ErrInvalidDemoCount        = errors.New("作る件数は1件から50件の間で入力してください")
	ErrInvalidDemoInterval     = errors.New("間隔は1秒から10秒の間で入力してください")
	ErrInvalidDemoDistribution = errors.New("出方は均等か障害多めを選んでください")
	ErrDemoNeedsApplicant      = errors.New("デモで使う申請者がいません。申請者を顧客に結びつけてから、もう一度始めてください")
)

type DemoRun struct {
	ID             string
	StartedBy      string
	TotalCount     int
	IntervalSec    int
	Distribution   string
	Status         string
	GeneratedCount int
	StartedAt      time.Time
	StoppedAt      time.Time
}

//件数、間隔、出方を確かめる
func ValidateDemo(count, intervalSec int, distribution string) error {
	if count < 1 || count > 50 {
		return ErrInvalidDemoCount
	}
	if intervalSec < 1 || intervalSec > 10 {
		return ErrInvalidDemoInterval
	}
	switch distribution {
	case DemoEven, DemoIncident:
		return nil
	default:
		return ErrInvalidDemoDistribution
	}
}
