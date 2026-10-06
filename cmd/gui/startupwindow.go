package main

import "fyne.io/fyne/v2"

// windowStartMode is how the main window is brought up at launch.
type windowStartMode int

const (
	startWindowShown  windowStartMode = iota // normal start: show the window
	startWindowHidden                        // tray-only start: window exists but stays hidden
)

func windowStartModeFor(startMinimized bool) windowStartMode {
	if startMinimized {
		return startWindowHidden
	}
	return startWindowShown
}

// prepareNativeWindow makes sure the window has a real native (Win32)
// handle even when it must stay hidden. It must run on Fyne's main
// thread, from the app-started hook.
//
// Why: on Windows, Fyne watches the system theme (dark/light) and, on any
// change, applies it to EVERY window via setDarkMode, which asks the
// window for its native handle (GetWin32Window). A window that has never
// been shown has no handle yet — and Hide() on such a window does nothing
// — so a tray-only start left the app one theme-change event away from a
// nil-pointer panic. That event fires at login (while Windows applies the
// theme) and again whenever the user or the OS switches dark/light mode,
// so it could kill the manager minutes or days after launch while the
// rclone processes it started kept the drives mounted. Fyne's queued
// theme handler only runs after the app-started hook returns, so creating
// the handle here (Show immediately followed by Hide, with no event-loop
// turn in between) closes that window.
//
// Fyne sources (v2.8.0): app/app_windows.go watchTheme,
// internal/driver/glfw/window_windows.go setDarkMode, window.go Show/Hide,
// loop.go runGL.
func prepareNativeWindow(win fyne.Window, mode windowStartMode) {
	if mode != startWindowHidden {
		return
	}
	win.Show()
	win.Hide()
}
