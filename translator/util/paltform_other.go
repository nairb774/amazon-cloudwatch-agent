// +build !windows

package util

import (
	"errors"
	"fmt"
	"runtime"
)

func GetOSMajorVersion() (int, error) {
	return 0, errors.New(fmt.Sprintf("Unsupported operation on %s", runtime.GOOS))
}
