//go:build linux

package cmd

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// deviceWriter wraps a block-device file and kicks off asynchronous writeback
// for every range it writes (sync_file_range). This keeps dirty pages bounded
// instead of letting gigabytes pile up in the page cache - which is what makes a
// USB write appear to fly to 100% and then stall for minutes on the final fsync
// (and often freezes the rest of the desktop). The writeback is non-blocking, so
// it costs nothing on the hot path.
type deviceWriter struct {
	f      *os.File
	offset int64
}

func newDeviceWriter(f *os.File) io.Writer { return &deviceWriter{f: f} }

func (w *deviceWriter) Write(p []byte) (int, error) {
	n, err := w.f.Write(p)
	if n > 0 {
		_ = unix.SyncFileRange(int(w.f.Fd()), w.offset, int64(n), unix.SYNC_FILE_RANGE_WRITE)
		w.offset += int64(n)
	}
	return n, err
}
