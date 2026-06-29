//go:build linux

package device

import (
	"path/filepath"
	"syscall"
	"testing"
)

func testEnumerator() *linuxEnumerator {
	return &linuxEnumerator{
		sysBlock: filepath.Join("testdata", "sys", "block"),
		devDir:   "/dev",
		mounts:   filepath.Join("testdata", "proc", "mounts"),
	}
}

func TestLinuxListRemovable(t *testing.T) {
	devices, err := testEnumerator().ListRemovable()
	if err != nil {
		t.Fatal(err)
	}

	byPath := make(map[string]Device, len(devices))
	for _, d := range devices {
		byPath[d.Path] = d
	}

	if len(devices) != 2 {
		t.Fatalf("expected 2 removable devices, got %d: %+v", len(devices), devices)
	}

	sdb, ok := byPath["/dev/sdb"]
	if !ok {
		t.Fatal("expected /dev/sdb in results")
	}
	if want := uint64(15728640) * 512; sdb.Size != want {
		t.Errorf("sdb size = %d, want %d", sdb.Size, want)
	}
	if sdb.Model != "Generic Flash Drive" {
		t.Errorf("sdb model = %q, want %q", sdb.Model, "Generic Flash Drive")
	}

	if sdc, ok := byPath["/dev/sdc"]; !ok {
		t.Error("expected /dev/sdc in results")
	} else if sdc.Model != "Kingston DataTraveler" {
		t.Errorf("sdc model = %q, want %q", sdc.Model, "Kingston DataTraveler")
	}

	if _, ok := byPath["/dev/nvme0n1"]; ok {
		t.Error("non-removable nvme0n1 should be excluded")
	}
	if _, ok := byPath["/dev/zram0"]; ok {
		t.Error("zero-size zram0 should be excluded")
	}
}

func TestIsPartitionOf(t *testing.T) {
	cases := []struct {
		dev, target string
		want        bool
	}{
		{"/dev/sdb", "/dev/sdb", true},
		{"/dev/sdb1", "/dev/sdb", true},
		{"/dev/sdb12", "/dev/sdb", true},
		{"/dev/nvme0n1p1", "/dev/nvme0n1", true},
		{"/dev/sdc", "/dev/sdb", false},
		{"/dev/sdbb", "/dev/sdb", false},
		{"/dev/sda", "/dev/sd", false},
	}
	for _, c := range cases {
		if got := isPartitionOf(c.dev, c.target); got != c.want {
			t.Errorf("isPartitionOf(%q, %q) = %v, want %v", c.dev, c.target, got, c.want)
		}
	}
}

func TestUnescapeUdev(t *testing.T) {
	cases := []struct{ in, want string }{
		{"KINGSTON", "KINGSTON"},
		{`Kali\x20Linux\x20amd64\x201`, "Kali Linux amd64 1"},
		{`a\x2db`, "a-b"},
		{`trailing\x2`, `trailing\x2`}, // malformed escape left untouched
	}
	for _, c := range cases {
		if got := unescapeUdev(c.in); got != c.want {
			t.Errorf("unescapeUdev(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUnescapeMount(t *testing.T) {
	if got := unescapeMount(`/mnt/My\040Drive`); got != "/mnt/My Drive" {
		t.Errorf("unescapeMount space = %q", got)
	}
	if got := unescapeMount("/mnt/plain"); got != "/mnt/plain" {
		t.Errorf("unescapeMount plain = %q", got)
	}
}

func TestLinuxUnmountNoMatch(t *testing.T) {
	// /dev/sdz is not in the fixture mounts, so Unmount must be a no-op.
	if err := testEnumerator().Unmount("/dev/sdz"); err != nil {
		t.Errorf("Unmount of unmounted device should be nil, got %v", err)
	}
}

func TestLinuxUnmountMissingMountsFile(t *testing.T) {
	e := &linuxEnumerator{mounts: filepath.Join("testdata", "proc", "does-not-exist")}
	if err := e.Unmount("/dev/sdb"); err != nil {
		t.Errorf("missing mounts file should yield nil, got %v", err)
	}
}

func TestUnmountOnePlainSuccess(t *testing.T) {
	calls := 0
	orig := unmountSyscall
	unmountSyscall = func(string, int) error { calls++; return nil }
	t.Cleanup(func() { unmountSyscall = orig })

	if err := unmountOne("/mnt/x"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected a single plain unmount, got %d calls", calls)
	}
}

func TestUnmountOneFallsBackToLazyWhenBusy(t *testing.T) {
	var flags []int
	orig := unmountSyscall
	unmountSyscall = func(_ string, f int) error {
		flags = append(flags, f)
		if f == 0 {
			return syscall.EBUSY // first attempt: device or resource busy
		}
		return nil // lazy detach succeeds
	}
	t.Cleanup(func() { unmountSyscall = orig })

	if err := unmountOne("/run/media/user/OMARCHY"); err != nil {
		t.Fatalf("unmountOne should succeed via lazy detach, got %v", err)
	}
	if len(flags) != 2 || flags[0] != 0 || flags[1] != syscall.MNT_DETACH {
		t.Errorf("expected plain(0) then MNT_DETACH(%d), got %v", syscall.MNT_DETACH, flags)
	}
}

func TestUnmountOneLazyAlsoFails(t *testing.T) {
	orig := unmountSyscall
	unmountSyscall = func(string, int) error { return syscall.EBUSY }
	t.Cleanup(func() { unmountSyscall = orig })

	if err := unmountOne("/mnt/x"); err == nil {
		t.Error("expected an error when both plain and lazy unmount fail")
	}
}
