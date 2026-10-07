package entity

import "errors"

type AgentAvailability string

const (
	AgentAvailable AgentAvailability = "available"
	AgentBusy      AgentAvailability = "busy"
	AgentOffline   AgentAvailability = "offline"
)

var (
	ErrInvalidAvailability     = errors.New("稼働は待機中か離席を選んでください")
	ErrAvailabilityRole        = errors.New("稼働を切り替えられるのは担当者だけです")
	ErrAvailabilityOthers      = errors.New("他の担当者の稼働を変えられるのは管理者だけです")
	ErrAvailabilityOfflineOnly = errors.New("他の担当者に指定できるのは離席だけです")
	ErrCannotBecomeAvailable   = errors.New("対応中のチケットがあるため、待機中にはできません。離席にすると、そのチケットは待ちに戻ります")
)

//待機中か離席かを確かめる
func ParseAvailability(raw string) (AgentAvailability, error) {
	switch AgentAvailability(raw) {
	case AgentAvailable, AgentOffline:
		return AgentAvailability(raw), nil
	default:
		return "", ErrInvalidAvailability
	}
}

//稼働を切り替えてよいかを確かめる
func CanSetAvailability(actorRole Role, actorID, targetID string, targetRole Role, next AgentAvailability) error {
	if next != AgentAvailable && next != AgentOffline {
		return ErrInvalidAvailability
	}
	if actorID == targetID {
		if actorRole != RoleAgent {
			return ErrAvailabilityRole
		}
		return nil
	}
	if actorRole != RoleAdmin {
		return ErrAvailabilityOthers
	}
	if targetRole != RoleAgent {
		return ErrAvailabilityRole
	}
	if next != AgentOffline {
		return ErrAvailabilityOfflineOnly
	}
	return nil
}
