//go:build !linux

package system

func resident() *uint64 { return nil }
