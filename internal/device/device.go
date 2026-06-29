// Package device enumerates removable block devices across platforms and
// exposes a small interface the rest of Brenner builds on. Only the Linux
// implementation is exercised on this host; macOS and Windows ship behind build
// tags, and unsupported platforms return ErrUnsupported.
package device

import (
	"errors"
	"fmt"
	"os"
)

// ErrUnsupported is returned by the enumerator on platforms Brenner cannot yet
// inspect.
var ErrUnsupported = errors.New("brenner: device enumeration is not supported on this platform yet")

// Device describes a single removable block device.
type Device struct {
	Path  string // e.g. /dev/sdb, /dev/disk2, \\.\PhysicalDrive1
	Label string // filesystem label, when one can be determined
	Model string // human-friendly model/vendor string
	Size  uint64 // capacity in bytes
}

// Title returns the most descriptive name available for the device.
func (d Device) Title() string {
	switch {
	case d.Model != "":
		return d.Model
	case d.Label != "":
		return d.Label
	default:
		return "Unknown device"
	}
}

// Enumerator lists removable devices and unmounts them before a write.
type Enumerator interface {
	ListRemovable() ([]Device, error)
	Unmount(path string) error
}

// Default returns the enumerator for the current platform. If the
// BRENNER_FAKE_DEVICES environment variable is set, a FakeEnumerator with sample
// drives is returned instead so the interface can be demoed or screenshotted
// without real hardware.
func Default() Enumerator {
	if os.Getenv("BRENNER_FAKE_DEVICES") != "" {
		return NewFakeEnumerator()
	}
	return platformEnumerator()
}

// HumanSize renders a byte count using decimal (SI) units, the convention
// storage vendors print on the box: e.g. 8000000000 -> "8.0 GB".
func HumanSize(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
