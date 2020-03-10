// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package non_blocking_io

import (
	"errors"
	"fmt"
	"io"
	"syscall"

	"golang.org/x/sys/unix"
)

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

func NewReader(fd uintptr) (io.Reader, error) {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get fd flags - %w", err)
	}
	if err := validateNonBlock(flags); err != nil {
		return nil, err
	}
	if err := validateReadable(flags); err != nil {
		return nil, err
	}
	return &FD{fd: fd}, nil
}

func NewWriter(fd uintptr) (io.Writer, error) {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get fd flags - %w", err)
	}
	if err := validateNonBlock(flags); err != nil {
		return nil, err
	}
	if err := validateWritable(flags); err != nil {
		return nil, err
	}
	return &FD{fd: fd}, nil
}

func NewReadWriter(fd uintptr) (io.ReadWriter, error) {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get fd flags - %w", err)
	}
	if err := validateNonBlock(flags); err != nil {
		return nil, err
	}
	if err := validateReadable(flags); err != nil {
		return nil, err
	}
	if err := validateWritable(flags); err != nil {
		return nil, err
	}

	return &FD{fd: fd}, nil
}

func NewReadWriteCloser(fd uintptr) (io.ReadWriteCloser, error) {
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot get fd flags - %w", err)
	}
	if err := validateNonBlock(flags); err != nil {
		return nil, err
	}
	if err := validateReadable(flags); err != nil {
		return nil, err
	}
	if err := validateWritable(flags); err != nil {
		return nil, err
	}
	if err := validateClosable(flags); err != nil {
		return nil, err
	}

	return &FD{fd: fd}, nil
}

func validateNonBlock(flags int) error {
	if flags & unix.O_NONBLOCK != unix.O_NONBLOCK {
		return ErrBlockingFd
	}
	return nil
}

func validateReadable(flags int) error {
	if flags & unix.O_RDWR != unix.O_RDWR && flags & unix.O_RDONLY != unix.O_RDONLY {
		return errors.New("file descriptor is not readable")
	}
	return nil
}

func validateWritable(flags int) error {
	if flags & unix.O_RDWR != unix.O_RDWR && flags & unix.O_WRONLY != unix.O_WRONLY {
		return errors.New("file descriptor is not writable")
	}
	return nil
}

func validateClosable(flags int) error {
	if flags & unix.O_CLOEXEC != unix.O_CLOEXEC {
		return errors.New("file descriptor is not closable")
	}
	return nil
}

func Open(path string, mode int, perm uint32) (io.ReadWriteCloser, error) {
	fd, err := syscall.Open(path, mode|unix.O_RDWR|unix.O_NONBLOCK, perm)
	if err != nil {
		return nil, err
	}

	rwc, err := NewReadWriteCloser(uintptr(fd))
	if err != nil {
		return nil, err
	}

	return rwc, nil
}