//go:build linux

package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// blockDeviceSize returns the capacity of a whole block device by reading
// /sys/block/<name>/size (in 512-byte sectors). sysfs is world-readable, so this
// works without root. It returns ok=false for partitions or non-block paths.
func blockDeviceSize(path string) (uint64, bool) {
	name := filepath.Base(path)
	data, err := os.ReadFile(filepath.Join("/sys/block", name, "size"))
	if err != nil {
		return 0, false
	}
	sectors, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || sectors == 0 {
		return 0, false
	}
	return sectors * 512, true
}
