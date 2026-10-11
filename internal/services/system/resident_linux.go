//go:build linux

package system

import (
	"os"
	"strconv"
	"strings"
)

func resident() *uint64 {
	b, e := os.ReadFile("/proc/self/statm")
	if e != nil {
		return nil
	}
	f := strings.Fields(string(b))
	if len(f) < 2 {
		return nil
	}
	pages, e := strconv.ParseUint(f[1], 10, 64)
	if e != nil {
		return nil
	}
	v := pages * uint64(os.Getpagesize())
	return &v
}
