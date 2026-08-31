package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// checkRcloneUpdate looks up the latest rclone (wiserain fork) release and
// compares it against the locally installed copy, offering to
// install/update as appropriate. manual=true (the version label was
// clicked) also reports "already up to date" / lookup errors; manual=false
// (a routine background check) stays quiet unless there's something to
// offer — same split as checkForUpdate for the app itself.
func (rm *rcloneManager) checkRcloneUpdate(manual bool) {
	go func() {
		latest, err := engine.FetchLatestReleaseTag(nil, engine.RcloneReleaseAPI)
		if err != nil {
			rm.logf("ERROR", "[버전] rclone 릴리스 조회 실패: %v", err)
			if manual {
				fyne.Do(func() { dialog.ShowError(err, rm.win) })
			}
			return
		}
		rm.latestRcloneVersion = latest

		exe, ok := rm.rcloneExePath()
		if !ok {
			if manual {
				fyne.Do(func() {
					dialog.ShowConfirm("rclone 설치",
						fmt.Sprintf("rclone v%s를 설치할까요?", latest),
						func(yes bool) {
							if yes {
								rm.installOrUpdateRclone(latest, nil)
							}
						}, rm.win)
				})
			}
			return
		}

		local, ok := localRcloneVersionRaw(exe)
		if !ok {
			if manual {
				fyne.Do(func() {
					dialog.ShowInformation("rclone", "로컬 rclone 버전을 확인할 수 없습니다.", rm.win)
				})
			}
			return
		}

		if engine.CompareVersions(local, latest) >= 0 {
			if manual {
				fyne.Do(func() { dialog.ShowInformation("rclone", "이미 최신 버전입니다.", rm.win) })
			}
			return
		}

		active := rm.activeMountsSnapshot()
		fyne.Do(func() {
			rm.revealWindow()
			rm.rcloneUpdateAvailable = true
			rm.updateTrayTooltip()
			msg := fmt.Sprintf("rclone v%s → v%s로 업데이트할까요?", local, latest)
			if len(active) > 0 {
				msg += "\n\n현재 마운트 중인 드라이브가 있습니다 — 업데이트 전 모두 해제하고, 완료 후 자동으로 다시 마운트합니다."
			}
			dialog.ShowConfirm("rclone 업데이트", msg, func(yes bool) {
				if yes {
					rm.installOrUpdateRclone(latest, active)
				}
			}, rm.win)
		})
	}()
}

// localRcloneVersionRaw runs `rclone version` and returns the raw parsed
// version string (for comparison), separate from
// formatRcloneVersionLabel's display-formatted text.
func localRcloneVersionRaw(exe string) (string, bool) {
	cmd := exec.Command(exe, "version")
	engine.ConfigureBackgroundProcess(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	return engine.ParseLocalRcloneVersion(string(out))
}

// activeMountsSnapshot returns the Mount configs currently running, so
// they can be unmounted before an rclone update and remounted after.
func (rm *rcloneManager) activeMountsSnapshot() []engine.Mount {
	rm.activeMu.Lock()
	runningIDs := make(map[string]bool, len(rm.active))
	for id := range rm.active {
		runningIDs[id] = true
	}
	rm.activeMu.Unlock()

	var active []engine.Mount
	for _, m := range rm.cfgSnapshot().Mounts {
		if runningIDs[m.ID] {
			active = append(active, m)
		}
	}
	return active
}

// installOrUpdateRclone unmounts remountAfter (if any) and waits for it to
// actually finish — otherwise rclone.exe may still be locked by the very
// process being stopped — then downloads version, installs it, updates
// the configured rclone path, and remounts. Mirrors the Python version's
// _do_rc_down().
func (rm *rcloneManager) installOrUpdateRclone(version string, remountAfter []engine.Mount) {
	rm.logf("INFO", "[rclone] v%s 설치/업데이트 시작", version)

	destDir := rm.appDir
	if p := strings.TrimSpace(rm.cfgSnapshot().RclonePath); p != "" {
		destDir = filepath.Dir(p)
	}

	go func() {
		rm.updatingRclone.Store(true)
		defer rm.updatingRclone.Store(false)

		var progress *dialog.CustomDialog

		if len(remountAfter) > 0 {
			fyne.Do(func() {
				rm.revealWindow()
				progress = dialog.NewCustomWithoutButtons("rclone 업데이트 중",
					widget.NewLabel("마운트 해제 중..."), rm.win)
				progress.Show()
			})
			rm.unmountAllAndWait()
			fyne.Do(func() { progress.Hide() })
		}

		fyne.Do(func() {
			rm.revealWindow()
			progress = dialog.NewCustomWithoutButtons("rclone 업데이트 중",
				widget.NewLabel(fmt.Sprintf("rclone v%s 다운로드 중...", version)), rm.win)
			progress.Show()
			rm.rcVersionText.SetText("다운로드 중...")
		})

		status, err := engine.DownloadRclone(nil, destDir, version)
		if err != nil {
			rm.logf("ERROR", "[rclone] 다운로드 실패: %v", err)
			fyne.Do(func() {
				progress.Hide()
				rm.revealWindow()
				dialog.ShowError(err, rm.win)
				rm.refreshVersionLabel()
			})
			return
		}

		switch status {
		case engine.DownloadStatusInstalled:
			newPath := filepath.Join(destDir, "rclone.exe")
			rm.logf("INFO", "[rclone] v%s 설치 완료: %s", version, newPath)
			fyne.Do(func() {
				progress.Hide()
				rm.withCfg(func(cfg *engine.Config) { cfg.RclonePath = newPath })
				rm.rcPathEntry.SetText(newPath)
				rm.persist()
				rm.refreshVersionLabel()
				rm.rcloneUpdateAvailable = false
				rm.updateTrayTooltip()
				dialog.ShowInformation("완료", "rclone 설치/업데이트 완료!", rm.win)
			})
			for _, m := range remountAfter {
				rm.mount(m)
			}

		case engine.DownloadStatusManual:
			rm.logf("WARN", "[rclone] 실행 중 파일 잠김 — rclone_new.exe로 저장, 수동 교체 필요")
			fyne.Do(func() {
				progress.Hide()
				rm.refreshVersionLabel()
				rm.revealWindow()
				dialog.ShowInformation("수동 교체 필요",
					fmt.Sprintf("rclone.exe가 사용 중이라 자동 교체하지 못했습니다.\n%s 폴더의 rclone_new.exe를 rclone.exe로 직접 바꿔주세요.", destDir),
					rm.win)
			})
		}
	}()
}
