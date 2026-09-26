//go:build !windows

package engine

import (
	"errors"
	"os/exec"
)

// ConfigureBackgroundProcess is a no-op on non-Windows platforms — hiding
// a console window and Windows process groups are Windows-only concerns.
func ConfigureBackgroundProcess(cmd *exec.Cmd) {}

// ErrGracefulStopUnsupported is returned by SignalGracefulStop where no
// CTRL_BREAK-equivalent mechanism exists; callers should fall back to
// Process.Kill().
var ErrGracefulStopUnsupported = errors.New("graceful stop signal not supported on this platform")

func SignalGracefulStop(pid int) error {
	return ErrGracefulStopUnsupported
}

// KillOrphanedRclone is a no-op off Windows — the real implementation
// (process_windows.go) force-kills any leftover rclone.exe from a
// previous session before this one starts mounting.
func KillOrphanedRclone() error { return nil }
