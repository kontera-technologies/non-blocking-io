package non_blocking_io_test

import (
	"errors"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	nbio "github.com/kontera-technologies/non-blocking-io"
)

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
	time.Sleep(time.Millisecond * 10)
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
