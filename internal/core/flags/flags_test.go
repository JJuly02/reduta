package flags

import "testing"

func TestVerify(t *testing.T) {
	specs := []Spec{
		{Hash: Hash("flag{Case_Sensitive}", true), CaseSensitive: true},
		{Hash: Hash("flag{loose}", false), CaseSensitive: false},
	}
	cases := []struct {
		in   string
		want bool
	}{
		{"flag{Case_Sensitive}", true},
		{"  flag{Case_Sensitive}  ", true}, // trimmed
		{"flag{case_sensitive}", false},    // wrong case on a CS flag
		{"FLAG{LOOSE}", true},              // case-insensitive flag
		{"nope", false},
	}
	for _, c := range cases {
		if got := Verify(c.in, specs); got != c.want {
			t.Errorf("Verify(%q)=%v want %v", c.in, got, c.want)
		}
	}
}
