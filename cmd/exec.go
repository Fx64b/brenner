package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fx64b/brenner/internal/cache"
	"github.com/fx64b/brenner/internal/device"
	"github.com/fx64b/brenner/internal/flash"
	"github.com/fx64b/brenner/internal/ui"
)

// doFlash writes isoPath onto devicePath, showing progress and (optionally)
// verifying afterwards. requireRoot gates the euid check so tests can target a
// regular file without privileges.
func doFlash(enum device.Enumerator, devicePath, isoPath string, size uint64, verify, requireRoot bool) error {
	if requireRoot {
		isoAbs, err := filepath.Abs(isoPath)
		if err != nil {
			isoAbs = isoPath
		}
		childArgs := []string{"flash", "--device", devicePath, "--iso", isoAbs, "--yes"}
		if !verify {
			childArgs = append(childArgs, "--no-verify")
		}
		if handled, err := elevateIfNeeded(devicePath, childArgs); handled {
			return err
		}
	}
	if err := ensureWritable(devicePath, requireRoot); err != nil {
		return err
	}
	if err := enum.Unmount(devicePath); err != nil {
		return fmt.Errorf("unmount %s: %w", devicePath, err)
	}

	src, err := os.Open(isoPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(devicePath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("Writing %s → %s", filepath.Base(isoPath), devicePath)
	op := func(report func(flash.Progress)) error {
		// newDeviceWriter triggers incremental writeback on Linux so the page
		// cache stays bounded and the final Sync doesn't stall for minutes.
		if _, err := flash.Copy(newDeviceWriter(dst), src, size, flash.DefaultBlockSize, report); err != nil {
			return err
		}
		return dst.Sync()
	}
	if err := runProgress(title, size, op); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	fmt.Println(ui.OkStyle.Render("Write complete."))

	if verify {
		ok, err := verifyWrite(devicePath, isoPath, size)
		if err != nil {
			return fmt.Errorf("verification error: %w", err)
		}
		if !ok {
			return errors.New("verification FAILED: device contents do not match the image")
		}
		fmt.Println(ui.OkStyle.Render("Verified OK."))
	}

	fmt.Println(ui.OkStyle.Render(fmt.Sprintf("✓ Done — %s burned to %s. Safe to remove the drive.",
		filepath.Base(isoPath), devicePath)))
	return nil
}

// doWipe erases devicePath using the chosen mode (quick, zero or secure).
func doWipe(enum device.Enumerator, devicePath string, size uint64, mode string) error {
	childArgs := []string{"wipe", "--device", devicePath, "--mode", mode, "--yes"}
	if handled, err := elevateIfNeeded(devicePath, childArgs); handled {
		return err
	}
	if err := ensureWritable(devicePath, true); err != nil {
		return err
	}
	if err := enum.Unmount(devicePath); err != nil {
		return fmt.Errorf("unmount %s: %w", devicePath, err)
	}

	dst, err := os.OpenFile(devicePath, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	var method string
	op := func(report func(flash.Progress)) error {
		var werr error
		method, werr = wipeByMode(dst, size, mode, report)
		if werr != nil {
			return werr
		}
		return dst.Sync()
	}
	if err := runProgress(wipeTitle(mode, devicePath), size, op); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	fmt.Println(ui.OkStyle.Render("Wipe complete (" + method + ")."))
	if mode == ui.WipeQuick {
		fmt.Println(ui.SubtitleStyle.Render("The drive is reusable now; file data may still be recoverable — use --mode zero or secure to overwrite it."))
	}
	return nil
}

func wipeTitle(mode, devicePath string) string {
	switch mode {
	case ui.WipeQuick:
		return "Quick-wiping " + devicePath
	case ui.WipeSecure:
		return "Securely wiping " + devicePath + " (random)"
	default:
		return "Wiping " + devicePath + " (zero-fill)"
	}
}

// ensureWritable verifies the target can be written. When it is a real block
// device and requireRoot is set, root is required.
func ensureWritable(path string, requireRoot bool) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // a new regular file (e.g. an .img) will be created
		}
		return err
	}
	isBlockDevice := info.Mode()&os.ModeDevice != 0
	if isBlockDevice && requireRoot && os.Geteuid() != 0 {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("writing to %s requires Administrator — re-run from an elevated prompt", path)
		}
		return fmt.Errorf("writing to %s requires root — re-run with sudo", path)
	}
	return nil
}

// errSilent signals that a child process (or sudo) already reported the failure,
// so the top level should exit non-zero without printing anything further.
var errSilent = errors.New("brenner: silent failure")

