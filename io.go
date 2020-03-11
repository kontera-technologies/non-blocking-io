// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package non_blocking_io

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var ErrBlockingFd = errors.New("file descriptor is set to blocking")

type Fd struct {
	fd uintptr
}

// Close closes the file descriptor.
func (fd Fd) Close() error {
	return unix.Close(int(fd.fd))
}

// Read is an implementation of ``io.Reader`` that will return an error if the file descriptor is not ready for reading.
func (fd Fd) Read(p []byte) (int, error) {
	n, err := unix.Read(int(fd.fd), p)
	if err != nil && err.(unix.Errno).Timeout() && n < 0 {
		n = 0
	}

	return n, err
}

// Write is an implementation of ``io.Writer`` that will return an error if the file descriptor is not ready for writing.
func (fd Fd) Write(p []byte) (int, error) {
	n, err := unix.Write(int(fd.fd), p)
	if err != nil && err.(unix.Errno).Timeout() && n < 0 {
		n = 0
	}

	return n, err
}

// SelectRead is the same as Read, but blocks for ``timeout`` until the file descriptor is ready for reading.
func (fd Fd) SelectRead(p []byte, timeout unix.Timeval) (int, error) {
	fdSet := unix.FdSet{}
	fdSet.Set(int(fd.fd))
	_, err := unix.Select(1, &fdSet, &unix.FdSet{}, &unix.FdSet{}, &timeout)
	if err != nil {
		return 0, err
	}

	return fd.Read(p)
}

// SelectWrite is the same as Write, but blocks for ``timeout`` until the file descriptor is ready for writing.
func (fd Fd) SelectWrite(p []byte, timeout unix.Timeval) (int, error) {
	fdSet := unix.FdSet{}
	fdSet.Set(int(fd.fd))
	_, err := unix.Select(1, &unix.FdSet{}, &fdSet, &unix.FdSet{}, &timeout)
	if err != nil {
		return 0, err
	}

	return fd.Write(p)
}

// NewFd creates a new file struct.
// Will return an error if the file descriptor is not valid, or if it is blocking.
// Use ``UnblockFd`` to transform a blocking file descriptor into non-blocking.
func NewFd(fd uintptr) (*Fd, error) {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get fd flags - %w", err)
	}
	if err := validateNonBlock(flags); err != nil {
		return nil, err
	}

	return &Fd{fd: fd}, nil
}

func validateNonBlock(flags int) error {
	if flags&unix.O_NONBLOCK != unix.O_NONBLOCK && flags&unix.O_WRONLY != unix.O_WRONLY {
		return ErrBlockingFd
	}
	return nil
}

// Open opens the file descriptor of a path as non-blocking and returns a new ``Fd`` struct from it.
func Open(path string, mode int, perm uint32) (*Fd, error) {
	fd, err := unix.Open(path, mode|unix.O_NONBLOCK, perm)
	if err != nil {
		return nil, err
	}

	rwc, err := NewFd(uintptr(fd))
	if err != nil {
		return nil, err
	}

	return rwc, nil
}

// UnblockFd transforms a blocking file descriptor to non-blocking.
func UnblockFd(fd uintptr) error {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return fmt.Errorf("cannot get fd flags - %w", err)
	}

	_, err = unix.FcntlInt(fd, unix.F_SETFL, flags|unix.O_NONBLOCK)
	if err != nil {
		return fmt.Errorf("cannot set fd flags - %w", err)
	}

	return nil
}

// NewFifo uses the ``mkfifo`` unix command to creates a pipe-like file descriptor.
// Use it to pipe the stdin / stdout / stderr of a process to a non-blocking ``Fd`` struct.
func NewFifo() (*Fd, error) {
	dir, err := ioutil.TempDir("", "fifo")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "fifo")
	err = unix.Mkfifo(path, 0600)
	if err != nil {
		return nil, err
	}

	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	return &Fd{fd: uintptr(fd)}, nil
}
