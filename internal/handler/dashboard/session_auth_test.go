package dashboard

import "testing"

func TestParseSessionUserID(t *testing.T) {
	cases := []struct {
		in   any
		want int64
		ok   bool
	}{
		{int64(42), 42, true},
		{float64(42), 42, true},
		{"42", 42, true},
		{"", 0, false},
		{nil, 0, false},
	}
	for _, tc := range cases {
		got, ok := parseSessionUserID(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("parseSessionUserID(%v) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
