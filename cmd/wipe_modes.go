package cmd

import (
	"crypto/rand"
	"io"
	"os"

	"github.com/fx64b/brenner/internal/flash"
	"github.com/fx64b/brenner/internal/ui"
)

// quickWipeRegion is zeroed at each end of the device for a quick wipe. 8 MiB
// comfortably covers the MBR, the primary GPT (header + entries), most
// filesystem superblocks, and - at the tail - the backup GPT.
const quickWipeRegion = 8 << 20

// wipeByMode dispatches to the chosen wipe strategy and returns a short
// description of what was actually done (for the completion message).
func wipeByMode(dst *os.File, size uint64, mode string, report func(flash.Progress)) (string, error) {
	switch mode {
	case ui.WipeQuick:
		return quickWipe(dst, size, report)
	case ui.WipeSecure:
		return "random overwrite", secureWipe(dst, size, report)
	default: // ui.WipeZero
		return fastWipe(dst, size, report)
	}
}

// secureWipe overwrites the whole device with cryptographically-random bytes.
// The USB write speed, not the CSPRNG, is the bottleneck.
func secureWipe(dst *os.File, size uint64, report func(flash.Progress)) error {
	return flash.Overwrite(newDeviceWriter(dst), size, flash.DefaultBlockSize, func(b []byte) error {
		_, err := rand.Read(b)
		return err
	}, report)
}

// quickWipeCore zeros the first and last quickWipeRegion bytes of the device,
// which is enough to destroy the partition table and filesystem signatures so
// the OS sees a blank device. It is shared by every platform's quickWipe.
func quickWipeCore(f *os.File, size uint64, report func(flash.Progress)) error {
	head := min(uint64(quickWipeRegion), size)
	if err := zeroRegion(f, 0, head); err != nil {
		return err
	}

	if size > uint64(quickWipeRegion) {
		tail := uint64(quickWipeRegion)
		if err := zeroRegion(f, int64(size-tail), tail); err != nil {
			return err
		}
	}

	if report != nil {
		report(flash.Progress{Written: size, Total: size})
	}
	return nil
}

func zeroRegion(f *os.File, offset int64, length uint64) error {
	if length == 0 {
		return nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	return flash.Wipe(f, length, flash.DefaultBlockSize, nil)
}
