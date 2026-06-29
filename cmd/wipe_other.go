//go:build !linux

package cmd

import (
	"os"

	"github.com/fx64b/brenner/internal/flash"
)

// fastWipe on non-Linux platforms streams zeros; the block-device discard/zero
// ioctls are Linux-specific.
func fastWipe(f *os.File, size uint64, report func(flash.Progress)) (string, error) {
	return "zero-fill", flash.Wipe(f, size, flash.DefaultBlockSize, report)
}

// quickWipe on non-Linux platforms zeros the head and tail (no discard / partition
// re-read ioctls available here).
func quickWipe(f *os.File, size uint64, report func(flash.Progress)) (string, error) {
	if err := quickWipeCore(f, size, report); err != nil {
		return "", err
	}
	return "quick — signatures cleared", nil
}
