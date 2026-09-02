//go:build !windows

package main

// availableDrives is a no-op off Windows — the real implementation
// (drives_windows.go) lists every currently accessible drive letter so
// the file/folder picker can jump between them.
func availableDrives() []string { return nil }
