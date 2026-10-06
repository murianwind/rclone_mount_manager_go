package main

import "fyne.io/fyne/v2/dialog"

// showGated shows the dialog build() creates, unless one for the same key
// is already open. Must be called on the Fyne UI thread (wrap in
// fyne.Do). The key is released when the dialog closes, however it was
// dismissed.
func (rm *rcloneManager) showGated(key string, build func() dialog.Dialog) {
	if !rm.dialogGate.tryAcquire(key) {
		return
	}
	rm.revealWindow()
	d := build()
	d.SetOnClosed(func() { rm.dialogGate.release(key) })
	d.Show()
}

func (rm *rcloneManager) showGatedInfo(key, title, message string) {
	rm.showGated(key, func() dialog.Dialog {
		return dialog.NewInformation(title, message, rm.win)
	})
}

func (rm *rcloneManager) showGatedError(key string, err error) {
	rm.showGated(key, func() dialog.Dialog {
		return dialog.NewError(err, rm.win)
	})
}
