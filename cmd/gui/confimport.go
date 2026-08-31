package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// showConfImportDialog는 파일 선택 다이얼로그로 rclone.conf를 고르게 한 뒤,
// 그 안의 리모트를 체크박스로 선택해 cfg.Remotes(원본 목록)에 추가한다.
// 기존 Python 버전의 _import_conf()/ConfImportDialog에 대응한다.
func (rm *rcloneManager) showConfImportDialog() {
	startDir := rm.appDir
	if defaultConf, ok := engine.FindDefaultRcloneConf(rm.appDir); ok {
		startDir = parentDir(defaultConf)
	}

	rm.showFilePicker("rclone.conf 선택", startDir, func(path string) {
		remotes := engine.ParseRcloneConf(path)
		if len(remotes) == 0 {
			dialog.ShowInformation("가져오기", "해당 파일에서 리모트를 찾지 못했습니다.", rm.win)
			return
		}
		rm.showRemoteSelectDialog(remotes)
	})
}

// showRemoteSelectDialog는 파싱된 리모트 후보 목록을 체크박스로 보여주고,
// 선택된 것만 cfg.Remotes에 (이미 있는 이름은 건너뛰고) 추가한다.
func (rm *rcloneManager) showRemoteSelectDialog(candidates []engine.Remote) {
	existing := make(map[string]bool)
	for _, r := range rm.cfgSnapshot().Remotes {
		existing[r.Name] = true
	}

	checks := make([]*widget.Check, len(candidates))
	rows := make([]fyne.CanvasObject, 0, len(candidates))
	for i, r := range candidates {
		label := fmt.Sprintf("%s [%s]", r.Name, r.Type)
		if existing[r.Name] {
			label += "  (이미 있음)"
		}
		c := widget.NewCheck(label, nil)
		c.SetChecked(!existing[r.Name]) // 새 리모트는 기본 선택, 이미 있는 건 기본 해제
		checks[i] = c
		rows = append(rows, c)
	}

	content := container.NewVBox(rows...)
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(360, 300))

	dialog.ShowCustomConfirm("리모트 선택", "가져오기", "취소", scroll, func(ok bool) {
		if !ok {
			return
		}
		added := 0
		rm.withCfg(func(cfg *engine.Config) {
			for i, r := range candidates {
				if checks[i].Checked && !existing[r.Name] {
					cfg.Remotes = append(cfg.Remotes, r)
					added++
				}
			}
		})
		if added > 0 {
			rm.persist()
		}
	}, rm.win)
}

func parentDir(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[:i]
	}
	return path
}
