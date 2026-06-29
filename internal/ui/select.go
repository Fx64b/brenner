package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fx64b/brenner/internal/cache"
	"github.com/fx64b/brenner/internal/device"
)

// Action constants returned by SelectAction.
const (
	ActionFlash = "flash"
	ActionWipe  = "wipe"
)

// ActionLabel renders an action constant for display in the selection history.
func ActionLabel(action string) string {
	switch action {
	case ActionFlash:
		return "Flash ISO"
	case ActionWipe:
		return "Wipe"
	default:
		return action
	}
}

// Wipe mode constants.
const (
	WipeQuick  = "quick" // erase partition table + signatures; instant
	WipeZero   = "zero"  // overwrite every byte with 0x00; slow
	WipeSecure = "secure"
)

// NormalizeWipeMode validates and canonicalises a --mode value (with a few
// friendly aliases).
func NormalizeWipeMode(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", WipeQuick, "fast", "signature":
		return WipeQuick, nil
	case WipeZero, "full", "zerofill", "zero-fill":
		return WipeZero, nil
	case WipeSecure, "random", "rand":
		return WipeSecure, nil
	default:
		return "", fmt.Errorf("unknown wipe mode %q (use quick, zero, or secure)", s)
	}
}

// WipeModeLabel renders a wipe mode for the selection history.
func WipeModeLabel(mode string) string {
	switch mode {
	case WipeQuick:
		return "Quick (partition table + signatures)"
	case WipeZero:
		return "Full zero-fill"
	case WipeSecure:
		return "Secure random overwrite"
	default:
		return mode
	}
}

// SelectWipeMode asks how to wipe the device, spelling out the speed/security
// trade-off of each mode in the option text.
func SelectWipeMode(devicePath string) (string, error) {
	var mode string
	field := huh.NewSelect[string]().
		Title("How should "+devicePath+" be wiped?").
		Options(
			huh.NewOption("Quick — clear partition table & signatures · ~1s · data still recoverable", WipeQuick),
			huh.NewOption("Full — zero-fill every byte · slow · gone by normal means", WipeZero),
			huh.NewOption("Secure — random overwrite · slowest · limited benefit on flash", WipeSecure),
		).
		Value(&mode)
	if err := run(field); err != nil {
		return "", err
	}
	return mode, nil
}

// Sentinel values used internally by the ISO picker. The NUL prefix keeps them
// from ever colliding with a real filesystem path.
const (
	isoScanHome = "\x00scan-home"
	isoManual   = "\x00manual"
)

// ErrAborted is returned when the user cancels a form (esc / ctrl+c).
var ErrAborted = huh.ErrUserAborted

func run(field huh.Field) error {
	return huh.NewForm(huh.NewGroup(field)).WithTheme(Theme()).Run()
}

// SelectDevice asks the user to choose one of the given devices.
func SelectDevice(devices []device.Device) (device.Device, error) {
	var chosen string
	field := huh.NewSelect[string]().
		Title("Select a USB device").
		Options(deviceOptions(devices)...).
		Value(&chosen)
	if err := run(field); err != nil {
		return device.Device{}, err
	}
	for _, d := range devices {
		if d.Path == chosen {
			return d, nil
		}
	}
	return device.Device{}, ErrAborted
}

// SelectAction asks whether to flash or wipe.
func SelectAction() (string, error) {
	var action string
	field := huh.NewSelect[string]().
		Title("What do you want to do?").
		Options(
			huh.NewOption("Flash ISO", ActionFlash),
			huh.NewOption("Wipe", ActionWipe),
		).
		Value(&action)
	return action, run(field)
}

// ISOChoice captures the result of the ISO picker.
type ISOChoice struct {
	Path     string // chosen cached ISO, when set
	ScanHome bool   // user wants to scan ~ for ISOs
	Manual   bool   // user wants to type a path
}

// SelectISO presents cached ISOs plus "scan home" and "enter path" options.
func SelectISO(entries []cache.Entry) (ISOChoice, error) {
	var value string
	field := huh.NewSelect[string]().
		Title("Select an ISO").
		Options(isoOptions(entries)...).
		Value(&value)
	if err := run(field); err != nil {
		return ISOChoice{}, err
	}
	switch value {
	case isoScanHome:
		return ISOChoice{ScanHome: true}, nil
	case isoManual:
		return ISOChoice{Manual: true}, nil
	default:
		return ISOChoice{Path: value}, nil
	}
}

// PickPath asks the user to choose one path from a list (e.g. scan results).
func PickPath(title string, paths []string) (string, error) {
	opts := make([]huh.Option[string], len(paths))
	for i, p := range paths {
		opts[i] = huh.NewOption(p, p)
	}
	var chosen string
	field := huh.NewSelect[string]().Title(title).Options(opts...).Value(&chosen)
	return chosen, run(field)
}

// EnterPath prompts for a free-form path.
func EnterPath(title string) (string, error) {
	var path string
	field := huh.NewInput().
		Title(title).
		Placeholder("/path/to/image.iso").
		Value(&path)
	if err := run(field); err != nil {
		return "", err
	}
	return strings.TrimSpace(path), nil
}

// Confirm shows a yes/no prompt with custom button labels.
func Confirm(title, affirmative, negative string) (bool, error) {
	var ok bool
	field := huh.NewConfirm().
		Title(title).
		Affirmative(affirmative).
		Negative(negative).
		Value(&ok)
	if err := run(field); err != nil {
		return false, err
	}
	return ok, nil
}
