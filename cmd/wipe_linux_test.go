//go:build linux

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fx64b/brenner/internal/flash"
)

// TestFastWipeFallsBackToZeroFillOnRegularFile confirms that when the discard /
// zeroout ioctls don't apply (a plain file, not a block device) fastWipe still
// produces an all-zero result via the streamed fallback.
func TestFastWipeFallsBackToZeroFillOnRegularFile(t *testing.T) {
	const size = 2 * 1024 * 1024
	p := filepath.Join(t.TempDir(), "disk.img")
	if err := os.WriteFile(p, bytes.Repeat([]byte{0xAB}, size), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(p, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	method, err := fastWipe(f, size, func(flash.Progress) {})
	if err != nil {
		t.Fatalf("fastWipe: %v", err)
	}
	if method != "zero-fill" {
		t.Errorf("method = %q, want zero-fill (ioctls do not apply to regular files)", method)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != size {
		t.Fatalf("file size = %d, want %d", len(got), size)
	}
	for i, b := range got {
		if b != 0 {
			t.Fatalf("byte %d = %d, want 0", i, b)
		}
	}
}

func TestDeviceWriterWritesAllBytes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.bin")
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	want := bytes.Repeat([]byte("brenner"), 100000) // ~700 KB across multiple writes
	w := newDeviceWriter(f)
	half := len(want) / 2
	if _, err := w.Write(want[:half]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(want[half:]); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("deviceWriter did not write the expected bytes")
	}
}
