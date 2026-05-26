package comps

import "testing"

func TestValidateRequestRequiresRadius(t *testing.T) {
	err := validateRequest(RunRequest{
		Mode:  "A",
		Scope: ScopeInput{Type: "radius"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRequestZipScope(t *testing.T) {
	err := validateRequest(RunRequest{
		Mode:  "B",
		Scope: ScopeInput{Type: "zip", PostalCodes: []string{"33602"}},
	})
	if err != nil {
		t.Fatal(err)
	}
}
