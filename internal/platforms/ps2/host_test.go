package ps2

import (
	"runtime"
	"testing"
)

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

func TestOfficialTarballURLByOS(t *testing.T) {
	if officialTarballURLFor("windows") == "" || officialTarballURLFor("linux") == "" {
		t.Fatal("windows and linux must have a ps2dev tarball URL")
	}
	if officialTarballURLFor("darwin") != "" {
		t.Fatal("darwin must not get a ps2dev tarball URL")
	}
	if officialTarballURL() != officialTarballURLFor(runtime.GOOS) {
		t.Fatal("officialTarballURL must match this host")
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
