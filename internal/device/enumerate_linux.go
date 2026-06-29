//go:build linux

package device

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// sectorSize is the fixed unit reported by /sys/block/<dev>/size.
const sectorSize = 512

// linuxEnumerator reads removable devices from sysfs. The root paths are fields
// so tests can point them at a fixture tree.
type linuxEnumerator struct {
	sysBlock string // default /sys/block
	devDir   string // default /dev
	mounts   string // default /proc/mounts
}

func platformEnumerator() Enumerator {
	return &linuxEnumerator{sysBlock: "/sys/block", devDir: "/dev", mounts: "/proc/mounts"}
}

func (e *linuxEnumerator) ListRemovable() ([]Device, error) {
	entries, err := os.ReadDir(e.sysBlock)
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, entry := range entries {
		name := entry.Name()
		base := filepath.Join(e.sysBlock, name)

		if readIntFile(filepath.Join(base, "removable")) != 1 {
			continue
		}
		sectors := readUintFile(filepath.Join(base, "size"))
		if sectors == 0 {
			continue
		}

		devices = append(devices, Device{
			Path:  filepath.Join(e.devDir, name),
			Size:  sectors * sectorSize,
			Model: readModel(base),
			Label: readLabel(e.devDir, name),
		})
	}
	return devices, nil
}

// Unmount unmounts every filesystem mounted from the device or one of its
// partitions, deepest mountpoint first.
func (e *linuxEnumerator) Unmount(path string) error {
	f, err := os.Open(e.mounts)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var targets []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if isPartitionOf(fields[0], path) {
			targets = append(targets, unescapeMount(fields[1]))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	sort.Slice(targets, func(i, j int) bool { return len(targets[i]) > len(targets[j]) })
	for _, mp := range targets {
		if err := unmountOne(mp); err != nil {
			return fmt.Errorf("unmount %s: %w", mp, err)
		}
	}
	return nil
}

// unmountSyscall performs a single unmount(2). It is a package variable so the
// busy→lazy retry can be exercised in tests without root.
var unmountSyscall = syscall.Unmount

// unmountOne unmounts a single mountpoint. If a plain unmount fails — typically
// EBUSY because a process still holds the filesystem open — it falls back to a
// lazy MNT_DETACH unmount. The user has already agreed to overwrite the whole
// device, so the filesystem just needs to be detached from the directory tree
// before we write; lazy detach does exactly that even when the mount is busy.
func unmountOne(mountpoint string) error {
	if err := unmountSyscall(mountpoint, 0); err == nil {
		return nil
	}
	return unmountSyscall(mountpoint, syscall.MNT_DETACH)
}

// isPartitionOf reports whether dev is the target device itself or one of its
// partitions (e.g. /dev/sdb1 or /dev/nvme0n1p1 for /dev/nvme0n1).
func isPartitionOf(dev, target string) bool {
	if dev == target {
		return true
	}
	if !strings.HasPrefix(dev, target) {
		return false
	}
	suffix := dev[len(target):]
	suffix = strings.TrimPrefix(suffix, "p")
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func readModel(base string) string {
	model := strings.TrimSpace(readStringFile(filepath.Join(base, "device", "model")))
	vendor := strings.TrimSpace(readStringFile(filepath.Join(base, "device", "vendor")))
	switch {
	case vendor != "" && model != "":
		return vendor + " " + model
	case model != "":
		return model
	default:
		return vendor
	}
}

// readLabel resolves the filesystem label by scanning /dev/disk/by-label for a
// symlink that points back at one of the device's partitions.
func readLabel(devDir, name string) string {
	byLabel := filepath.Join(devDir, "disk", "by-label")
	entries, err := os.ReadDir(byLabel)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		target, err := filepath.EvalSymlinks(filepath.Join(byLabel, entry.Name()))
		if err != nil {
			continue
		}
		if strings.HasPrefix(filepath.Base(target), name) {
			return unescapeUdev(entry.Name())
		}
	}
	return ""
}

// unescapeUdev decodes the \xNN hex escapes udev uses in /dev/disk/by-label
// names, so "Kali\x20Linux" becomes "Kali Linux".
func unescapeUdev(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+3 < len(s) && s[i+1] == 'x' {
			if v, err := strconv.ParseUint(s[i+2:i+4], 16, 8); err == nil {
				b.WriteByte(byte(v))
				i += 4
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func readStringFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func readIntFile(path string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(readStringFile(path)))
	return n
}

func readUintFile(path string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(readStringFile(path)), 10, 64)
	return n
}

// unescapeMount decodes the octal escapes (\040 space, \011 tab, \012 newline,
// \134 backslash) that /proc/mounts uses in mountpoint paths.
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(s)
}
