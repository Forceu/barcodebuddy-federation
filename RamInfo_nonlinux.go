// +build !linux !amd64,!arm64

package main

import "errors"

func getRamInfo() (string, string, error) {
	return "", "", errors.New("platform: Not supported on this platform")
}
