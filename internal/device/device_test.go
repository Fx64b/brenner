package device

import "testing"

func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1000, "1.0 kB"},
		{8_000_000_000, "8.0 GB"},
		{16_022_241_280, "16.0 GB"},
		{1_500_000_000_000, "1.5 TB"},
	}
	for _, c := range cases {
		if got := HumanSize(c.in); got != c.want {
			t.Errorf("HumanSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDeviceTitle(t *testing.T) {
	cases := []struct {
		dev  Device
		want string
	}{
		{Device{Model: "Samsung", Label: "ARCH"}, "Samsung"},
		{Device{Label: "ARCH"}, "ARCH"},
		{Device{}, "Unknown device"},
	}
	for _, c := range cases {
		if got := c.dev.Title(); got != c.want {
			t.Errorf("Title(%+v) = %q, want %q", c.dev, got, c.want)
		}
	}
}

func TestFakeEnumerator(t *testing.T) {
	f := NewFakeEnumerator()
	devs, err := f.ListRemovable()
	if err != nil {
		t.Fatal(err)
	}
	if len(devs) != 2 {
		t.Fatalf("want 2 devices, got %d", len(devs))
	}
	if err := f.Unmount("/dev/sdb"); err != nil {
		t.Fatal(err)
	}
	if len(f.Unmounted) != 1 || f.Unmounted[0] != "/dev/sdb" {
		t.Errorf("Unmount not recorded: %v", f.Unmounted)
	}
}
