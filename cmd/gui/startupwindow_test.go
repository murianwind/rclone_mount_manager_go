package main

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestWindowStartModeFor(t *testing.T) {
	Scenario(t, "GIVEN 트레이로 최소화 시작이 켜져 있음 WHEN 시작 방식 결정 THEN 창을 만들되 숨긴 채로 시작한다", func(t *testing.T) {
		if got := windowStartModeFor(true); got != startWindowHidden {
			t.Errorf("got %v, 기대값 startWindowHidden", got)
		}
	})
	Scenario(t, "GIVEN 트레이로 최소화 시작이 꺼져 있음 WHEN 시작 방식 결정 THEN 창을 보여준다", func(t *testing.T) {
		if got := windowStartModeFor(false); got != startWindowShown {
			t.Errorf("got %v, 기대값 startWindowShown", got)
		}
	})
}

// prepareNativeWindow는 실제 Windows 네이티브 핸들을 만드는 코드라 헤드리스 테스트로는
// 핸들 존재 여부를 직접 확인할 수 없다 — 최소한 어떤 모드에서도 panic 없이 끝나는지만 보장한다.
func TestPrepareNativeWindowDoesNotPanic(t *testing.T) {
	Scenario(t, "GIVEN 어떤 시작 방식이든 WHEN prepareNativeWindow THEN panic 없이 끝난다", func(t *testing.T) {
		fynetest.NewApp()
		for _, mode := range []windowStartMode{startWindowShown, startWindowHidden} {
			win := fynetest.NewWindow(widget.NewLabel("x"))
			prepareNativeWindow(win, mode)
		}
	})
}
