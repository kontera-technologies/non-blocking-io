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

type FD struct {
	fd uintptr
}

func (fd FD) Close() error {
	return unix.Close(int(fd.fd))
}

func (fd FD) Read(p []byte) (int, error) {
	n, err := unix.Read(int(fd.fd), p)
	if err != nil && err.(unix.Errno).Timeout() && n < 0 {
		n = 0
	}

	return n, err
}

func (fd FD) Write(p []byte) (int, error) {
	n, err := unix.Write(int(fd.fd), p)
	if err != nil && err.(unix.Errno).Timeout() && n < 0 {
		n = 0
	}

	return n, err
}

func NewFD(fd uintptr) (*FD, error) {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get fd flags - %w", err)
	}
	if err := validateNonBlock(flags); err != nil {
		return nil, err
	}

	return &FD{fd: fd}, nil
}

func validateNonBlock(flags int) error {
	if flags&unix.O_NONBLOCK != unix.O_NONBLOCK && flags&unix.O_WRONLY != unix.O_WRONLY {
		return ErrBlockingFd
	}
	return nil
}

func Open(path string, mode int, perm uint32) (*FD, error) {
	fd, err := unix.Open(path, mode|unix.O_NONBLOCK, perm)
	if err != nil {
		return nil, fmt.Errorf("non_blocking_io.Open(%s,%03b,%d)- %w", path, mode|unix.O_NONBLOCK, perm, err)
	}

	rwc, err := NewFD(uintptr(fd))
	if err != nil {
		return nil, err
	}

	return rwc, nil
}

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

func NewFifo() (*FD, error) {
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

	return &FD{fd: uintptr(fd)}, nil
}
