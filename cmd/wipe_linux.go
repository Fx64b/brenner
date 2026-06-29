//go:build linux

package cmd

import (
	"io"
	"os"
	"unsafe"

	"github.com/fx64b/brenner/internal/flash"
	"golang.org/x/sys/unix"
)

// Block-device ioctls. blkDiscard/blkZeroout take a [2]uint64{start, length} in
// bytes; blkRRPart takes no argument.
const (
	blkDiscard = 0x1277 // BLKDISCARD - _IO(0x12, 119)
	blkZeroout = 0x127f // BLKZEROOUT - _IO(0x12, 127)
	blkRRPart  = 0x125f // BLKRRPART  - _IO(0x12, 95)
)

// quickWipe destroys the partition table and filesystem signatures as fast as
// possible: discard the whole device when supported (best effort), zero the head
// and tail, then ask the kernel to re-read the (now empty) partition table.
func quickWipe(f *os.File, size uint64, report func(flash.Progress)) (string, error) {
	fd := f.Fd()
	discarded := blkRangeIoctl(fd, blkDiscard, 0, size) == nil

	if err := quickWipeCore(f, size, report); err != nil {
		return "", err
	}

	// Best effort: drop the stale partition table from the kernel so lsblk etc.
	// immediately show the device as blank. Fails harmlessly on busy/plain files.
	_, _, _ = unix.Syscall(unix.SYS_IOCTL, fd, blkRRPart, 0)

	if discarded {
		return "quick - discarded (TRIM)", nil
	}
	return "quick - signatures cleared", nil
}

func blkRangeIoctl(fd, req uintptr, start, length uint64) error {
	rng := [2]uint64{start, length}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(unsafe.Pointer(&rng[0]))); errno != 0 {
		return errno
	}
	return nil
}

// fastWipe clears a device as quickly as the hardware allows, returning the
// method used:
//
//  1. BLKDISCARD the whole device - near-instant when the controller supports
//     TRIM/discard (common on SSDs and USB3 sticks), which is the big win over a
//     full zero-fill.
//  2. BLKZEROOUT in chunks - hardware-accelerated zeroing where available, else
//     an efficient in-kernel zero-fill, with progress per chunk.
//  3. A portable streamed zero-fill - only reached for plain files (e.g. .img
//     targets) where the ioctls don't apply.
func fastWipe(f *os.File, size uint64, report func(flash.Progress)) (string, error) {
	fd := f.Fd()

	if blkRangeIoctl(fd, blkDiscard, 0, size) == nil {
		report(flash.Progress{Written: size, Total: size})
		return "discard/TRIM", nil
	}

	if err := zeroOutAll(fd, size, report); err == nil {
		return "BLKZEROOUT", nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return "zero-fill", flash.Wipe(newDeviceWriter(f), size, flash.DefaultBlockSize, report)
}

func zeroOutAll(fd uintptr, size uint64, report func(flash.Progress)) error {
	const chunk = 1 << 30 // 1 GiB per ioctl keeps progress responsive
	for off := uint64(0); off < size; {
		n := min(uint64(chunk), size-off)
		if err := blkRangeIoctl(fd, blkZeroout, off, n); err != nil {
			return err
		}
		off += n
		report(flash.Progress{Written: off, Total: size})
	}
	return nil
}
