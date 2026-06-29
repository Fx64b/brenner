package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAndList(t *testing.T) {
	srcDir := t.TempDir()
	iso := filepath.Join(srcDir, "arch.iso")
	writeFile(t, iso, "iso-contents")

	c := New(t.TempDir())
	dst, err := c.Save(iso)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(dst) != c.Dir() {
		t.Errorf("saved to %q, want under %q", dst, c.Dir())
	}
	if b, err := os.ReadFile(dst); err != nil || string(b) != "iso-contents" {
		t.Fatalf("cached file wrong: %q err=%v", b, err)
	}

	entries, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "arch.iso" {
		t.Fatalf("List = %+v", entries)
	}
	if entries[0].Size != uint64(len("iso-contents")) {
		t.Errorf("size = %d, want %d", entries[0].Size, len("iso-contents"))
	}
}

func TestListSortsAndFilters(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "zeta.iso"), "z")
	writeFile(t, filepath.Join(dir, "alpha.iso"), "a")
	writeFile(t, filepath.Join(dir, "notes.txt"), "ignore me")
	writeFile(t, filepath.Join(dir, "UPPER.ISO"), "u") // case-insensitive extension

	entries, err := New(dir).List()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	want := []string{"UPPER.ISO", "alpha.iso", "zeta.iso"}
	if !equalStrings(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestListMissingDir(t *testing.T) {
	entries, err := New(filepath.Join(t.TempDir(), "nope")).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty, got %v", entries)
	}
}

func TestDefaultUsesBrennerHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRENNER_HOME", dir)
	c, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if c.Dir() != dir {
		t.Errorf("Dir = %q, want %q", c.Dir(), dir)
	}
}

func TestUserHomeDirFallback(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; SUDO_USER resolution differs")
	}
	got, err := UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want, _ := os.UserHomeDir()
	if got != want {
		t.Errorf("UserHomeDir = %q, want %q", got, want)
	}
}

func TestSaveSkipsCopyWhenAlreadyCached(t *testing.T) {
	dir := t.TempDir()
	iso := filepath.Join(dir, "already.iso")
	writeFile(t, iso, "x")

	dst, err := New(dir).Save(iso)
	if err != nil {
		t.Fatal(err)
	}
	if dst != iso {
		t.Errorf("dst = %q, want %q", dst, iso)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
