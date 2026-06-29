package cmd

import (
	"fmt"

	"github.com/fx64b/brenner/internal/device"
	"github.com/fx64b/brenner/internal/ui"
	"github.com/spf13/cobra"
)

var (
	wipeDevice string
	wipeYes    bool
	wipeMode   string
)

var wipeCmd = &cobra.Command{
	Use:   "wipe",
	Short: "Erase a device",
	Long: "Erase a device using one of three modes:\n\n" +
		"  quick   destroy the partition table + filesystem signatures (~1s,\n" +
		"          data remains recoverable) — the default\n" +
		"  zero    overwrite every byte with 0x00 (slow, thorough)\n" +
		"  secure  overwrite with random data (slowest; limited benefit on flash\n" +
		"          due to wear-leveling)\n\n" +
		"On Linux, quick/zero use the device's discard (TRIM) or hardware zeroing\n" +
		"when available.",
	Example: "  brenner wipe --device /dev/sdb\n" +
		"  brenner wipe --device /dev/sdb --mode zero --yes",
	RunE: runWipe,
}

func init() {
	wipeCmd.Flags().StringVar(&wipeDevice, "device", "", "target device path (e.g. /dev/sdb)")
	wipeCmd.Flags().BoolVarP(&wipeYes, "yes", "y", false, "skip the confirmation prompt")
	wipeCmd.Flags().StringVar(&wipeMode, "mode", ui.WipeQuick, "wipe mode: quick, zero, or secure")
}

func runWipe(cmd *cobra.Command, _ []string) error {
	if wipeDevice == "" {
		return runInteractive(cmd)
	}

	mode, err := ui.NormalizeWipeMode(wipeMode)
	if err != nil {
		return err
	}

	enum := newEnumerator()
	size, err := resolveSize(enum, wipeDevice)
	if err != nil {
		return err
	}

	if !wipeYes {
		ok, err := confirmTTY(fmt.Sprintf("Wipe %s (%s) — %s? This DESTROYS all data. [y/N] ",
			wipeDevice, device.HumanSize(size), ui.WipeModeLabel(mode)))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	return doWipe(enum, wipeDevice, size, mode)
}
