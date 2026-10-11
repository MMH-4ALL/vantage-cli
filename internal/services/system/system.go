// Package system reports measured runtime information; unsupported values are omitted.
package system

import (
	"os"
	"runtime"
	"time"
)

type Info struct {
	OS                   string  `json:"os"`
	Architecture         string  `json:"architecture"`
	CPUs                 int     `json:"logical_cpus"`
	Hostname             string  `json:"hostname,omitempty"`
	HeapBytes            uint64  `json:"go_heap_bytes"`
	RuntimeReservedBytes uint64  `json:"go_runtime_reserved_bytes"`
	ResidentBytes        *uint64 `json:"resident_bytes,omitempty"`
	UptimeSeconds        float64 `json:"uptime_seconds"`
	GoVersion            string  `json:"go_version"`
	Directory            string  `json:"directory"`
}

func Collect(start time.Time) Info {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	host, _ := os.Hostname()
	dir, _ := os.Getwd()
	return Info{runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), host, m.Alloc, m.Sys, resident(), time.Since(start).Seconds(), runtime.Version(), dir}
}
