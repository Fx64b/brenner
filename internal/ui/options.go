package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fx64b/brenner/internal/cache"
	"github.com/fx64b/brenner/internal/device"
)

// DeviceLabel formats a device for a select list, e.g.
// "[8.0 GB]  /dev/sdb — Samsung Flash Drive".
func DeviceLabel(d device.Device) string {
	return fmt.Sprintf("[%s]  %s — %s", device.HumanSize(d.Size), d.Path, d.Title())
}

func deviceOptions(devices []device.Device) []huh.Option[string] {
	opts := make([]huh.Option[string], len(devices))
	for i, d := range devices {
		opts[i] = huh.NewOption(DeviceLabel(d), d.Path)
	}
	return opts
}

// IsoLabel formats a cached ISO entry, e.g. "Cached: arch.iso (842.5 MB)".
func IsoLabel(e cache.Entry) string {
	return fmt.Sprintf("Cached: %s (%s)", e.Name, device.HumanSize(e.Size))
}

func isoOptions(entries []cache.Entry) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(entries)+2)
	for _, e := range entries {
		opts = append(opts, huh.NewOption(IsoLabel(e), e.Path))
	}
	opts = append(opts,
		huh.NewOption("Scan home directory (~)", isoScanHome),
		huh.NewOption("Enter path manually", isoManual),
	)
	return opts
}
