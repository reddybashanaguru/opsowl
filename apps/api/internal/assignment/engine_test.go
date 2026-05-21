package assignment

import (
	"testing"
)

func TestRuleTypes(t *testing.T) {
	// Verify Rule struct is well-formed
	r := Rule{
		Name:          "Test Rule",
		PriorityOrder: 100,
		IsActive:      true,
	}
	if r.Name != "Test Rule" {
		t.Errorf("unexpected Name: %q", r.Name)
	}
	if r.PriorityOrder != 100 {
		t.Errorf("unexpected PriorityOrder: %d", r.PriorityOrder)
	}
}

func TestMatchInputDefaults(t *testing.T) {
	input := MatchInput{Priority: "high"}
	if input.SourceID != nil {
		t.Error("SourceID should be nil by default")
	}
	if input.Priority != "high" {
		t.Errorf("unexpected Priority: %q", input.Priority)
	}
}
