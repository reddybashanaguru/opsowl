// Package scanner wraps ClamAV clamd for antivirus scanning of uploaded files.
// It communicates via the clamd INSTREAM protocol over TCP — no external
// Go library required.  The interface allows tests to inject a no-op stub.
package scanner

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// ErrVirusFound is returned when clamd reports a known threat.
var ErrVirusFound = errors.New("antivirus: threat detected")

// ErrUnavailable is returned when clamd cannot be reached.
// Callers may choose to allow uploads (log + continue) or block.
var ErrUnavailable = errors.New("antivirus: scanner unavailable")

// Scanner defines the scanning interface so implementations can be swapped in tests.
type Scanner interface {
	// ScanFile opens the file at path and streams it to clamd.
	// Returns ErrVirusFound when a threat is detected, ErrUnavailable
	// when clamd is unreachable, or nil when clean.
	ScanFile(path string) error

	// ScanReader streams r to clamd.
	ScanReader(r io.Reader) error
}

// ClamdScanner connects to a running clamd daemon over TCP.
type ClamdScanner struct {
	addr    string        // e.g. "localhost:3310"
	timeout time.Duration // per-connection timeout
}

// New returns a ClamdScanner pointed at addr (default "localhost:3310").
func New(addr string) *ClamdScanner {
	if addr == "" {
		addr = "localhost:3310"
	}
	return &ClamdScanner{addr: addr, timeout: 30 * time.Second}
}

// ScanFile opens path and calls ScanReader.
func (c *ClamdScanner) ScanFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("scanner: open file: %w", err)
	}
	defer f.Close()
	return c.ScanReader(f)
}

// ScanReader streams r to clamd using the INSTREAM protocol.
//
// Protocol (from clamd(8)):
//  1. Send "zINSTREAM\0"
//  2. For each chunk: 4-byte big-endian length + chunk bytes
//  3. Send a 4-byte zero to signal EOF
//  4. Read response: "stream: OK\n" or "stream: <name> FOUND\n"
func (c *ClamdScanner) ScanReader(r io.Reader) error {
	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return ErrUnavailable
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))

	// Step 1: command
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return ErrUnavailable
	}

	// Step 2: stream chunks
	buf := make([]byte, 4096)
	lenBuf := make([]byte, 4)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			binary.BigEndian.PutUint32(lenBuf, uint32(n))
			if _, err := conn.Write(lenBuf); err != nil {
				return ErrUnavailable
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return ErrUnavailable
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("scanner: read: %w", readErr)
		}
	}

	// Step 3: zero-length chunk = EOF
	binary.BigEndian.PutUint32(lenBuf, 0)
	if _, err := conn.Write(lenBuf); err != nil {
		return ErrUnavailable
	}

	// Step 4: read response
	resp, err := io.ReadAll(io.LimitReader(conn, 256))
	if err != nil {
		return ErrUnavailable
	}

	result := strings.TrimSpace(string(resp))
	switch {
	case strings.HasSuffix(result, "OK"):
		return nil
	case strings.Contains(result, "FOUND"):
		// e.g. "stream: Eicar-Signature FOUND"
		parts := strings.SplitN(result, ":", 2)
		threat := strings.TrimSuffix(strings.TrimSpace(parts[len(parts)-1]), " FOUND")
		return fmt.Errorf("%w: %s", ErrVirusFound, threat)
	default:
		return fmt.Errorf("scanner: unexpected clamd response: %q", result)
	}
}

// NoopScanner is a pass-through used when CLAMAV_ADDR is empty (dev / CI).
// ScanReader must drain r to prevent io.Pipe deadlocks when used with TeeReader.
type NoopScanner struct{}

func (NoopScanner) ScanFile(_ string) error { return nil }
func (NoopScanner) ScanReader(r io.Reader) error {
	_, _ = io.Copy(io.Discard, r)
	return nil
}

// FromEnv returns a ClamdScanner if CLAMAV_ADDR is set, otherwise a NoopScanner.
// This lets production deployments opt in to scanning without breaking dev/CI.
func FromEnv() Scanner {
	addr := os.Getenv("CLAMAV_ADDR")
	if addr == "" {
		return NoopScanner{}
	}
	return New(addr)
}
