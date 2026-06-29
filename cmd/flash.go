package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flashDevice   string
	flashISO      string
	flashYes      bool
	flashNoVerify bool
)

var flashCmd = &cobra.Command{
	Use:   "flash",
	Short: "Flash an ISO image onto a device",
	Long: "Flash an ISO image onto a device.\n\n" +
		"With --device and --iso both supplied, Brenner runs non-interactively;\n" +
		"otherwise it falls through to the interactive flow.",
	Example: "  brenner flash --device /dev/sdb --iso ~/Downloads/arch.iso\n" +
		"  brenner flash --device /dev/sdb --iso arch.iso --yes --no-verify",
	RunE: runFlash,
}

func init() {
	flashCmd.Flags().StringVar(&flashDevice, "device", "", "target device path (e.g. /dev/sdb)")
	flashCmd.Flags().StringVar(&flashISO, "iso", "", "path to the ISO image")
	flashCmd.Flags().BoolVarP(&flashYes, "yes", "y", false, "skip the confirmation prompt")
	flashCmd.Flags().BoolVar(&flashNoVerify, "no-verify", false, "skip post-write SHA-256 verification")
}

func runFlash(cmd *cobra.Command, _ []string) error {
	if flashDevice == "" || flashISO == "" {
		return runInteractive(cmd)
	}

	info, err := os.Stat(flashISO)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not an ISO", flashISO)
	}

	enum := newEnumerator()
	if !flashYes {
		ok, err := confirmTTY(fmt.Sprintf("Flash %s → %s? This DESTROYS all data. [y/N] ", flashISO, flashDevice))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	if err := doFlash(enum, flashDevice, flashISO, uint64(info.Size()), !flashNoVerify, true); err != nil {
		return err
	}

	// In scripted use (--yes) and in the elevated child we stay silent; an
	// interactive flag-mode user still gets the cache offer.
	if !flashYes {
		offerCache(flashISO)
	}
	return nil
}
