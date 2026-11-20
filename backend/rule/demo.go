package rule

import "escalator/entity"

//件数、間隔、出方を確かめる
func ValidateDemo(count, intervalSec int, distribution string) error {
	if count < 1 || count > 50 {
		return ErrInvalidDemoCount
	}
	if intervalSec < 1 || intervalSec > 10 {
		return ErrInvalidDemoInterval
	}
	switch distribution {
	case entity.DemoEven, entity.DemoIncident:
		return nil
	default:
		return ErrInvalidDemoDistribution
	}
}
