//go:build windows

package engine

import (
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	kernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")

	// consoleAttachMu serializes the AttachConsole → signal → FreeConsole
	// critical section below. AttachConsole/FreeConsole are process-wide
	// Windows state (there's only one "current console" per process, not
	// per goroutine) — without this, stopping several mounts at once
	// (unmountAllAndWait spawns one goroutine per mount) let two calls
	// race: one goroutine's AttachConsole(pidA) could be stolen by
	// another's concurrent AttachConsole(pidB), so a graceful stop that
	// should have succeeded failed with "The handle is invalid." and fell
	// back to a hard Kill().
	consoleAttachMu sync.Mutex
)

// ConfigureBackgroundProcess hides the console window while giving the child
// its own console and process group. GenerateConsoleCtrlEvent can only deliver
// CTRL_BREAK to a process group that shares the caller's console; a -H=windowsgui
// parent has no console of its own, so the child needs a private hidden one.
// Must be called before cmd.Start().
func ConfigureBackgroundProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NEW_CONSOLE,
	}
}

// attachConsole attaches the current process to the console owned by pid.
// x/sys/windows v0.30.0 does not expose AttachConsole/FreeConsole as package
// functions, so call the stable kernel32 exports directly.
func attachConsole(pid uint32) error {
	r1, _, e1 := procAttachConsole.Call(uintptr(pid))
	if r1 == 0 {
		return e1
	}
	return nil
}

func freeConsole() {
	_, _, _ = procFreeConsole.Call()
}

// SignalGracefulStop asks a rclone process to shut down cleanly by sending
// CTRL_BREAK_EVENT. A GUI parent normally has no console, so the first direct
// attempt is expected to fail. In that case temporarily attach to the child's
// hidden console, send the signal, then detach again. The attach/detach
// dance is serialized (see consoleAttachMu) since it's process-wide state —
// stopping several mounts concurrently must not let one steal another's
// console attachment mid-signal.
func SignalGracefulStop(pid int) error {
	if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid)); err == nil {
		return nil
	}

	consoleAttachMu.Lock()
	defer consoleAttachMu.Unlock()

	if attachErr := attachConsole(uint32(pid)); attachErr != nil {
		return attachErr
	}
	defer freeConsole()

	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(pid))
}

// KillOrphanedRclone force-kills every currently running rclone.exe
// process. Meant to be called once at app startup, before the first
// auto-mount attempt: if a previous session vanished (crashed, got
// killed by Windows, etc.) without cleanly unmounting, its rclone.exe can
// keep running on its own — Windows doesn't tie a child process's
// lifetime to its parent's — and permanently blocks this session from
// remounting the same drive letter no matter how many times we retry
// waiting for it to go away on its own (see staleretry.go, which only
// helps with genuinely transient WinFsp cleanup timing, not an actually
// still-running conflicting process).
//
// Exit code 128 means taskkill found nothing to kill — the common,
// expected case on a normal startup — and is not treated as an error.
func KillOrphanedRclone() error {
	cmd := exec.Command("taskkill", "/F", "/IM", "rclone.exe", "/T")
	ConfigureBackgroundProcess(cmd)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
		return nil
	}
	return err
}
