// Package flash performs the actual byte-pushing: streaming an image onto a
// device, zero-filling a device, and verifying a write by hash. Everything here
// works on plain io.Reader/io.Writer so it can target a real block device or an
// ordinary file (handy for tests and for producing .img files).
package flash

import (
	"bytes"
	"crypto/sha256"
	"io"
)

// DefaultBlockSize matches typical USB write granularity and keeps progress
// updates smooth without drowning the kernel in syscalls.
const DefaultBlockSize = 4 << 20 // 4 MiB

// Progress reports how far a Copy or Wipe has advanced.
type Progress struct {
	Written uint64
	Total   uint64
}

// Fraction returns completion in the range [0, 1]; 0 when the total is unknown.
func (p Progress) Fraction() float64 {
	if p.Total == 0 {
		return 0
	}
	return float64(p.Written) / float64(p.Total)
}

// Copy streams src into dst in blockSize chunks, invoking progress (if non-nil)
// after every chunk. total is passed through to the callback for ETA/percent
// math. It returns the number of bytes written.
func Copy(dst io.Writer, src io.Reader, total uint64, blockSize int, progress func(Progress)) (uint64, error) {
	if blockSize <= 0 {
		blockSize = DefaultBlockSize
	}
	buf := make([]byte, blockSize)
	var written uint64
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			w, writeErr := dst.Write(buf[:n])
			written += uint64(w)
			if writeErr != nil {
				return written, writeErr
			}
			if w < n {
				return written, io.ErrShortWrite
			}
			if progress != nil {
				progress(Progress{Written: written, Total: total})
			}
		}
		if readErr == io.EOF {
			return written, nil
		}
		if readErr != nil {
			return written, readErr
		}
	}
}

// Wipe writes size zero bytes to dst, reporting progress along the way.
func Wipe(dst io.Writer, size uint64, blockSize int, progress func(Progress)) error {
	if blockSize <= 0 {
		blockSize = DefaultBlockSize
	}
	zeros := make([]byte, blockSize)
	var written uint64
	for written < size {
		chunk := uint64(blockSize)
		if remaining := size - written; remaining < chunk {
			chunk = remaining
		}
		w, err := dst.Write(zeros[:chunk])
		written += uint64(w)
		if err != nil {
			return err
		}
		if uint64(w) < chunk {
			return io.ErrShortWrite
		}
		if progress != nil {
			progress(Progress{Written: written, Total: size})
		}
	}
	return nil
}

// Overwrite streams size bytes into dst in blockSize chunks, filling each chunk
// with fill (e.g. crypto/rand.Read for a random "secure" wipe) and reporting
// progress. The device write is the bottleneck, so fill can be a normal CSPRNG.
func Overwrite(dst io.Writer, size uint64, blockSize int, fill func([]byte) error, progress func(Progress)) error {
	if blockSize <= 0 {
		blockSize = DefaultBlockSize
	}
	buf := make([]byte, blockSize)
	var written uint64
	for written < size {
		chunk := uint64(blockSize)
		if remaining := size - written; remaining < chunk {
			chunk = remaining
		}
		b := buf[:chunk]
		if err := fill(b); err != nil {
			return err
		}
		n, err := dst.Write(b)
		written += uint64(n)
		if err != nil {
			return err
		}
		if uint64(n) < chunk {
			return io.ErrShortWrite
		}
		if progress != nil {
			progress(Progress{Written: written, Total: size})
		}
	}
	return nil
}

// Verify hashes the full source and the first size bytes read back from the
// device, reporting whether they match. size should be the byte count returned
// by Copy. progress (if non-nil) is reported as the device — the slow part on a
// USB stick — is read back.
func Verify(device io.Reader, source io.Reader, size uint64, progress func(Progress)) (bool, error) {
	sourceSum := sha256.New()
	if _, err := io.Copy(sourceSum, source); err != nil {
		return false, err
	}
	deviceSum := sha256.New()
	read, err := Copy(deviceSum, io.LimitReader(device, int64(size)), size, DefaultBlockSize, progress)
	if err != nil {
		return false, err
	}
	if read != size {
		return false, io.ErrUnexpectedEOF
	}
	return bytes.Equal(sourceSum.Sum(nil), deviceSum.Sum(nil)), nil
}
