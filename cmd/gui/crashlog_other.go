//go:build !windows

package main

// redirectStderrToFile is a no-op off Windows. The real implementation
// (crashlog_windows.go) repoints the OS-level stderr handle so an
// unrecovered panic's stack trace lands in a file instead of vanishing —
// only meaningful for a -H=windowsgui build with no console.
func redirectStderrToFile(path string) {}
