package iso

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func plantVIDEO(t *testing.T, root string) {
	t.Helper()
	v := filepath.Join(root, "VIDEO_TS")
	if err := os.MkdirAll(v, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"VIDEO_TS.IFO", "VIDEO_TS.BUP", "VIDEO_TS.VOB"} {
		if err := os.WriteFile(filepath.Join(v, n), []byte(n+"-payload"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMasterRejectsMissingIFO(t *testing.T) {
	dir := t.TempDir()
	_, err := Master(dir, filepath.Join(dir, "out.iso"), VolumeProps{VolumeID: "X"}, nil)
	if err == nil || !strings.Contains(err.Error(), "VIDEO_TS.IFO") {
		t.Fatalf("got %v", err)
	}
}

func TestMasterBridgeAndProperties(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	out := filepath.Join(t.TempDir(), "disc.iso")
	res, err := Master(src, out, VolumeProps{
		VolumeID:      "MY_DVD",
		Title:         "My Interactive Movie",
		Publisher:     "BLAZIUM",
		CopyrightFile: "",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.VolumeID != "MY_DVD" {
		t.Fatalf("volume %s", res.VolumeID)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	vol, err := ReadPVDVolume(f)
	if err != nil {
		t.Fatal(err)
	}
	if vol != "MY_DVD" {
		t.Fatalf("pvd volume %q", vol)
	}
	pub, err := ReadPVDPublisher(f)
	if err != nil {
		t.Fatal(err)
	}
	if pub != "BLAZIUM" {
		t.Fatalf("publisher %q", pub)
	}
	if err := HasUDFVRS(f); err != nil {
		t.Fatal(err)
	}
	ifo, err := FindISO9660(f, "VIDEO_TS/VIDEO_TS.IFO")
	if err != nil {
		t.Fatal(err)
	}
	if ifo.Size == 0 || ifo.Dir {
		t.Fatalf("ifo %+v", ifo)
	}
	if loc, ok := res.Files["VIDEO_TS/VIDEO_TS.IFO"]; !ok || loc.Sector != uint64(ifo.Sector) {
		t.Fatalf("shared extent iso=%d udf=%v", ifo.Sector, loc)
	}
}

func TestMasterExtrasAndLicense(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	shots := filepath.Join(t.TempDir(), "shots")
	if err := os.MkdirAll(filepath.Join(shots, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shots, "a.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shots, "nested", "b.png"), []byte("png2"), 0o644); err != nil {
		t.Fatal(err)
	}
	autorun := filepath.Join(t.TempDir(), "AUTORUN.INF")
	if err := os.WriteFile(autorun, []byte("[autorun]\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("recursive", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "r.iso")
		res, err := MasterFromMeta(DiscMeta{
			Dir:       src,
			Out:       out,
			Volume:    "X",
			License:   "MIT License",
			Copyright: "Copyright 2026",
			Recursive: true,
			Extras: []Extra{
				{Host: autorun},
				{Host: shots, Disc: "SHOTS"},
			},
		}, out)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := FindISO9660(f, "LICENSE.TXT"); err != nil {
			t.Fatal(err)
		}
		if _, err := FindISO9660(f, "COPYRIGH.TXT"); err != nil {
			t.Fatal(err)
		}
		if _, ok := res.Files["COPYRIGHT.TXT"]; !ok {
			t.Fatalf("udf missing COPYRIGHT.TXT in %v", res.Files)
		}
		if _, err := FindISO9660(f, "AUTORUN.INF"); err != nil {
			t.Fatal(err)
		}
		if _, err := FindISO9660(f, "SHOTS/A.PNG"); err != nil {
			t.Fatal(err)
		}
		if _, err := FindISO9660(f, "SHOTS/NESTED/B.PNG"); err != nil {
			t.Fatal(err)
		}
		if len(res.Extras) < 3 {
			t.Fatalf("extras %v", res.Extras)
		}
	})

	t.Run("nonrecursive", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "nr.iso")
		_, err := MasterFromMeta(DiscMeta{
			Dir:    src,
			Out:    out,
			Volume: "X",
			Extras: []Extra{{Host: shots, Disc: "SHOTS", Recursive: false}},
		}, out)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := FindISO9660(f, "SHOTS/A.PNG"); err != nil {
			t.Fatal(err)
		}
		if _, err := FindISO9660(f, "SHOTS/NESTED/B.PNG"); err == nil {
			t.Fatal("nested file should be omitted")
		}
	})
}

func TestExtraProtectsVideoTS(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	f := filepath.Join(t.TempDir(), "x.bin")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := MasterFromMeta(DiscMeta{
		Dir:    src,
		Out:    filepath.Join(t.TempDir(), "o.iso"),
		Extras: []Extra{{Host: f, Disc: "VIDEO_TS/X.BIN"}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "VIDEO_TS") {
		t.Fatalf("got %v", err)
	}
}

func TestDuplicateExtra(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	f := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(f, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := MasterFromMeta(DiscMeta{
		Dir: src,
		Out: filepath.Join(t.TempDir(), "o.iso"),
		Extras: []Extra{
			{Host: f, Disc: "A.TXT"},
			{Host: f, Disc: "A.TXT"},
		},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("got %v", err)
	}
}

func TestDiscSetAndNoCopyrightFile(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	out := filepath.Join(t.TempDir(), "d.iso")
	_, err := MasterFromMeta(DiscMeta{Dir: src, Out: out, Volume: "X", Disc: 2, Discs: 3}, out)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := FindISO9660(f, "COPYRIGHT.TXT"); err == nil {
		t.Fatal("copyright should be absent")
	}
}

func TestAutorunConflict(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	inf := filepath.Join(t.TempDir(), "AUTORUN.INF")
	if err := os.WriteFile(inf, []byte("[autorun]\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := MasterFromMeta(DiscMeta{
		Dir:     src,
		Out:     filepath.Join(t.TempDir(), "o.iso"),
		Autorun: Autorun{Label: "Game"},
		Extras:  []Extra{{Host: inf}},
	}, "")
	if err == nil || !strings.Contains(err.Error(), "AUTORUN") {
		t.Fatalf("got %v", err)
	}
}

func TestMetaRoundTrip(t *testing.T) {
	src := t.TempDir()
	plantVIDEO(t, src)
	metaPath := filepath.Join(t.TempDir(), "disc.interdvd.json")
	m := InitMeta()
	m.Dir = src
	m.Out = filepath.Join(t.TempDir(), "frommeta.iso")
	m.Title = "From Meta"
	m.License = "MIT"
	if err := WriteMeta(metaPath, m); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadMeta(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "From Meta" {
		t.Fatalf("%+v", loaded)
	}
	res, err := MasterFromMeta(loaded, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Title != "From Meta" {
		t.Fatalf("title %s", res.Title)
	}
	if err := ValidateMeta(loaded); err != nil {
		t.Fatal(err)
	}
	bad := InitMeta()
	bad.Dir = t.TempDir()
	if err := ValidateMeta(bad); err == nil {
		t.Fatal("expected missing IFO")
	}
}

func TestBadDiscSet(t *testing.T) {
	_, err := DiscMeta{Dir: "x", Disc: 3, Discs: 2}.Resolve()
	if err == nil {
		t.Fatal("expected invalid disc set")
	}
}
