package domain

import "fmt"

type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// ToInt returns numeric value for priority comparison
func (p Priority) ToInt() int {
	switch p {
	case PriorityLow:
		return 1
	case PriorityMedium:
		return 2
	case PriorityHigh:
		return 3
	case PriorityCritical:
		return 4
	default:
		return 2 // default to normal
	}
}

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "medium"
	}
}

func (p Priority) QueueName() string {
	return fmt.Sprintf("jobs_%s", p.String())
}
