package iso

import "testing"

func TestSanitizeVolumeID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"My Disc!", "MY_DISC"},
		{"hello-world", "HELLO_WORLD"},
		{"abc", "ABC"},
		{stringsRepeat("A", 40), stringsRepeat("A", 32)},
	}
	if got := VolumeOr("", "blazium2"); got != "BLAZIUM2" {
		t.Fatalf("VolumeOr empty: %q", got)
	}
	if got := VolumeOr("My Disc!", "X"); got != "MY_DISC" {
		t.Fatalf("VolumeOr dirty: %q", got)
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
