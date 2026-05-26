package comps

import (
	"encoding/json"
	"testing"
)

func TestDeriveConditionFromPropertyConditionFixer(t *testing.T) {
	raw := json.RawMessage(`{"PropertyCondition":"Fixer","PublicRemarks":"Spacious pool home"}`)
	if got := DeriveConditionFromProperty(raw); got != conditionPoor {
		t.Fatalf("got %q want poor", got)
	}
}

func TestDeriveConditionFromPropertyConditionArray(t *testing.T) {
	raw := json.RawMessage(`{"PropertyCondition":["Existing","Fixer"]}`)
	if got := DeriveConditionFromProperty(raw); got != conditionPoor {
		t.Fatalf("got %q want poor", got)
	}
}

func TestDeriveConditionFromPropertyConditionExistingOnly(t *testing.T) {
	raw := json.RawMessage(`{"PropertyCondition":"Existing","PublicRemarks":"Nice home"}`)
	if got := DeriveConditionFromProperty(raw); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestDeriveConditionFromPublicRemarksExcellent(t *testing.T) {
	raw := json.RawMessage(`{"PublicRemarks":"Stunning turnkey estate in mint condition"}`)
	if got := DeriveConditionFromProperty(raw); got != conditionExcellent {
		t.Fatalf("got %q want excellent", got)
	}
}

func TestDeriveConditionFromPublicRemarksPoor(t *testing.T) {
	raw := json.RawMessage(`{"PublicRemarks":"Investor special fixer upper needs major work"}`)
	if got := DeriveConditionFromProperty(raw); got != conditionPoor {
		t.Fatalf("got %q want poor", got)
	}
}

func TestDeriveConditionFromPublicRemarksFair(t *testing.T) {
	raw := json.RawMessage(`{"PublicRemarks":"Sold as-is; needs some tlc and updating"}`)
	if got := DeriveConditionFromProperty(raw); got != conditionFair {
		t.Fatalf("got %q want fair", got)
	}
}

func TestDeriveConditionFromPublicRemarksGood(t *testing.T) {
	raw := json.RawMessage(`{"PublicRemarks":"Well maintained and move-in ready"}`)
	if got := DeriveConditionFromProperty(raw); got != conditionGood {
		t.Fatalf("got %q want good", got)
	}
}

func TestDeriveConditionPropertyConditionWinsOverRemarks(t *testing.T) {
	raw := json.RawMessage(`{"PropertyCondition":"Fixer","PublicRemarks":"Mint condition turnkey"}`)
	if got := DeriveConditionFromProperty(raw); got != conditionPoor {
		t.Fatalf("PropertyCondition Fixer should map to poor, got %q", got)
	}
}

func TestApplyDerivedConditionRespectsExplicitRequest(t *testing.T) {
	explicit := "good"
	sub := SubjectProfile{
		Condition: "good",
		Raw:       json.RawMessage(`{"PublicRemarks":"fixer upper"}`),
	}
	in := SubjectInput{Condition: &explicit}
	applyDerivedCondition(&sub, in)
	if sub.Condition != "good" {
		t.Fatalf("explicit condition overridden: %q", sub.Condition)
	}
}

func TestApplyDerivedConditionFromRaw(t *testing.T) {
	sub := SubjectProfile{Raw: json.RawMessage(`{"PublicRemarks":"Immaculate updated home"}`)}
	applyDerivedCondition(&sub, SubjectInput{})
	if sub.Condition != conditionGood {
		t.Fatalf("got %q want good", sub.Condition)
	}
}
