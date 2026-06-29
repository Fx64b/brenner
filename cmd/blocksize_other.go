//go:build !linux

package cmd

// blockDeviceSize has no portable implementation outside Linux; callers fall
// back to the device enumerator for sizing.
func blockDeviceSize(string) (uint64, bool) { return 0, false }