// Seams for tests: command execution and sudo discovery.
var (
	elevateRunner = runInheritingStdio
	sudoLookPath  = func() (string, error) { return exec.LookPath("sudo") }
)

func runInheritingStdio(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// shouldElevate reports whether writing to devicePath requires re-executing under
// sudo: we are not root, this is not Windows, and the path is an existing device
// node. Regular files (e.g. .img targets) never need elevation.
func shouldElevate(devicePath string) bool {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		return false
	}
	info, err := os.Stat(devicePath)
	return err == nil && info.Mode()&os.ModeDevice != 0
}

// elevateIfNeeded re-executes brenner under sudo when the target is a real device
// node and we lack root, letting the system prompt for the password. It returns
// handled=true when a child process was run (its result is in err) or when
// elevation was required but impossible; in either case the caller must stop and
// return err. Otherwise it returns handled=false and the caller proceeds.
func elevateIfNeeded(devicePath string, childArgs []string) (handled bool, err error) {
	if !shouldElevate(devicePath) {
		return false, nil
	}

	sudo, lookErr := sudoLookPath()
	if lookErr != nil {
		return true, fmt.Errorf("writing to %s requires root, but sudo was not found — re-run as root", devicePath)
	}
	self, exeErr := os.Executable()
	if exeErr != nil {
		return true, exeErr
	}

	fmt.Println(ui.SubtitleStyle.Render("Root required to write to " + devicePath + " — you may be prompted for your password."))

	// --elevated marks the child so it skips the "don't run as root" warning.
	args := append([]string{self}, childArgs...)
	args = append(args, "--elevated")
	if runErr := elevateRunner(sudo, args...); runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return true, errSilent // sudo or the child already explained the failure
		}
		return true, fmt.Errorf("could not escalate with sudo: %w", runErr)
	}
	return true, nil
}

func verifyWrite(devicePath, isoPath string, size uint64) (bool, error) {
	dev, err := os.Open(devicePath)
	if err != nil {
		return false, err
	}
	defer dev.Close()

	src, err := os.Open(isoPath)
	if err != nil {
		return false, err
	}
	defer src.Close()

	var ok bool
	runErr := runProgress("Verifying "+devicePath, size, func(report func(flash.Progress)) error {
		var verr error
		ok, verr = flash.Verify(dev, src, size, report)
		return verr
	})
	return ok, runErr
}

// offerCache asks (only on an interactive terminal) whether to copy a freshly
// flashed ISO into the cache for quick reuse. It is a no-op when the image is
// already cached or stdout is not a terminal, so it stays out of the way in
// scripts and in the elevated child process.
func offerCache(isoPath string) {
	if !isTerminal(os.Stdout) {
		return
	}
	store, err := cache.Default()
	if err != nil {
		return
	}
	entries, _ := store.List()
	if isCached(entries, isoPath) {
		return
	}

	save, err := ui.Confirm(
		fmt.Sprintf("Save %s to %s for quick access next time?", filepath.Base(isoPath), store.Dir()),
		"Save", "No thanks")
	if err != nil || !save {
		return
	}
	if _, err := store.Save(isoPath); err != nil {
		fmt.Fprintln(os.Stderr, ui.WarnStyle.Render("could not cache ISO:")+" "+err.Error())
		return
	}
	fmt.Println(ui.OkStyle.Render("Saved to " + store.Dir() + "."))
}

// resolveSize determines a device's capacity: it prefers the enumerated size and
// falls back to a regular file's length (so .img targets work in tests).
func resolveSize(enum device.Enumerator, path string) (uint64, error) {
	devices, listErr := enum.ListRemovable()
	if listErr == nil {
		for _, d := range devices {
			if d.Path == path {
				return d.Size, nil
			}
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.Mode().IsRegular() {
		return uint64(info.Size()), nil
	}
	// A whole block device that isn't flagged removable: read its size straight
	// from sysfs, which is world-readable (so this works before we elevate).
	if size, ok := blockDeviceSize(path); ok {
		return size, nil
	}
	return 0, fmt.Errorf("could not determine the size of %s", path)
}

// runProgress shows the animated bar on a terminal, or prints plain percentage
// lines when stdout is redirected.
func runProgress(title string, total uint64, op ui.WriteOp) error {
	if isTerminal(os.Stdout) {
		return ui.RunWithProgress(title, total, op)
	}
	return ui.PlainRun(os.Stdout, title, total, op)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// confirmTTY reads a y/N answer from stdin (used by flag mode without --yes).
func confirmTTY(prompt string) (bool, error) {
	fmt.Print(ui.WarnStyle.Render(prompt))
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}
