package main

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

const updateAssetName = "RcloneManager.zip"

// checkForUpdate looks up the latest GitHub release and, if it's newer
// than appVersion, offers to install it. When manual is false (the
// silent startup check), nothing is shown unless an update is found —
// mirrors the Python version's quiet periodic check.
func (rm *rcloneManager) checkForUpdate(manual bool) {
	go func() {
		rm.logf("INFO", "[업데이트] 확인 시작 (현재 v%s)", appVersion)
		rel, err := engine.FetchLatestRelease(nil, engine.AppReleaseAPI)
		if err != nil {
			rm.logf("ERROR", "[업데이트] 릴리스 조회 실패: %v (repo가 private면 인증 없이 실패합니다 — public인지 확인해 주세요)", err)
			if manual {
				fyne.Do(func() { dialog.ShowError(err, rm.win) })
			}
			return
		}
		rm.logf("INFO", "[업데이트] 최신 릴리스 v%s 확인됨 (asset %d개)", rel.Version, len(rel.Assets))

		if engine.CompareVersions(appVersion, rel.Version) >= 0 {
			rm.logf("INFO", "[업데이트] 이미 최신 버전")
			if manual {
				fyne.Do(func() { dialog.ShowInformation("업데이트 확인", "이미 최신 버전입니다.", rm.win) })
			}
			return
		}

		assetURL := findAsset(rel.Assets, updateAssetName)
		if assetURL == "" {
			rm.logf("ERROR", "[업데이트] v%s 릴리스에 %s 자산이 없음", rel.Version, updateAssetName)
			if manual {
				fyne.Do(func() {
					dialog.ShowInformation("업데이트 확인",
						fmt.Sprintf("v%s가 있지만 다운로드 파일(%s)을 찾지 못했습니다.", rel.Version, updateAssetName), rm.win)
				})
			}
			return // release published without the expected asset — nothing to offer
		}
		rm.logf("INFO", "[업데이트] v%s 다운로드 가능, 확인창 표시", rel.Version)

		fyne.Do(func() {
			rm.revealWindow()
			rm.appUpdateAvailable = true
			rm.updateTrayTooltip()
			rm.showUpdateConfirmDialog(rel.Version, rel.Body, func() {
				rm.performUpdate(assetURL)
			})
		})
	}()
}

// showUpdateConfirmDialog shows the release notes in a scrollable area
// (dialog.ShowConfirm's plain message can't scroll, so a long release
// body would either overflow the screen or have to be truncated — this
// lets it be read in full instead).
func (rm *rcloneManager) showUpdateConfirmDialog(version, body string, onConfirm func()) {
	label := widget.NewLabel(formatUpdateConfirmMessage(version, body))
	label.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(label)
	scroll.SetMinSize(fyne.NewSize(420, 300))

	var d dialog.Dialog
	yesBtn := widget.NewButton("업데이트", func() {
		d.Hide()
		onConfirm()
	})
	yesBtn.Importance = widget.HighImportance
	noBtn := widget.NewButton("취소", func() {
		d.Hide()
		rm.logf("INFO", "[업데이트] 사용자가 업데이트를 취소함")
	})

	content := container.NewBorder(nil, container.NewHBox(noBtn, yesBtn), nil, nil, scroll)
	d = dialog.NewCustomWithoutButtons("업데이트 가능", content, rm.win)
	d.Resize(fyne.NewSize(460, 380))
	d.Show()
}

// formatUpdateConfirmMessage builds the update-confirm dialog's message,
// including the release's own notes when there are any — shown in full,
// since the dialog itself scrolls. Pulled out as a pure function for
// testing — see update_test.go.
func formatUpdateConfirmMessage(version, body string) string {
	msg := fmt.Sprintf("새 버전 v%s가 있습니다.", version)

	body = strings.TrimSpace(body)
	if body != "" {
		msg += "\n\n" + body
	}

	return msg + "\n\n지금 업데이트할까요? (적용 후 앱이 자동으로 재시작됩니다)"
}

// findAsset returns the download URL of the release asset named name, or
// "" if not present. Pulled out as a pure function for testing.
func findAsset(assets []engine.ReleaseAsset, name string) string {
	for _, a := range assets {
		if a.Name == name {
			return a.DownloadURL
		}
	}
	return ""
}

// performUpdate downloads the release zip, swaps it into place, and
// relaunches — then quits this (now-outdated) process.
func (rm *rcloneManager) performUpdate(assetURL string) {
	rm.logf("INFO", "[업데이트] 다운로드 시작: %s", assetURL)
	progress := dialog.NewCustomWithoutButtons("업데이트 중",
		widget.NewLabel("새 버전을 다운로드하고 있습니다..."), rm.win)
	progress.Show()

	go func() {
		newExe, err := engine.DownloadAppUpdate(nil, rm.appDir, assetURL)
		if err != nil {
			rm.logf("ERROR", "[업데이트] 다운로드/추출 실패: %v", err)
			fyne.Do(func() { progress.Hide(); rm.revealWindow(); dialog.ShowError(err, rm.win) })
			return
		}
		rm.logf("INFO", "[업데이트] 다운로드 완료: %s", newExe)

		currentExe, err := os.Executable()
		if err != nil {
			rm.logf("ERROR", "[업데이트] 현재 실행 파일 경로 확인 실패: %v", err)
			fyne.Do(func() { progress.Hide(); rm.revealWindow(); dialog.ShowError(err, rm.win) })
			return
		}

		if active := rm.activeMountsSnapshot(); len(active) > 0 {
			rm.logf("INFO", "[업데이트] 재시작 전 마운트 %d개 해제", len(active))
			fyne.Do(func() {
				progress.Hide()
				progress = dialog.NewCustomWithoutButtons("업데이트 중",
					widget.NewLabel("마운트 해제 중..."), rm.win)
				progress.Show()
			})
			rm.unmountAllAndWait()
		}

		if err := engine.ApplyUpdate(currentExe, newExe); err != nil {
			rm.logf("ERROR", "[업데이트] 교체/재시작 실패: %v", err)
			fyne.Do(func() { progress.Hide(); rm.revealWindow(); dialog.ShowError(err, rm.win) })
			return
		}
		rm.logf("INFO", "[업데이트] 적용 완료, 재시작함")
		fyne.Do(func() {
			progress.Hide()
			fyne.CurrentApp().Quit() // the new version is already launching
		})
	}()
}
