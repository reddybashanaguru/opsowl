package alerts

import "strings"

// matchedRule holds the result of rule evaluation for an incoming alert.
type matchedRule struct {
	rule     AlertRuleResponse
	priority string
	tags     []string
	mode     string // queue | auto
}

// matchRule returns the first active rule (ordered by sort_order asc) that
// matches the alert's type and risk score.
func matchRule(rules []AlertRuleResponse, alert NormalisedAlert) *matchedRule {
	for _, r := range rules {
		if !r.IsActive {
			continue
		}
		typeMatch := r.AlertType == "*" ||
			strings.EqualFold(r.AlertType, alert.AlertType)
		scoreMatch := alert.RiskScore >= r.MinRiskScore
		if typeMatch && scoreMatch {
			priority := r.PriorityMapping
			if priority == "" {
				priority = "medium"
			}
			return &matchedRule{
				rule:     r,
				priority: priority,
				tags:     r.Tags,
				mode:     r.Mode,
			}
		}
	}
	// No rule matched — default to queue mode, medium priority
	return &matchedRule{
		priority: riskLevelToPriority(alert.RiskLevel),
		tags:     []string{},
		mode:     "queue",
	}
}

func riskLevelToPriority(level string) string {
	switch strings.ToLower(level) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}
