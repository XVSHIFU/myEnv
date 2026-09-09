//go:build !linux || !runtrace

// Package runtrace contains opt-in diagnostic instrumentation, absent from release builds.
package runtrace

func Mark(string) {}
func Flush()      {}
