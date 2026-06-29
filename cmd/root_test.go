package cmd

import "testing"

func TestShouldWarnRoot(t *testing.T) {
	cases := []struct {
		name     string
		euid     int
		elevated bool
		want     bool
	}{
		{"normal user", 1000, false, false},
		{"root, user-invoked", 0, false, true},
		{"root, self-elevated child", 0, true, false},
		{"normal user with stray flag", 1000, true, false},
	}
	for _, c := range cases {
		if got := shouldWarnRoot(c.euid, c.elevated); got != c.want {
			t.Errorf("%s: shouldWarnRoot(%d, %v) = %v, want %v", c.name, c.euid, c.elevated, got, c.want)
		}
	}
}

func TestBuildVersion(t *testing.T) {
	// Without ldflags injection, buildVersion falls back to the module version or
	// "dev" — but never an empty string.
	if got := buildVersion(); got == "" {
		t.Error("buildVersion() returned an empty string")
	}

	prev := version
	version = "v9.9.9"
	t.Cleanup(func() { version = prev })
	if got := buildVersion(); got != "v9.9.9" {
		t.Errorf("buildVersion() = %q, want v9.9.9 when version is set via ldflags", got)
	}
}

func TestElevatedFlagIsHidden(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("elevated")
	if f == nil {
		t.Fatal("expected a persistent --elevated flag")
	}
	if !f.Hidden {
		t.Error("--elevated should be hidden from help output")
	}
}
