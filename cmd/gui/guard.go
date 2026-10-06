package main

import (
	"runtime/debug"
	"strings"
)

// maxPanicStackLines keeps one recovered panic from flooding the
// 1000-line rotating log.
const maxPanicStackLines = 14

// guard runs fn and, if it panics, logs where/why (with a short stack)
// instead of letting the panic take the whole process down. It wraps the
// long-lived background loops (network monitor, schedule monitor, mount
// exit handling): a panic in any goroutine kills the entire app, but the
// rclone.exe children it started keep running — leaving a mounted drive
// with no manager. Logging the stack (not silently swallowing it) keeps
// the underlying bug visible.
func (rm *rcloneManager) guard(where string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			rm.logf("ERROR", "[패닉 복구] %s: %v\n%s", where, r, firstLines(string(debug.Stack()), maxPanicStackLines))
		}
	}()
	fn()
}

// firstLines returns at most the first n lines of s.
func firstLines(s string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}
