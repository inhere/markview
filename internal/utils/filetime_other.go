//go:build !windows

package utils

import "time"

func FileCreatedTime(_ any) time.Time {
	return time.Time{}
}
