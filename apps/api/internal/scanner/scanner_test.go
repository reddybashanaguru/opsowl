package scanner_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/reddybashanaguru/opsowl/internal/scanner"
)

// fakeClamd spins up a TCP server that behaves like clamd INSTREAM.
// response is the raw string sent back after receiving EOF chunk.
func fakeClamd(t *testing.T, response string) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// drain instream command
		cmd := make([]byte, 10)
		if _, err := io.ReadFull(conn, cmd); err != nil {
			return
		}

		// drain chunks until zero-length terminator
		lenBuf := make([]byte, 4)
		for {
			if _, err := io.ReadFull(conn, lenBuf); err != nil {
				return
			}
			n := binary.BigEndian.Uint32(lenBuf)
			if n == 0 {
				break
			}
			if _, err := io.ReadFull(conn, make([]byte, n)); err != nil {
				return
			}
		}
		_, _ = fmt.Fprint(conn, response)
	}()
	return ln.Addr().String(), func() {
		ln.Close()
		<-done
	}
}

func TestScanReader_CleanFile(t *testing.T) {
	addr, stop := fakeClamd(t, "stream: OK\n")
	defer stop()

	sc := scanner.New(addr)
	if err := sc.ScanReader(strings.NewReader("hello world")); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestScanReader_VirusFound(t *testing.T) {
	addr, stop := fakeClamd(t, "stream: Eicar-Signature FOUND\n")
	defer stop()

	sc := scanner.New(addr)
	err := sc.ScanReader(strings.NewReader("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"))
	if err == nil {
		t.Fatal("expected ErrVirusFound, got nil")
	}
	if !strings.Contains(err.Error(), "threat detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanReader_ClamdUnreachable(t *testing.T) {
	// Port 1 is never open; dial should fail immediately.
	sc := scanner.New("127.0.0.1:1")
	err := sc.ScanReader(strings.NewReader("data"))
	if err != scanner.ErrUnavailable {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestNoopScanner_DrainsPipe(t *testing.T) {
	// Verify NoopScanner drains the reader so no pipe deadlock occurs.
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- scanner.NoopScanner{}.ScanReader(pr)
	}()

	payload := bytes.Repeat([]byte("a"), 1<<20) // 1 MB
	if _, err := pw.Write(payload); err != nil {
		t.Fatal(err)
	}
	pw.Close()

	if err := <-done; err != nil {
		t.Fatalf("NoopScanner returned error: %v", err)
	}
}

func TestNoopScanner_ScanFile(t *testing.T) {
	// ScanFile on a real file should return nil.
	if err := (scanner.NoopScanner{}).ScanFile("/dev/null"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
