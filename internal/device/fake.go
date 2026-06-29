package device

// FakeEnumerator is an in-memory Enumerator used for tests, demos and
// screenshots. It records every path passed to Unmount.
type FakeEnumerator struct {
	Devices    []Device
	Unmounted  []string
	UnmountErr error
}

// NewFakeEnumerator returns a FakeEnumerator preloaded with a couple of
// realistic-looking sample drives.
func NewFakeEnumerator() *FakeEnumerator {
	return &FakeEnumerator{
		Devices: []Device{
			{Path: "/dev/sdb", Label: "ARCH_202606", Model: "Samsung Flash Drive", Size: 8 * 1000 * 1000 * 1000},
			{Path: "/dev/sdc", Label: "KINGSTON", Model: "Kingston DataTraveler", Size: 16 * 1000 * 1000 * 1000},
		},
	}
}

// ListRemovable returns the preloaded devices.
func (f *FakeEnumerator) ListRemovable() ([]Device, error) { return f.Devices, nil }

// Unmount records the path and returns the configured error (nil by default).
func (f *FakeEnumerator) Unmount(path string) error {
	f.Unmounted = append(f.Unmounted, path)
	return f.UnmountErr
}
