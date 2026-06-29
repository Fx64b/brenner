// Package cmd wires Brenner's cobra command tree: the interactive root plus the
// flash, wipe and list subcommands.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/fx64b/brenner/internal/device"
	"github.com/fx64b/brenner/internal/ui"
	"github.com/spf13/cobra"
)

// Build metadata. These are set via -ldflags at release time (GoReleaser);
// otherwise the version falls back to the module version embedded by `go install`.
var (
	version = ""
	commit  = ""
	date    = ""
)

// buildVersion resolves the version string: an ldflags-injected value wins,
// then the module version recorded by `go install module@version`, else "dev".
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// newEnumerator is the seam tests use to inject a fake device enumerator.
var newEnumerator = device.Default

// elevated is set (via the hidden --elevated flag) on the child process Brenner
// re-execs through sudo, so the "don't run as root" warning is shown only when a
// user invokes brenner as root directly.
var elevated bool

var rootCmd = &cobra.Command{
	Use:   "brenner",
	Short: "Burn ISO images onto USB drives from your terminal",
	Long: ui.Banner() + "\n\n" +
		"Brenner flashes ISO images onto USB drives and wipes them - a fast,\n" +
		"native, terminal-first alternative to heavyweight GUI burners.\n\n" +
		"Run with no arguments for an interactive flow, or use the flags on\n" +
		"`flash`/`wipe` to script it.",
	SilenceUsage:     true,
	SilenceErrors:    true,
	PersistentPreRun: func(_ *cobra.Command, _ []string) { warnIfRoot() },
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runInteractive(cmd)
	},
}

// shouldWarnRoot reports whether to nag about running as root: euid is 0 and this
// is not the child process we elevated ourselves.
func shouldWarnRoot(euid int, elevated bool) bool {
	return euid == 0 && !elevated
}

// warnIfRoot advises against running Brenner as root, unless this is the child
// process we elevated ourselves. Brenner only needs root for the write itself,
// which it handles via sudo on demand.
func warnIfRoot() {
	if !shouldWarnRoot(os.Geteuid(), elevated) {
		return
	}
	fmt.Fprintln(os.Stderr,
		ui.WarnStyle.Render("Warning: running Brenner as root is not recommended.")+"\n   "+
			ui.SubtitleStyle.Render("Run it as your normal user - Brenner asks for sudo only when it needs to write."))
}

// Execute runs the root command and handles top-level error reporting.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if !errors.Is(err, errSilent) {
			fmt.Fprintln(os.Stderr, ui.WarnStyle.Render("error:")+" "+err.Error())
		}
		os.Exit(1)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Brenner version",
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "brenner %s\n", buildVersion())
		if commit != "" {
			fmt.Fprintf(out, "commit: %s\n", commit)
		}
		if date != "" {
			fmt.Fprintf(out, "built:  %s\n", date)
		}
		return nil
	},
}

func init() {
	rootCmd.Version = buildVersion()
	rootCmd.AddCommand(flashCmd, wipeCmd, listCmd, versionCmd)
	rootCmd.SetVersionTemplate("brenner {{.Version}}\n")
	rootCmd.PersistentFlags().BoolVar(&elevated, "elevated", false, "internal: set when Brenner re-executes itself via sudo")
	_ = rootCmd.PersistentFlags().MarkHidden("elevated")
}
