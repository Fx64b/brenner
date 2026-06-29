//go:build !linux && !darwin && !windows

package device

func platformEnumerator() Enumerator { return unsupportedEnumerator{} }

type unsupportedEnumerator struct{}

func (unsupportedEnumerator) ListRemovable() ([]Device, error) { return nil, ErrUnsupported }
func (unsupportedEnumerator) Unmount(string) error             { return ErrUnsupported }
