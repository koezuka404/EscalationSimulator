package rule

import (
	"time"

	"escalator/entity"
)

//メールと接続元のログインを止めている時間内かを返す
func AttemptLocked(attempt entity.LoginAttempt, now time.Time) bool {
	return !attempt.LockedUntil.IsZero() && now.Before(attempt.LockedUntil)
}
