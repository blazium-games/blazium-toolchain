package iso

import "testing"

func TestSanitizeVolumeID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", DefaultVolume},
		{"My Disc!", "MY_DISC"},
		{"hello-world", "HELLO_WORLD"},
		{"abc", "ABC"},
		{stringsRepeat("A", 40), stringsRepeat("A", 32)},
	}
	for _, c := range cases {
		if got := SanitizeVolumeID(c.in); got != c.want {
			t.Fatalf("SanitizeVolumeID(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b = append(b, s[0])
	}
	return string(b)
}
