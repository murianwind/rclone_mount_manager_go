//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// redirectStderrToFile repoints the OS-level stderr handle (not just Go's
// os.Stderr variable) at path. This is what actually matters for crash
// diagnostics: when a goroutine panics without a recover(), the Go
// runtime prints the panic message and full stack trace directly to file
// descriptor 2 via a low-level write — it does not go through the os
// package, so merely reassigning os.Stderr in Go code has no effect on
// it. Since this app is built with -H=windowsgui (no console), that
// output currently has nowhere to go and is simply lost. Redirecting the
// underlying Windows handle with SetStdHandle is what lets it land in a
// real file instead — the difference between "the app just vanished"
// and "여기 crash.log에 정확히 왜 죽었는지 남아있다."
//
// Errors are deliberately ignored: if this fails for any reason, the app
// should still start normally — crash logging is a nice-to-have, not
// something that should ever block startup.
func redirectStderrToFile(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	os.Stderr = f
}
