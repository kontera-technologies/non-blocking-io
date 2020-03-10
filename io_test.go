package non_blocking_io_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
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

func TestNewFD(t *testing.T) {
	var err error
	_, err = nbio.NewFD(999)
	if err == nil || !errors.Is(err, unix.EBADF) {
		t.Errorf(`Expected: "%s", received: "%s"`, unix.EBADF, err)
	}

	_, err = nbio.NewFD(os.Stdin.Fd())
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

	_, err = nbio.NewFD(os.Stdin.Fd())
	if err != nil {
		t.Fatal(err)
	}
}

func TestFD_Write_block(t *testing.T) {
	dir, err := ioutil.TempDir("", "fifo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	fifoPath := filepath.Join(dir, "fifo")
	err = unix.Mkfifo(fifoPath, 0600)
	if err != nil {
		t.Fatal(err)
	}
	c := make(chan bool)
	defer close(c)
	go func() {
		fdR, err := nbio.Open(fifoPath, unix.O_RDONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer fdR.Close()
		// Keep the file descriptor (fdR) open until the main function exists!
		<-c
	}()

	// Give the above go-routine a millisecond to open the fifo for reading.
	time.Sleep(time.Millisecond)
	w, err := nbio.Open(fifoPath, unix.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Write a huge chunk of data to fill the internal system buffer for the fifo file.
	_, err = w.Write(bytes.Repeat([]byte("foo\n"), 100000))
	if err != nil {
		t.Fatal(err)
	}

	n, err := w.Write([]byte("foo\n"))
	if n > 0 {
		t.Fatalf("No data expected, %d bytes read", n)
	}
	if !errors.Is(err, unix.EAGAIN) {
		t.Fatalf("unix.EAGAIN error expected, received: %v", err)
	}
}

func TestFD_Read(t *testing.T) {
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

	c := make(chan bool)
	defer close(c)
	go func() {
		fdW, err := unix.Open(fifoName, unix.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer unix.Close(fdW)
		f := os.NewFile(uintptr(fdW), "tmp-w")
		f.Write([]byte(expected))
		// Keep the file descriptor (fdW) open until the main function exists!
		<-c
	}()

	fd, err := unix.Open(fifoName, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)

	err = nbio.UnblockFd(uintptr(fd))
	if err != nil {
		t.Fatal(err)
	}

	r, err := nbio.NewFD(uintptr(fd))
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
		if !errors.Is(err, unix.EAGAIN) {
			t.Errorf("unix.EAGAIN error Expected, received: %v", err)
		}

		if n != 0 || string(buf[:n]) != "" {
			t.Errorf(`Invalid output, expected: "", received: %s`, string(buf[:n]))
		}
	})
}

func TestNewFifo(t *testing.T) {
	var err error
	rw, err := nbio.NewFifo()
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("testdata/foo.sh")
	cmd.Stdout = rw
	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Wait for a few milliseconds to give the process time to start.
	time.Sleep(time.Millisecond * 20)
	expected := "foo\n"

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, 100)
		n, err := rw.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if n != len([]byte(expected)) {
			t.Fatalf(`Invalid output length, expected: %d, received: %d`, len(expected), n)
		}

		if string(buf[:n]) != expected {
			t.Fatalf(`Invalid output, expected: "%s", received: "%s"`, expected, string(buf[:n]))
		}
	})

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, 100)
		n, err := rw.Read(buf)
		if n > 0 {
			t.Fatalf("No data expected, %d bytes read", n)
		}
		if !errors.Is(err, unix.EAGAIN) {
			t.Fatalf("unix.EAGAIN error expected, received: %v", err)
		}
	})

	time.Sleep(time.Millisecond * 100)
	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, 100)
		n, err := rw.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if n != len([]byte(expected)) {
			t.Fatalf(`Invalid output length, expected: %d, received: %d`, len(expected), n)
		}

		if string(buf[:n]) != expected {
			t.Fatalf(`Invalid output, expected: "%s", received: "%s"`, expected, string(buf[:n]))
		}
	})
}

// _TestFile_Read is used to demonstrate that TestFD_Read would have failed with the
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

	c := make(chan bool)
	defer close(c)
	go func() {
		fdW, err := unix.Open(fifoName, unix.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer unix.Close(fdW)
		f := os.NewFile(uintptr(fdW), "tmp-w")
		f.Write([]byte(expected))
		// Keep the file descriptor (fdW) open until the main function exists!
		<-c
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

// _Test_StdoutPipe is used to demonstrate that TestNewFifo would have failed with the
// regular Command.StdoutPipe function.
//noinspection GoUnusedFunction
func _Test_StdoutPipe(t *testing.T) {
	var err error

	cmd := exec.Command("testdata/foo.sh")
	r, err := cmd.StdoutPipe()
	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	// Wait for a few milliseconds to give the process time to start.
	time.Sleep(time.Millisecond * 10)
	expected := "foo\n"

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, 100)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}

		if n != len([]byte(expected)) {
			t.Fatalf(`Invalid output length, expected: %d, received: %d`, len(expected), n)
		}

		if string(buf[:n]) != expected {
			t.Fatalf(`Invalid output, expected: "%s", received: "%s"`, expected, string(buf[:n]))
		}
	})

	deadlineFn(t, time.Millisecond, func(c chan bool) {
		defer close(c)
		buf := make([]byte, 100)
		n, err := r.Read(buf)
		if n > 0 {
			t.Fatalf("No data expected, %d bytes read", n)
		}
		if !errors.Is(err, unix.EAGAIN) {
			t.Fatalf("unix.EAGAIN error expected, received: %v", err)
		}
	})
}
