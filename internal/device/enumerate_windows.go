//go:build windows

package device

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// windowsEnumerator queries USB disks via PowerShell. This implementation
// compiles and is believed correct but is not exercised on the Linux build host.
type windowsEnumerator struct{}

func platformEnumerator() Enumerator { return windowsEnumerator{} }

func (windowsEnumerator) ListRemovable() ([]Device, error) {
	const script = `Get-Disk | Where-Object BusType -eq 'USB' | Select-Object Number,FriendlyName,Size | ConvertTo-Json`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	// ConvertTo-Json emits a bare object for a single disk and an array for many.
	if strings.HasPrefix(trimmed, "{") {
		trimmed = "[" + trimmed + "]"
	}

	var raw []struct {
		Number       int
		FriendlyName string
		Size         uint64
	}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, err
	}

	var devices []Device
	for _, r := range raw {
		devices = append(devices, Device{
			Path:  fmt.Sprintf(`\\.\PhysicalDrive%d`, r.Number),
			Model: r.FriendlyName,
			Size:  r.Size,
		})
	}
	return devices, nil
}

// Unmount is a no-op on Windows: the volume is locked and dismounted through the
// raw device handle at write time.
func (windowsEnumerator) Unmount(string) error { return nil }
