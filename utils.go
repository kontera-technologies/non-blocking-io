package non_blocking_io

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

var ErrBlockingFd = errors.New("file descriptor is set to blocking")

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

func bitwise2bool(b int) bool {
	return b > 0
}