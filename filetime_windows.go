//go:build windows

package main

import (
	"os"
	"syscall"
	"time"
)

func fileCreatedTime(info os.FileInfo) time.Time {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return time.Time{}
	}

	return time.Unix(0, data.CreationTime.Nanoseconds())
}
