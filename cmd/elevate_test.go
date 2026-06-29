package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShouldElevateRegularFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	p := filepath.Join(t.TempDir(), "x.img")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if shouldElevate(p) {
		t.Error("a regular .img file must not require elevation")
	}
}

func TestShouldElevateDeviceNode(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	if runtime.GOOS == "windows" {
		t.Skip("posix device semantics only")
	}
	if _, err := os.Stat("/dev/null"); err != nil {
		t.Skip("/dev/null unavailable")
	}
	if !shouldElevate("/dev/null") {
		t.Error("a device node must require elevation when not root")
	}
}

// TestElevateBuildsSudoCommand exercises the full elevation path with the sudo
// runner stubbed, so it asserts the reconstructed `sudo brenner ...` command
// without actually escalating.
func TestElevateBuildsSudoCommand(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	if runtime.GOOS == "windows" {
		t.Skip("posix only")
	}
	if _, err := os.Stat("/dev/null"); err != nil {
		t.Skip("/dev/null unavailable")
	}

	var gotName string
	var gotArgs []string
	prevRunner, prevLook := elevateRunner, sudoLookPath
	elevateRunner = func(name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	}
	sudoLookPath = func() (string, error) { return "/usr/bin/sudo", nil }
	t.Cleanup(func() { elevateRunner, sudoLookPath = prevRunner, prevLook })

	childArgs := []string{"flash", "--device", "/dev/null", "--iso", "/tmp/a.iso", "--yes"}
	handled, err := elevateIfNeeded("/dev/null", childArgs)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v, want true, nil", handled, err)
	}
	if gotName != "/usr/bin/sudo" {
		t.Errorf("runner invoked %q, want /usr/bin/sudo", gotName)
	}
	self, _ := os.Executable()
	if len(gotArgs) == 0 || gotArgs[0] != self {
		t.Fatalf("first sudo arg = %v, want self %q", gotArgs, self)
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"flash", "--device /dev/null", "--iso /tmp/a.iso", "--yes", "--elevated"} {
		if !strings.Contains(joined, want) {
			t.Errorf("sudo args %q missing %q", joined, want)
		}
	}
}

func TestElevateSudoNotFound(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	if runtime.GOOS == "windows" {
		t.Skip("posix only")
	}
	if _, err := os.Stat("/dev/null"); err != nil {
		t.Skip("/dev/null unavailable")
	}

	prevLook := sudoLookPath
	sudoLookPath = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { sudoLookPath = prevLook })

	handled, err := elevateIfNeeded("/dev/null", []string{"wipe", "--device", "/dev/null", "--yes"})
	if !handled {
		t.Fatal("expected handled=true when sudo is missing")
	}
	if err == nil || !strings.Contains(err.Error(), "sudo was not found") {
		t.Errorf("err = %v, want a 'sudo was not found' message", err)
	}
}
