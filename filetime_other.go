//go:build !windows

package main

import "time"

func fileCreatedTime(_ any) time.Time {
	return time.Time{}
}
