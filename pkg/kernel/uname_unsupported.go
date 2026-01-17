//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly && !aix

package kernel

import (
	"errors"
)

// utsName represents the system name structure. It is defined here to make it
// portable as it is available on Linux but not on Windows.
type utsName struct {
	Release [65]byte
}

func uname() (*utsName, error) {
	return nil, errors.New("kernel version detection is not available on this platform")
}
