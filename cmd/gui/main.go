// Command gui is RcloneManager's Windows desktop app: a Fyne front end
// over the internal/engine package.
//
// File layout (by concern, not by widget type):
//
//	main.go         - entrypoint, app/window bootstrap
//	model.go        - rcloneManager state, small shared helpers (logf, persist, isRunning)
//	rows.go         - combined 원본/마운트 table row model
//	layout.go       - top-level window layout (header, path row, options row)
//	table.go        - the mount list table
//	dialogs.go      - add/edit/delete/mount-failure dialogs
//	confimport.go   - rclone.conf import dialog
//	reorder.go      - 위/아래 list reordering
//	mount.go        - mount/unmount process lifecycle, auto-mount, network monitor
//	update.go       - app self-update check/download/apply
//	rcloneupdate.go - rclone.exe install/update check/download/apply
//	tray.go         - system tray icon + menu
//	crashlog_windows.go / crashlog_other.go - redirect stderr to a file so
//	                  an unrecovered panic's stack trace is actually
//	                  captured, instead of vanishing (no console on a
//	                  -H=windowsgui build)
package main

import (
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

const appVersion = "1.0.7"
const issueURL = "https://github.com/Murianwind/rclone_mount_manager_go/issues/new"

// 컬럼 합(약 686px) + 창 여백/스크롤바를 감안해 기본 창 너비에 여유를 둔다 —
// 정확히 맞추면 창 테두리에 마지막 컬럼이 잘려 보인다.
const defaultWindowWidth = 820
const defaultWindowHeight = 520

func main() {
	appDir := mustAppDir()

	crashLogPath := filepath.Join(appDir, "RcloneManager.crash.log")
	archivedCrash, hadCrash := archivePreviousCrashLog(crashLogPath, time.Now())
	redirectStderrToFile(crashLogPath)

	log := engine.RotatingLog{Path: filepath.Join(appDir, "RcloneManager.log"), MaxLines: 1000}

	if exe, err := os.Executable(); err == nil {
		go engine.CleanupPreviousExe(exe)
	}

	fyneApp := app.NewWithID("com.murianwind.rclonemanager")
	fyneApp.SetIcon(appIcon)
	fyneApp.Settings().SetTheme(newWindowsFontTheme(fyneApp.Settings().Theme()))
	win := fyneApp.NewWindow("RcloneManager")
	win.SetIcon(appIcon)

	rm := newRcloneManager(appDir, log, win)
	rm.logf("INFO", "[시작] RcloneManager v%s 시작됨", appVersion)
	if hadCrash {
		rm.logf("WARN", "[크래시] 이전 실행이 예기치 않게 종료된 기록이 있습니다 — %s 파일을 확인해 주세요", archivedCrash)
	}

	if fixed, err := engine.CheckAndFixStartup(); err != nil {
		rm.logf("WARN", "[시작프로그램] 경로 재등록 확인 실패: %v", err)
	} else if fixed {
		rm.logf("INFO", "[시작프로그램] 등록된 경로가 실행 파일과 달라 재등록함")
	}

	cfg, err := rm.store.Load()
	if err != nil {
		rm.logf("ERROR", "[설정] mounts.json 로드 실패: %v", err)
		rm.revealWindow()
		dialog.ShowError(err, win)
	}
	rm.cfg = cfg

	win.Resize(fyne.NewSize(savedOr(cfg.WindowWidth, defaultWindowWidth), savedOr(cfg.WindowHeight, defaultWindowHeight)))

	rm.build()
	rm.setupTray(fyneApp)
	rm.refreshVersionLabel()
	rm.startNetworkMonitor()
	rm.startScheduleMonitor()
	enforceMinWindowSize(win)

	win.SetCloseIntercept(func() {
		fyne.Do(func() {
			rm.saveWindowSize()
			rm.selectedRow = -1
			rm.table.Refresh()
			win.Hide()
		})
	})

	fyneApp.Lifecycle().SetOnStarted(func() {
		rm.checkForUpdate(false)
		rm.checkRcloneUpdate(false)
	})

	if rm.cfgSnapshot().StartMinimized {
		fyneApp.Run()
	} else {
		win.Show()
		fyneApp.Run()
	}
}

func mustAppDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func savedOr(saved, fallback float32) float32 {
	if saved > 0 {
		return saved
	}
	return fallback
}
