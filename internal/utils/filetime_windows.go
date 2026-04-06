//go:build windows

package utils

import (
	"os"
	"syscall"
	"time"
)

func FileCreatedTime(info os.FileInfo) time.Time {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return time.Time{}
	}

	return time.Unix(0, data.CreationTime.Nanoseconds())
}
