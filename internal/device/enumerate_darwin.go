//go:build darwin

package device

import (
	"os/exec"
	"strconv"
	"strings"
)

// darwinEnumerator shells out to diskutil. This implementation compiles and is
// believed correct but is not exercised on the Linux build host.
type darwinEnumerator struct{}

func platformEnumerator() Enumerator { return darwinEnumerator{} }

func (darwinEnumerator) ListRemovable() ([]Device, error) {
	out, err := exec.Command("diskutil", "list", "external", "physical").Output()
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/dev/disk") && strings.Contains(line, "external") {
			id := strings.Fields(line)[0]
			devices = append(devices, infoForDisk(id))
		}
	}
	return devices, nil
}

func (darwinEnumerator) Unmount(path string) error {
	return exec.Command("diskutil", "unmountDisk", path).Run()
}

func infoForDisk(id string) Device {
	d := Device{Path: id}
	out, err := exec.Command("diskutil", "info", id).Output()
	if err != nil {
		return d
	}
	for _, line := range strings.Split(string(out), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "Device / Media Name":
			d.Model = val
		case "Volume Name":
			if val != "" && !strings.HasPrefix(val, "Not applicable") {
				d.Label = val
			}
		case "Disk Size":
			d.Size = parseDiskSize(val)
		}
	}
	return d
}

// parseDiskSize pulls the exact byte count out of a diskutil "Disk Size" line
// such as "8.0 GB (8004304896 Bytes) (exactly ...)".
func parseDiskSize(val string) uint64 {
	open := strings.Index(val, "(")
	if open == -1 {
		return 0
	}
	fields := strings.Fields(val[open+1:])
	if len(fields) == 0 {
		return 0
	}
	n, _ := strconv.ParseUint(fields[0], 10, 64)
	return n
}
