package entity

type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

type Customer struct {
	ID         string
	Name       string
	Plan       Plan
	SLAMinutes int
}
