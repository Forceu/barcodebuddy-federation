// +build !linux !amd64,!arm64

package helper

import "errors"

func GetRamInfo() (string, string, error) {
	return "", "", errors.New("platform: Not supported on this platform")
}
