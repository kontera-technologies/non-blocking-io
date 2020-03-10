package non_blocking_io_test

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	nbio "github.com/kontera-technologies/non-blocking-io"
)

func deadlineFn(t testing.TB, dur time.Duration, fn func(c chan bool)) {
	t.Helper()
	c := make(chan bool)
	go fn(c)

	select {
	case <-c:
	case <-time.After(dur):
		t.Fatalf("execution took longer than %s", dur)
	}
}

func TestNewReadWriter(t *testing.T) {
	var err error
	_, err = nbio.NewReadWriter(999)
	if err == nil || !errors.Is(err, unix.EBADF) {
		t.Errorf(`Expected: "%s", received: "%s"`, unix.EBADF, err)
	}

	_, err = nbio.NewReadWriter(os.Stdin.Fd())
	if err == nil || !errors.Is(err, nbio.ErrBlockingFd) {
		t.Errorf(`Expected: "%s", received: "%s"`, nbio.ErrBlockingFd, err)
	}
}

func TestUnblockFd(t *testing.T) {
	var err error
	err = nbio.UnblockFd(os.Stdin.Fd())
	if err != nil {
		t.Fatal(err)
	}

	_, err = nbio.NewReader(os.Stdin.Fd())
	if err != nil {
		t.Fatal(err)
	}
}

func TestReader_Read(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestFifoEOF")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	fifoName := filepath.Join(dir, "fifo")
	err = syscall.Mkfifo(fifoName, 0600)
	if err != nil {
		t.Fatal(err)
	}
	expected := "foo"

	go func() {
		fd, err := unix.Open(fifoName, unix.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		f := os.NewFile(uintptr(fd), "tmp")
		f.Write([]byte(expected))
		// Keep the file descriptor (fd) open!
	}()

	log.Printf("%03b", unix.O_RDONLY|unix.O_NONBLOCK)
	fd, err := unix.Open(fifoName, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)

	err = nbio.UnblockFd(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	r, err := nbio.NewReader(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	// Sometimes the process needs another microsecond to write the data on the tmp file.
	time.Sleep(time.Microsecond)

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, len([]byte(expected))*2)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if n != len([]byte(expected)) || string(buf[:n]) != expected {
			t.Errorf(`Invalid output, expected: "%s", received: %s`, expected, string(buf[:n]))
		}
	})

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, len([]byte(expected))*2)
		n, err := r.Read(buf)
		if err != nil {
			t.Error(err)
		}

		if n != 0 || string(buf[:n]) != "" {
			t.Errorf(`Invalid output, expected: "", received: %s`, string(buf[:n]))
		}
	})
}

// _TestFile_Read is used to demonstrate that TestFile_Read would have failed with the
// regular File.Read function.
//noinspection GoUnusedFunction
func _TestFile_Read(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestFifoEOF")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	fifoName := filepath.Join(dir, "fifo")
	err = syscall.Mkfifo(fifoName, 0600)
	if err != nil {
		t.Fatal(err)
	}
	expected := "foo"

	go func() {
		fdW, err := unix.Open(fifoName, unix.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		f := os.NewFile(uintptr(fdW), "tmp-w")
		f.Write([]byte(expected))
		// Keep the file descriptor (fd) open!
	}()

	fdR, err := unix.Open(fifoName, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fdR)

	r := os.NewFile(uintptr(fdR), "tmp-r")

	// Sometimes the process needs another microsecond to write the data on the tmp file.
	time.Sleep(time.Microsecond)

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, len([]byte(expected))*2)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if n != len([]byte(expected)) || string(buf[:n]) != expected {
			t.Errorf(`Invalid output, expected: "%s", received: %s`, expected, string(buf[:n]))
		}
	})

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, len([]byte(expected))*2)
		n, err := r.Read(buf)
		if err != nil {
			t.Log(err)
		}

		if n != 0 || string(buf[:n]) != "" {
			t.Errorf(`Invalid output, expected: "", received: %s`, string(buf[:n]))
		}
	})
}
