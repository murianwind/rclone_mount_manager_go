package main

import (
	"testing"

	"fyne.io/fyne/v2/dialog"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestShowGated(t *testing.T) {
	newEnv := func() (*rcloneManager, func() dialog.Dialog, *int, *[]dialog.Dialog) {
		fynetest.NewApp()
		win := fynetest.NewWindow(widget.NewLabel("x"))
		rm := &rcloneManager{win: win}
		builds := 0
		var made []dialog.Dialog
		build := func() dialog.Dialog {
			builds++
			d := dialog.NewInformation("t", "m", win)
			made = append(made, d)
			return d
		}
		return rm, build, &builds, &made
	}

	// 회귀 테스트: 계속 실패하는 자동 마운트가 10초마다 같은 알림 창을 새로 쌓던 문제.
	Scenario(t, "GIVEN 같은 키의 알림이 이미 떠 있음 WHEN 또 요청 THEN 새로 만들지 않는다 (알림 누적 회귀 테스트)", func(t *testing.T) {
		rm, build, builds, _ := newEnv()
		rm.showGated("k", build)
		rm.showGated("k", build)
		rm.showGated("k", build)
		if *builds != 1 {
			t.Errorf("알림은 1번만 만들어져야 하는데 %d번", *builds)
		}
	})

	Scenario(t, "GIVEN 떠 있던 알림을 사용자가 닫음 WHEN 같은 키로 다시 요청 THEN 새로 뜬다", func(t *testing.T) {
		rm, build, builds, made := newEnv()
		rm.showGated("k", build)
		(*made)[0].Hide() // 사용자가 확인 버튼을 누른 것과 같은 경로
		rm.showGated("k", build)
		if *builds != 2 {
			t.Errorf("닫은 뒤엔 새로 떠야 하는데 총 %d번", *builds)
		}
	})

	Scenario(t, "GIVEN 서로 다른 키 WHEN 각각 요청 THEN 둘 다 뜬다", func(t *testing.T) {
		rm, build, builds, _ := newEnv()
		rm.showGated("a", build)
		rm.showGated("b", build)
		if *builds != 2 {
			t.Errorf("서로 다른 키는 각각 떠야 하는데 %d번", *builds)
		}
	})
}
