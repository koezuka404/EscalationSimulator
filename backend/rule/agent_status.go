package rule

import "escalator/entity"

//待機中か離席かを確かめる
func ParseAvailability(raw string) (entity.AgentAvailability, error) {
	switch entity.AgentAvailability(raw) {
	case entity.AgentAvailable, entity.AgentOffline:
		return entity.AgentAvailability(raw), nil
	default:
		return "", ErrInvalidAvailability
	}
}

//稼働を切り替えてよいかを確かめる
func CanSetAvailability(actorRole entity.Role, actorID, targetID string, targetRole entity.Role, next entity.AgentAvailability) error {
	if next != entity.AgentAvailable && next != entity.AgentOffline {
		return ErrInvalidAvailability
	}
	if actorID == targetID {
		if actorRole != entity.RoleAgent {
			return ErrAvailabilityRole
		}
		return nil
	}
	if actorRole != entity.RoleAdmin {
		return ErrAvailabilityOthers
	}
	if targetRole != entity.RoleAgent {
		return ErrAvailabilityRole
	}
	if next != entity.AgentOffline {
		return ErrAvailabilityOfflineOnly
	}
	return nil
}
