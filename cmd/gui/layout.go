package main

import (
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func (rm *rcloneManager) build() {
	rm.buildTable()

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, 16))

	// Table의 MinSize()는 컬럼 폭 합계를 반영하지 않아서, 이게 없으면
	// 기본 창 너비 계산이 표 내용보다 좁게 잡힌다. 눈에는 안 보이지만
	// 폭만 차지하는 사각형으로 "창이 이 정도는 돼야 자연스럽다"는 크기
	// 힌트를 준다.
	minWidthSpacer := canvas.NewRectangle(color.Transparent)
	minWidthSpacer.SetMinSize(fyne.NewSize(tableContentWidth+20, 0))

	topContent := container.NewVBox(
		rm.buildHeaderRow(),
		rm.buildRclonePathRow(),
		rm.buildStartupOptionsRow(),
		spacer,
		minWidthSpacer,
	)

	addBtn := widget.NewButtonWithIcon("추가", nil, func() { rm.showMountDialog(nil, "") })
	upBtn := widget.NewButton("🔼", func() { rm.moveSelectedUp() })
	downBtn := widget.NewButton("🔽", func() { rm.moveSelectedDown() })
	importBtn := widget.NewButtonWithIcon("conf 가져오기", nil, func() { rm.showConfImportDialog() })
	bottomContent := container.NewBorder(nil, nil, nil,
		container.NewHBox(upBtn, downBtn, importBtn, addBtn))

	// Table 내장 헤더 대신 직접 만든 헤더(buildTableHeader)를 표 위에
	// 놓고, 그 사이에 실제 여백을 둔다.
	headerGap := canvas.NewRectangle(color.Transparent)
	headerGap.SetMinSize(fyne.NewSize(0, 8))
	tableHeader := container.NewVBox(buildTableHeader(), headerGap)
	tableArea := container.NewBorder(tableHeader, nil, nil, nil, rm.table)

	// 세로 공간이 부족해지면 상단/하단이 먼저 스크롤로 양보하고 표는
	// 항상 최소 높이를 확보하도록, 일반 Border 대신 직접 만든 레이아웃을
	// 쓴다 (layout.Border는 상단/하단에 항상 원래 필요한 높이를 그대로
	// 주고 표엔 "남는 만큼"만 줘서, 창을 줄이면 표가 음수 높이까지
	// 찌그러졌었다). Scroll로 감싸야 실제로 줄어든 크기에서도 내용끼리
	// 겹치지 않고 스크롤로 흡수된다.
	topScroll := container.NewVScroll(topContent)
	bottomScroll := container.NewVScroll(bottomContent)

	root := container.New(&verticalGuardLayout{
		top: topScroll, center: tableArea, bottom: bottomScroll,
		topContent: topContent, bottomContent: bottomContent,
		minTop: 32, minBottom: 32,
	}, topScroll, tableArea, bottomScroll)

	rm.win.SetContent(root)
}

func (rm *rcloneManager) buildHeaderRow() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("RcloneManager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	versionBadge := widget.NewButton("v"+appVersion, func() { rm.checkForUpdate(true) })
	issueBtn := widget.NewButtonWithIcon("!", nil, func() {
		if u, err := url.Parse(issueURL); err == nil {
			_ = fyne.CurrentApp().OpenURL(u)
		}
	})
	return container.NewBorder(nil, nil, container.NewHBox(title, versionBadge), issueBtn)
}

func (rm *rcloneManager) buildRclonePathRow() fyne.CanvasObject {
	rm.rcPathEntry = widget.NewEntry()
	rm.rcPathEntry.SetText(rm.cfgSnapshot().RclonePath)
	rm.rcPathEntry.SetPlaceHolder("rclone.exe 경로")
	rm.rcPathEntry.MultiLine = true
	rm.rcPathEntry.Wrapping = fyne.TextWrapWord
	rm.rcPathEntry.SetMinRowsVisible(1) // 평소엔 한 줄만큼만 차지, 긴 경로는 줄바꿈+스크롤로 처리

	browseBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		startDir := filepath.Dir(strings.TrimSpace(rm.cfgSnapshot().RclonePath))
		rm.showFilePicker("rclone.exe 선택", startDir, func(path string) {
			rm.rcPathEntry.SetText(path)
			rm.withCfg(func(cfg *engine.Config) { cfg.RclonePath = path })
			rm.persist()
			rm.refreshVersionLabel()
		})
	})

	rm.rcPathEntry.OnSubmitted = func(s string) {
		rm.withCfg(func(cfg *engine.Config) { cfg.RclonePath = strings.TrimSpace(s) })
		rm.persist()
		rm.refreshVersionLabel()
	}

	rm.rcVersionText = widget.NewButton("rclone 확인 중...", func() { rm.checkRcloneUpdate(true) })

	return container.NewBorder(nil, nil, widget.NewLabel("rclone 경로:"),
		container.NewHBox(browseBtn, rm.rcVersionText), rm.rcPathEntry)
}

func (rm *rcloneManager) buildStartupOptionsRow() fyne.CanvasObject {
	autoStart := widget.NewCheck("시작 시 자동 실행", func(checked bool) {
		if err := engine.SetStartup(checked); err != nil {
			dialog.ShowError(err, rm.win)
		}
	})
	autoStart.SetChecked(engine.IsStartupEnabled())

	autoMount := widget.NewCheck("시작 시 자동 마운트", func(checked bool) {
		rm.withCfg(func(cfg *engine.Config) { cfg.AutoMount = checked })
		rm.persist()
	})
	autoMount.SetChecked(rm.cfgSnapshot().AutoMount)

	startMinimized := widget.NewCheck("시작 시 트레이로 최소화", func(checked bool) {
		rm.withCfg(func(cfg *engine.Config) { cfg.StartMinimized = checked })
		rm.persist()
	})
	startMinimized.SetChecked(rm.cfgSnapshot().StartMinimized)

	return container.NewHBox(autoStart, autoMount, startMinimized)
}

// rcloneExePath resolves the rclone.exe to use: the explicitly configured
// path if it exists, else appDir/rclone.exe.
func (rm *rcloneManager) rcloneExePath() (string, bool) {
	p := strings.TrimSpace(rm.cfgSnapshot().RclonePath)
	if p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	fallback := filepath.Join(rm.appDir, "rclone.exe")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, true
	}
	return "", false
}

func (rm *rcloneManager) refreshVersionLabel() {
	exe, ok := rm.rcloneExePath()
	if !ok {
		rm.rcVersionText.SetText("rclone 다운로드 필요")
		return
	}
	go func() {
		text := detectLocalRcloneVersion(exe)
		fyne.Do(func() { rm.rcVersionText.SetText(text) })
	}()
}
