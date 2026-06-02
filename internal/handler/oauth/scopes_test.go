package oauth

import "testing"

func TestParseAndValidateScopes_Default(t *testing.T) {
	scopes, err := ParseAndValidateScopes("")
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 4 {
		t.Fatalf("got %v", scopes)
	}
}

func TestParseAndValidateScopes_Unknown(t *testing.T) {
	_, err := ParseAndValidateScopes("monitor bogus")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAndValidateScopes_Dedupes(t *testing.T) {
	scopes, err := ParseAndValidateScopes("api monitor api")
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 {
		t.Fatalf("got %v", scopes)
	}
}

func TestScopeString(t *testing.T) {
	got := ScopeString([]string{"api", "monitor"})
	if got != "api monitor" {
		t.Fatalf("got %q", got)
	}
}
