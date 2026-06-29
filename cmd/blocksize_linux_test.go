//go:build linux

package cmd

import (
	"os"
	"testing"
)

func TestBlockDeviceSizeNonexistent(t *testing.T) {
	if _, ok := blockDeviceSize("/dev/null"); ok {
		t.Error("/dev/null has no /sys/block entry; expected ok=false")
	}
}

func TestBlockDeviceSizeReal(t *testing.T) {
	if _, err := os.Stat("/sys/block/zram0/size"); err != nil {
		t.Skip("/sys/block/zram0 unavailable on this host")
	}
	size, ok := blockDeviceSize("/dev/zram0")
	if !ok || size == 0 {
		t.Errorf("blockDeviceSize(/dev/zram0) = %d, ok=%v; want a nonzero size", size, ok)
	}
}
