package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fx64b/brenner/internal/flash"
)

func TestQuickWipeZeroesEndsOnly(t *testing.T) {
	const size = quickWipeRegion*2 + 4<<20 // 8 MiB head + 4 MiB middle + 8 MiB tail
	p := filepath.Join(t.TempDir(), "disk.img")
	if err := os.WriteFile(p, bytes.Repeat([]byte{0xAB}, size), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(p, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	method, err := quickWipe(f, size, func(flash.Progress) {})
	if err != nil {
		t.Fatalf("quickWipe: %v", err)
	}
	if !strings.HasPrefix(method, "quick") {
		t.Errorf("method = %q, want a quick-* method", method)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	zeros := make([]byte, quickWipeRegion)
	if !bytes.Equal(got[:quickWipeRegion], zeros) {
		t.Error("head region was not zeroed")
	}
	if !bytes.Equal(got[size-quickWipeRegion:], zeros) {
		t.Error("tail region was not zeroed")
	}
	if got[size/2] != 0xAB {
		t.Errorf("middle byte = %#x, want 0xAB (must be left untouched)", got[size/2])
	}
}

func TestSecureWipeRandomizes(t *testing.T) {
	const size = 1 << 20
	p := filepath.Join(t.TempDir(), "disk.img")
	if err := os.WriteFile(p, bytes.Repeat([]byte{0xAB}, size), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(p, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := secureWipe(f, size, func(flash.Progress) {}); err != nil {
		t.Fatalf("secureWipe: %v", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != size {
		t.Fatalf("size changed: %d, want %d", len(got), size)
	}
	uniform := true
	for _, b := range got {
		if b != got[0] {
			uniform = false
			break
		}
	}
	if uniform {
		t.Error("secure wipe output is uniform, not random")
	}
	if bytes.Contains(got, bytes.Repeat([]byte{0xAB}, 4096)) {
		t.Error("the original 0xAB pattern survived a secure wipe")
	}
}

func TestWipeByModeRoutes(t *testing.T) {
	const size = 64 << 10
	p := filepath.Join(t.TempDir(), "disk.img")
	if err := os.WriteFile(p, bytes.Repeat([]byte{0xAB}, size), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	method, err := wipeByMode(f, size, "zero", func(flash.Progress) {})
	if err != nil {
		t.Fatal(err)
	}
	if method == "" {
		t.Error("expected a non-empty method description")
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if !bytes.Equal(got, make([]byte, size)) {
		t.Error("zero mode did not produce an all-zero result")
	}
}
