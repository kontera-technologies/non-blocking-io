package non_blocking_io

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

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

	fd, err := syscall.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	return &FD{fd: uintptr(fd)}, nil
}
