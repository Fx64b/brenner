package cmd

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fx64b/brenner/internal/device"
)

func useFakeEnumerator(t *testing.T, fake device.Enumerator) {
	t.Helper()
	prev := newEnumerator
	newEnumerator = func() device.Enumerator { return fake }
	t.Cleanup(func() { newEnumerator = prev })
}

func TestListCommand(t *testing.T) {
	useFakeEnumerator(t, device.NewFakeEnumerator())

	var out bytes.Buffer
	listCmd.SetOut(&out)
	if err := runList(listCmd, nil); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"/dev/sdb", "/dev/sdc", "Samsung Flash Drive", "8.0 GB"} {
		if !strings.Contains(got, want) {
			t.Errorf("list output missing %q\n%s", want, got)
		}
	}
}

func TestListCommandEmpty(t *testing.T) {
	useFakeEnumerator(t, &device.FakeEnumerator{})

	var out bytes.Buffer
	listCmd.SetOut(&out)
	if err := runList(listCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No removable devices") {
		t.Errorf("expected empty message, got %q", out.String())
	}
}

// TestFlashToFileEndToEnd flashes an image to a regular .img file (no root, no
// real device) and confirms the verify path passes and the bytes match.
func TestFlashToFileEndToEnd(t *testing.T) {
	dir := t.TempDir()
	isoPath := filepath.Join(dir, "test.iso")
	imgPath := filepath.Join(dir, "out.img")

	iso := make([]byte, 3*1024*1024+512)
	for i := range iso {
		iso[i] = byte(i % 251)
	}
	if err := os.WriteFile(isoPath, iso, 0o644); err != nil {
		t.Fatal(err)
	}

	fake := device.NewFakeEnumerator()
	// requireRoot=false: target is a regular file, not a block device.
	if err := doFlash(fake, imgPath, isoPath, uint64(len(iso)), true, false); err != nil {
		t.Fatalf("doFlash: %v", err)
	}

	if len(fake.Unmounted) != 1 || fake.Unmounted[0] != imgPath {
		t.Errorf("expected unmount of %q, got %v", imgPath, fake.Unmounted)
	}

	got, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(got) != sha256.Sum256(iso) {
		t.Error("flashed image does not match source ISO")
	}
}

// TestOfferCacheSkipsWhenNotTTY confirms the cache prompt stays out of the way
// when stdout is not an interactive terminal (scripts, the elevated child).
func TestOfferCacheSkipsWhenNotTTY(t *testing.T) {
	home := t.TempDir()
	t.Setenv("BRENNER_HOME", home)

	iso := filepath.Join(t.TempDir(), "fresh.iso")
	if err := os.WriteFile(iso, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	offerCache(iso) // go test stdout is not a TTY, so this must neither prompt nor save

	entries, _ := os.ReadDir(home)
	if len(entries) != 0 {
		t.Errorf("offerCache wrote to the cache without an interactive terminal: %v", entries)
	}
}

func TestResolveSizeFromEnumerator(t *testing.T) {
	size, err := resolveSize(device.NewFakeEnumerator(), "/dev/sdb")
	if err != nil {
		t.Fatal(err)
	}
	if want := uint64(8 * 1000 * 1000 * 1000); size != want {
		t.Errorf("size = %d, want %d", size, want)
	}
}

func TestResolveSizeRegularFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.img")
	if err := os.WriteFile(p, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	size, err := resolveSize(device.NewFakeEnumerator(), p)
	if err != nil {
		t.Fatal(err)
	}
	if size != 4096 {
		t.Errorf("size = %d, want 4096", size)
	}
}
