package ps1

import "testing"

func TestHostOK(t *testing.T) {
	cases := []struct {
		goos string
		ok   bool
	}{
		{"windows", true},
		{"linux", true},
		{"darwin", false},
		{"js", false},
		{"", false},
	}
	for _, c := range cases {
		if got := hostOK(c.goos); got != c.ok {
			t.Fatalf("hostOK(%q)=%v want %v", c.goos, got, c.ok)
		}
	}
}

func TestCompileAssetsForOS(t *testing.T) {
	if n := len(compileAssetsFor("windows")); n == 0 {
		t.Fatal("windows assets empty")
	}
	linux := compileAssetsFor("linux")
	if len(linux) == 0 {
		t.Fatal("linux assets empty")
	}
	for _, a := range linux {
		if a.SHA256 == "" {
			t.Fatalf("linux %s missing SHA256", a.ID)
		}
	}
	if n := len(compileAssetsFor("darwin")); n != 0 {
		t.Fatalf("darwin must not get Linux zips, got %d", n)
	}
}

func TestRequireHostMatchesRuntime(t *testing.T) {
	err := requireHost()
	if HostSupported() && err != nil {
		t.Fatal(err)
	}
	if !HostSupported() && err == nil {
		t.Fatal("expected unsupported host error")
	}
}
