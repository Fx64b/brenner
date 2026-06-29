//go:build !linux

package cmd

import (
	"io"
	"os"
)

// newDeviceWriter is a no-op wrapper outside Linux; the sync_file_range
// optimization is Linux-specific.
func newDeviceWriter(f *os.File) io.Writer { return f }
