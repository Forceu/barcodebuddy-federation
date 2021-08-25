// +build linux,amd64 linux,arm64

package helper

import (
	"syscall"
)

func GetRamInfo() (string, string, error) {
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		return "", "", err
	}
	totalRam := info.Totalram
	freeRam := info.Freeram
	return ByteCountSI(totalRam), ByteCountSI(freeRam), nil
}
