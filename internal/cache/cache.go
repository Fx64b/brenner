// Package cache manages the per-user ISO store at ~/.brenner. It is deliberately
// a plain directory of .iso files — no database — so it stays inspectable and
// trivially testable.
package cache

import (
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
)

const isoExt = ".iso"

// Cache is a directory holding cached ISO images.
type Cache struct {
	dir string
}

// New returns a Cache rooted at dir.
func New(dir string) *Cache { return &Cache{dir: dir} }

// Default resolves the user's cache directory: $BRENNER_HOME when set, else
// <home>/.brenner, where <home> is resolved by UserHomeDir.
func Default() (*Cache, error) {
	if dir := os.Getenv("BRENNER_HOME"); dir != "" {
		return New(dir), nil
	}
	home, err := UserHomeDir()
	if err != nil {
		return nil, err
	}
	return New(filepath.Join(home, ".brenner")), nil
}

// UserHomeDir resolves the home directory Brenner should act for. When running
// under sudo, os.UserHomeDir() reports /root; this honors $SUDO_USER so the
// cache and home scan target the real user's home instead.
func UserHomeDir() (string, error) {
	if os.Geteuid() == 0 {
		if name := os.Getenv("SUDO_USER"); name != "" && name != "root" {
			if u, err := user.Lookup(name); err == nil && u.HomeDir != "" {
				return u.HomeDir, nil
			}
		}
	}
	return os.UserHomeDir()
}

// Dir returns the cache directory path.
func (c *Cache) Dir() string { return c.dir }

// Entry describes a cached ISO.
type Entry struct {
	Name string
	Path string
	Size uint64
}

// List returns the cached *.iso files sorted by name. A missing cache directory
// yields an empty list rather than an error.
func (c *Cache) List() ([]Entry, error) {
	dirEntries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var isos []Entry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), isoExt) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		isos = append(isos, Entry{
			Name: e.Name(),
			Path: filepath.Join(c.dir, e.Name()),
			Size: uint64(info.Size()),
		})
	}
	sort.Slice(isos, func(i, j int) bool { return isos[i].Name < isos[j].Name })
	return isos, nil
}

// Save copies srcPath into the cache (creating the directory if needed) and
// returns the destination path. If the source already lives in the cache the
// copy is skipped.
func (c *Cache) Save(srcPath string) (string, error) {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(c.dir, filepath.Base(srcPath))

	if srcAbs, err := filepath.Abs(srcPath); err == nil {
		if dstAbs, err := filepath.Abs(dst); err == nil && srcAbs == dstAbs {
			return dst, nil
		}
	}

	in, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	return dst, out.Close()
}
