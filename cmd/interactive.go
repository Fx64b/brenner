package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fx64b/brenner/internal/cache"
	"github.com/fx64b/brenner/internal/device"
	"github.com/fx64b/brenner/internal/ui"
	"github.com/spf13/cobra"
)

// runInteractive drives the full no-arguments TUI flow: pick device, pick
// action, then flash or wipe.
func runInteractive(_ *cobra.Command) error {
	fmt.Println(ui.Banner())
	fmt.Println()

	enum := newEnumerator()
	devices, err := enum.ListRemovable()
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		fmt.Println(ui.SubtitleStyle.Render("No removable devices found. Plug in a USB drive and try again."))
		return nil
	}

	dev, err := ui.SelectDevice(devices)
	if err != nil {
		return abortOrErr(err)
	}
	fmt.Println(ui.Step("Device", ui.DeviceLabel(dev)))

	action, err := ui.SelectAction()
	if err != nil {
		return abortOrErr(err)
	}
	fmt.Println(ui.Step("Action", ui.ActionLabel(action)))

	switch action {
	case ui.ActionFlash:
		return interactiveFlash(enum, dev)
	case ui.ActionWipe:
		return interactiveWipe(enum, dev)
	}
	return nil
}

func interactiveFlash(enum device.Enumerator, dev device.Device) error {
	store, err := cache.Default()
	if err != nil {
		return err
	}
	entries, _ := store.List()

	isoPath, err := chooseISO(entries)
	if err != nil {
		return abortOrErr(err)
	}
	if isoPath == "" {
		return nil
	}
	fmt.Println(ui.Step("Image", filepath.Base(isoPath)))

	info, err := os.Stat(isoPath)
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf("Flash %s → %s (%s)?  This DESTROYS all data on the device.",
		filepath.Base(isoPath), dev.Path, device.HumanSize(dev.Size))
	ok, err := ui.Confirm(ui.WarnStyle.Render(prompt), "Yes, burn it", "Cancel")
	if err != nil {
		return abortOrErr(err)
	}
	if !ok {
		fmt.Println(ui.SubtitleStyle.Render("Cancelled."))
		return nil
	}

	if err := doFlash(enum, dev.Path, isoPath, uint64(info.Size()), true, true); err != nil {
		return err
	}

	offerCache(isoPath)
	return nil
}

func interactiveWipe(enum device.Enumerator, dev device.Device) error {
	mode, err := ui.SelectWipeMode(dev.Path)
	if err != nil {
		return abortOrErr(err)
	}
	fmt.Println(ui.Step("Mode", ui.WipeModeLabel(mode)))

	prompt := fmt.Sprintf("Wipe %s (%s) - %s?  This DESTROYS all data on the device.",
		dev.Path, device.HumanSize(dev.Size), ui.WipeModeLabel(mode))
	ok, err := ui.Confirm(ui.WarnStyle.Render(prompt), "Yes, wipe it", "Cancel")
	if err != nil {
		return abortOrErr(err)
	}
	if !ok {
		fmt.Println(ui.SubtitleStyle.Render("Cancelled."))
		return nil
	}
	return doWipe(enum, dev.Path, dev.Size, mode)
}

// chooseISO resolves the ISO source from the cache picker, scanning the home
// directory or asking for a manual path as needed.
func chooseISO(entries []cache.Entry) (string, error) {
	choice, err := ui.SelectISO(entries)
	if err != nil {
		return "", err
	}
	switch {
	case choice.Path != "":
		return choice.Path, nil
	case choice.ScanHome:
		root := scanHomeRoot()
		found, err := scanISOs(root)
		if err != nil {
			return "", err
		}
		if len(found) == 0 {
			fmt.Println(ui.SubtitleStyle.Render("No ISO files found under " + root + "."))
			return ui.EnterPath("Enter the path to an ISO")
		}
		return ui.PickPath("Select an ISO found under "+root, found)
	default:
		return ui.EnterPath("Enter the path to an ISO")
	}
}

// scanHomeRoot picks the directory to search for ISOs. Under sudo, $HOME is
// /root, so it resolves the invoking user's home (via SUDO_USER); failing that
// it falls back to /home so the user's images are still found.
func scanHomeRoot() string {
	home, err := cache.UserHomeDir()
	if err != nil || home == "" || home == "/root" {
		return "/home"
	}
	return home
}

// scanISOs walks root (skipping hidden directories, bounded to maxDepth levels)
// collecting *.iso files.
func scanISOs(root string) ([]string, error) {
	const maxDepth = 4
	rootDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))

	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			if strings.Count(path, string(os.PathSeparator))-rootDepth > maxDepth {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".iso") {
			found = append(found, path)
		}
		return nil
	})
	return found, err
}

func isCached(entries []cache.Entry, path string) bool {
	abs, _ := filepath.Abs(path)
	for _, e := range entries {
		if entryAbs, _ := filepath.Abs(e.Path); entryAbs == abs {
			return true
		}
	}
	return false
}

// abortOrErr turns a user cancellation into a clean exit, passing other errors
// through.
func abortOrErr(err error) error {
	if errors.Is(err, ui.ErrAborted) {
		fmt.Println(ui.SubtitleStyle.Render("Aborted."))
		return nil
	}
	return err
}
