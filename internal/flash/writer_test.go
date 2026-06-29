package flash

import (
	"bytes"
	"testing"
)

func patternBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i * 7)
	}
	return b
}

func TestCopyWritesAllBytesWithProgress(t *testing.T) {
	// 10 MB plus a remainder, so the final block is partial.
	src := patternBytes(10*1024*1024 + 123)
	var dst bytes.Buffer

	var calls int
	var lastWritten uint64
	monotonic := true
	written, err := Copy(&dst, bytes.NewReader(src), uint64(len(src)), 1024*1024, func(p Progress) {
		calls++
		if p.Written < lastWritten {
			monotonic = false
		}
		lastWritten = p.Written
	})
	if err != nil {
		t.Fatal(err)
	}
	if written != uint64(len(src)) {
		t.Errorf("written = %d, want %d", written, len(src))
	}
	if !bytes.Equal(dst.Bytes(), src) {
		t.Error("destination bytes differ from source")
	}
	if calls == 0 {
		t.Error("progress callback never invoked")
	}
	if !monotonic {
		t.Error("progress was not monotonic")
	}
	if lastWritten != uint64(len(src)) {
		t.Errorf("final progress = %d, want %d", lastWritten, len(src))
	}
}

func TestVerify(t *testing.T) {
	data := []byte("brenner burns bytes brilliantly")
	// The device holds the image plus trailing garbage past the image size.
	device := append(append([]byte{}, data...), []byte("xxxxxxx")...)

	ok, err := Verify(bytes.NewReader(device), bytes.NewReader(data), uint64(len(data)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected verify to pass for matching prefix")
	}

	corrupt := append([]byte{}, device...)
	corrupt[0] ^= 0xff
	ok, err = Verify(bytes.NewReader(corrupt), bytes.NewReader(data), uint64(len(data)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected verify to fail for corrupted device")
	}
}

func TestVerifyReportsProgress(t *testing.T) {
	data := patternBytes(3*1024*1024 + 17)
	device := append(append([]byte{}, data...), []byte("trailing-garbage")...)

	var last uint64
	monotonic := true
	calls := 0
	ok, err := Verify(bytes.NewReader(device), bytes.NewReader(data), uint64(len(data)), func(p Progress) {
		calls++
		if p.Written < last {
			monotonic = false
		}
		last = p.Written
		if p.Total != uint64(len(data)) {
			t.Errorf("progress total = %d, want %d", p.Total, len(data))
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected verify to pass")
	}
	if calls == 0 {
		t.Error("verify reported no progress")
	}
	if !monotonic {
		t.Error("verify progress was not monotonic")
	}
	if last != uint64(len(data)) {
		t.Errorf("final verify progress = %d, want %d", last, len(data))
	}
}

func TestVerifyShortDevice(t *testing.T) {
	data := []byte("0123456789")
	if _, err := Verify(bytes.NewReader(data[:3]), bytes.NewReader(data), uint64(len(data)), nil); err == nil {
		t.Error("expected an error when the device is shorter than the image")
	}
}

func TestWipe(t *testing.T) {
	const size = 5*1024*1024 + 7
	var dst bytes.Buffer
	var lastWritten uint64
	if err := Wipe(&dst, size, 1024*1024, func(p Progress) { lastWritten = p.Written }); err != nil {
		t.Fatal(err)
	}
	if dst.Len() != size {
		t.Fatalf("wiped %d bytes, want %d", dst.Len(), size)
	}
	if lastWritten != size {
		t.Errorf("final progress = %d, want %d", lastWritten, size)
	}
	for i, b := range dst.Bytes() {
		if b != 0 {
			t.Fatalf("byte %d = %d, want 0", i, b)
		}
	}
}

func TestProgressFraction(t *testing.T) {
	if got := (Progress{Written: 50, Total: 200}).Fraction(); got != 0.25 {
		t.Errorf("fraction = %v, want 0.25", got)
	}
	if got := (Progress{Written: 10, Total: 0}).Fraction(); got != 0 {
		t.Errorf("zero-total fraction = %v, want 0", got)
	}
}
