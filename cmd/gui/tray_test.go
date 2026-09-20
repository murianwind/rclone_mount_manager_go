package main

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestTrayShortLabel(t *testing.T) {
	Scenario(t, "GIVEN 드라이브 문자가 지정된 마운트 WHEN 짧은 라벨 결정 THEN 드라이브 문자를 쓴다", func(t *testing.T) {
		got := trayShortLabel(engine.Mount{Remote: "gds", Drive: "G:"})
		if got != "G:" {
			t.Errorf("got %q, 기대값 %q", got, "G:")
		}
	})
	Scenario(t, "GIVEN 드라이브 문자가 없는 마운트 WHEN 짧은 라벨 결정 THEN 리모트 이름으로 대체한다", func(t *testing.T) {
		got := trayShortLabel(engine.Mount{Remote: "gds"})
		if got != "gds" {
			t.Errorf("got %q, 기대값 %q (리모트 이름 폴백)", got, "gds")
		}
	})
}

func TestTrayDisplayLabel(t *testing.T) {
	m := engine.Mount{Remote: "PLEX", RemotePath: "KODI", Drive: "E:"}

	Scenario(t, "GIVEN 마운트가 실행 중임 WHEN 트레이 메뉴 항목 텍스트 구성 THEN ■ 아이콘과 함께 표시된다", func(t *testing.T) {
		got := trayDisplayLabel(m, true)
		if got != "■  E:  (PLEX:KODI)" {
			t.Errorf("got %q, 기대값 %q", got, "■  E:  (PLEX:KODI)")
		}
	})

	Scenario(t, "GIVEN 마운트가 중지 상태임 WHEN 트레이 메뉴 항목 텍스트 구성 THEN ▶ 아이콘과 함께 표시된다", func(t *testing.T) {
		got := trayDisplayLabel(m, false)
		if got != "▶  E:  (PLEX:KODI)" {
			t.Errorf("got %q, 기대값 %q", got, "▶  E:  (PLEX:KODI)")
		}
	})

	// 부정/경계 케이스: 서브경로가 없는 리모트(예: "nas:")는 괄호 안에
	// 콜론만 덩그러니 남으면 안 된다.
	Scenario(t, "GIVEN 서브경로가 없는 리모트(예: nas:) WHEN 트레이 메뉴 항목 텍스트 구성 THEN 괄호 안에 불필요한 콜론이 남지 않는다 (경계 케이스)", func(t *testing.T) {
		bare := engine.Mount{Remote: "nas"}
		got := trayDisplayLabel(bare, false)
		if got != "▶  nas  (nas)" {
			t.Errorf("got %q, 기대값 %q", got, "▶  nas  (nas)")
		}
	})
}

func TestTrayTooltipText(t *testing.T) {
	Scenario(t, "GIVEN 업데이트가 없음 WHEN 툴팁 텍스트 결정 THEN 그냥 앱 이름만 보여준다", func(t *testing.T) {
		if got := trayTooltipText(false, false); got != "RcloneManager" {
			t.Errorf("got %q, 기대값 %q", got, "RcloneManager")
		}
	})

	Scenario(t, "GIVEN 앱 업데이트만 있음 WHEN 툴팁 텍스트 결정 THEN 앱 업데이트라고 구체적으로 보여준다", func(t *testing.T) {
		got := trayTooltipText(true, false)
		if got != "RcloneManager — 앱 업데이트 있음" {
			t.Errorf("got %q", got)
		}
	})

	Scenario(t, "GIVEN rclone 업데이트만 있음 WHEN 툴팁 텍스트 결정 THEN rclone 업데이트라고 구체적으로 보여준다", func(t *testing.T) {
		got := trayTooltipText(false, true)
		if got != "RcloneManager — rclone 업데이트 있음" {
			t.Errorf("got %q", got)
		}
	})

	// 부정/경계 케이스: 앱과 rclone 업데이트가 동시에 있으면 둘 다 있다고
	// 보여줘야 한다 — 둘 중 하나를 조용히 숨기면 안 된다.
	Scenario(t, "GIVEN 앱과 rclone 업데이트가 둘 다 있음 WHEN 툴팁 텍스트 결정 THEN 둘 다 있다고 보여준다 (경계 케이스)", func(t *testing.T) {
		got := trayTooltipText(true, true)
		if got != "RcloneManager — 앱/rclone 업데이트 있음" {
			t.Errorf("got %q", got)
		}
	})
}

func TestRefreshTrayMenuBeforeSetup(t *testing.T) {
	Scenario(t, "GIVEN setupTray가 아직 실행되지 않음(trayReady=false) WHEN refreshTrayMenu 호출 THEN panic 없이 조용히 아무것도 안 한다 (회귀 테스트 — autoMountAll 레이스 대응)", func(t *testing.T) {
		fynetest.NewApp()
		win := fynetest.NewApp().NewWindow("test")
		rm := &rcloneManager{win: win, cfg: engine.Config{}}
		// trayReady는 기본값(false)으로 둔다 — setupTray를 아직 안 부른 상태를 재현.
		rm.refreshTrayMenu() // panic이 나면 안 됨
	})

	Scenario(t, "GIVEN setupTray가 이미 실행됨(trayReady=true) WHEN refreshTrayMenu 호출 THEN panic 없이 정상적으로 진행된다", func(t *testing.T) {
		fynetest.NewApp()
		win := fynetest.NewApp().NewWindow("test")
		rm := &rcloneManager{win: win, cfg: engine.Config{}, trayReady: true}
		rm.refreshTrayMenu() // panic이 나면 안 됨
	})
}
