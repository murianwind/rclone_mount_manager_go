package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// runningMount tracks a live rclone mount process. done is closed by the
// single goroutine that owns cmd.Wait() — unmount() waits on it (with a
// timeout) instead of calling Wait() itself, since exec.Cmd.Wait() may
// only be called once. stderr captures rclone's error output so a failed
// mount can show *why* it failed instead of just going quietly back to
// "해제됨". stoppedByUs distinguishes a failure from a normal unmount
// (both make the process exit, often with a non-zero code).
type runningMount struct {
	cmd           *exec.Cmd
	done          chan struct{}
	stderr        *bytes.Buffer
	stoppedByUs   bool
	autoTriggered bool // started by autoMountAll(), not a direct user action
}

// rcloneManager is the single owner of all app state — config, running
// mount processes, and the widgets that need to be refreshed when either
// changes. One instance is created in main() and threaded through every
// UI callback and background goroutine.
type rcloneManager struct {
	appDir string
	log    engine.RotatingLog
	store  engine.Store
	win    fyne.Window

	// cfg holds mounts.json's contents. Guarded by cfgMu because it's read
	// from goroutines that never go through fyne.Do: the network monitor
	// (autoMountAll, every ~10s while connected) and the rclone-update
	// check (activeMountsSnapshot) both read cfg.Mounts directly on their
	// own goroutines, while UI actions (add/edit/delete/reorder a mount,
	// import remotes, toggle a checkbox) mutate the same slices on the
	// Fyne main thread. Without a lock, an add/delete racing a network
	// poll is a real, ordinary Go data race (one goroutine ranging over a
	// slice while another reassigns it).
	//
	// Use cfgSnapshot() for any read spanning more than a single field,
	// and withCfg() for any mutation — both take the lock for exactly the
	// duration of the copy/edit, never longer.
	cfgMu sync.RWMutex
	cfg   engine.Config

	table         *widget.Table
	rcPathEntry   *widget.Entry
	rcVersionText *widget.Button // clickable — tapping checks for a newer rclone

	selectedRow int // -1 = nothing selected; used by the 위/아래 이동 buttons

	latestRcloneVersion string // cached from the last successful check; "" = unknown

	activeMu sync.Mutex
	active   map[string]*runningMount

	// offlineMu guards offlineSince — when connectivity was last observed
	// down (zero value = currently believed connected). Used to suppress
	// mount-failure dialogs for auto-triggered attempts during a short,
	// expected outage; see shouldSuppressAutoMountFailure.
	offlineMu    sync.Mutex
	offlineSince time.Time

	// appUpdateAvailable/rcloneUpdateAvailable drive the tray tooltip's
	// status text (see updateTrayTooltip). Only ever touched from the
	// Fyne UI thread (inside fyne.Do), so no separate mutex.
	appUpdateAvailable    bool
	rcloneUpdateAvailable bool

	// updatingRclone is set for the duration of installOrUpdateRclone —
	// see autoMountAll's doc comment for why this needs to be atomic
	// (read from the network-monitor goroutine, written from the update
	// goroutine).
	updatingRclone atomic.Bool
}

func (rm *rcloneManager) isUpdatingRclone() bool { return rm.updatingRclone.Load() }

// cfgSnapshot returns a copy of rm.cfg safe to read from any goroutine —
// the Mounts/Remotes slices are cloned, not shared, so the caller can
// range over the result without holding any lock and without racing a
// concurrent mutation.
func (rm *rcloneManager) cfgSnapshot() engine.Config {
	rm.cfgMu.RLock()
	defer rm.cfgMu.RUnlock()
	cfg := rm.cfg
	cfg.Mounts = append([]engine.Mount(nil), rm.cfg.Mounts...)
	cfg.Remotes = append([]engine.Remote(nil), rm.cfg.Remotes...)
	return cfg
}

// withCfg runs fn with exclusive access to rm.cfg for the duration of the
// call — the only way any code should mutate rm.cfg. Keep fn to just the
// mutation itself (no dialogs, no I/O) so the lock is never held longer
// than necessary.
func (rm *rcloneManager) withCfg(fn func(cfg *engine.Config)) {
	rm.cfgMu.Lock()
	defer rm.cfgMu.Unlock()
	fn(&rm.cfg)
}

func newRcloneManager(appDir string, log engine.RotatingLog, win fyne.Window) *rcloneManager {
	return &rcloneManager{
		appDir:      appDir,
		log:         log,
		store:       engine.Store{Dir: appDir, Log: func(level, msg string) { _ = log.Write(level, msg) }},
		win:         win,
		active:      map[string]*runningMount{},
		selectedRow: -1,
	}
}

// logf writes one line to RcloneManager.log. Logging failures are
// deliberately ignored here (never let a broken log stop the app) — same
// intent as the Python version's write_log().
func (rm *rcloneManager) logf(level, format string, args ...any) {
	_ = rm.log.Write(level, fmt.Sprintf(format, args...))
}

func (rm *rcloneManager) isRunning(mountID string) bool {
	rm.activeMu.Lock()
	defer rm.activeMu.Unlock()
	_, running := rm.active[mountID]
	return running
}

// persist saves the current config to mounts.json and refreshes every
// view that shows mount state (the table and the tray menu) — the single
// place both need to stay in sync, so nothing calls store.Save directly.
func (rm *rcloneManager) persist() {
	if err := rm.store.Save(rm.cfgSnapshot()); err != nil {
		rm.logf("ERROR", "[설정] mounts.json 저장 실패: %v", err)
		rm.revealWindow()
		dialog.ShowError(err, rm.win)
		return
	}
	rm.table.Refresh()
	rm.refreshTrayMenu()
}

// saveWindowSize records the window's current size so it's restored on
// the next launch. Called on hide-to-tray and before quitting — the app
// never has a plain "on close" event otherwise, since the titlebar X is
// intercepted to hide rather than close.
func (rm *rcloneManager) saveWindowSize() {
	size := rm.win.Canvas().Size()
	rm.withCfg(func(cfg *engine.Config) {
		cfg.WindowWidth = size.Width
		cfg.WindowHeight = size.Height
	})
	rm.persist()
}

// revealWindow brings the main window to the front regardless of the
// "시작 시 트레이로 최소화" setting or its current hidden/minimized
// state. Call this right before showing any dialog that represents a
// real problem (a failed mount, a save error, ...) — a dialog rendered
// on a hidden window's canvas is itself invisible, so without this a
// user running minimized-to-tray would never know anything went wrong.
func (rm *rcloneManager) revealWindow() {
	rm.win.Show()
	rm.win.RequestFocus()
}
