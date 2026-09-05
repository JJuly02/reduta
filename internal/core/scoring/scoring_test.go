package scoring

import "testing"

func TestPoints(t *testing.T) {
	cases := map[string]int{
		`{"type":"static","points":500}`:                    500,
		`{"type":"dynamic","initial":500,"minimum":100}`:     500,
		`{"type":"manual"}`:                                  0,
		`{"points":250}`:                                     250,
	}
	for raw, want := range cases {
		if got := Points([]byte(raw)); got != want {
			t.Errorf("Points(%s)=%d want %d", raw, got, want)
		}
	}
}
