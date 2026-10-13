package numberjs

import "testing"

func TestFixed2MatchesMain(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		in, want float64
	}{{"binary below tie", 2.675, 2.67}, {"exact tie", 2.625, 2.63}, {"negative tie", -2.625, -2.63}, {"small fraction", 1.005, 1}, {"zero", 0, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Fixed2(tc.in); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
