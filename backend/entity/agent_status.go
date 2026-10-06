package entity

type AgentAvailability string

const (
	AgentAvailable AgentAvailability = "available"
	AgentBusy      AgentAvailability = "busy"
	AgentOffline   AgentAvailability = "offline"
)
