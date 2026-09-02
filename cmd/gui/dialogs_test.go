package main

import (
	"strings"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestMountDialogTitle(t *testing.T) {
	Scenario(t, "GIVEN 새 마운트를 추가하는 상황 WHEN 다이얼로그 제목 결정 THEN '마운트 추가'가 된다", func(t *testing.T) {
		if got := mountDialogTitle(false); got != "마운트 추가" {
			t.Errorf("mountDialogTitle(false) = %q, 기대값 %q", got, "마운트 추가")
		}
	})
	Scenario(t, "GIVEN 기존 마운트를 편집하는 상황 WHEN 다이얼로그 제목 결정 THEN '마운트 편집'이 된다", func(t *testing.T) {
		if got := mountDialogTitle(true); got != "마운트 편집" {
			t.Errorf("mountDialogTitle(true) = %q, 기대값 %q", got, "마운트 편집")
		}
	})
}

func TestMountIDFor(t *testing.T) {
	Scenario(t, "GIVEN 기존 마운트를 편집하는 상황 WHEN ID 결정 THEN 기존 ID를 그대로 유지한다", func(t *testing.T) {
		existing := &engine.Mount{ID: "abc-123"}
		if got := mountIDFor(existing); got != "abc-123" {
			t.Errorf("mountIDFor(existing) = %q, 기존 ID(abc-123)가 유지돼야 함", got)
		}
	})
	Scenario(t, "GIVEN 새 마운트를 추가하는 상황(existing == nil) WHEN ID 결정 THEN 새 ID를 발급한다", func(t *testing.T) {
		if got := mountIDFor(nil); got == "" {
			t.Errorf("mountIDFor(nil)은 비어있지 않은 새 ID를 반환해야 함")
		}
	})
}

func TestMountFailureMessage(t *testing.T) {
	m := engine.Mount{Remote: "PLEX", RemotePath: "KODI"}

	Scenario(t, "GIVEN rclone의 실제 오류 내용이 있음 WHEN 실패 메시지 구성 THEN 리모트 경로·오류 상세·로그 경로가 전부 포함된다", func(t *testing.T) {
		msg := mountFailureMessage(m, "CRITICAL: Fatal error: failed to mount", "/tmp/RcloneManager.log")
		if !strings.Contains(msg, "PLEX:KODI") {
			t.Errorf("메시지에 remote:path가 포함돼야 함: %q", msg)
		}
		if !strings.Contains(msg, "CRITICAL: Fatal error") {
			t.Errorf("메시지에 rclone 오류 상세가 포함돼야 함: %q", msg)
		}
		if !strings.Contains(msg, "/tmp/RcloneManager.log") {
			t.Errorf("메시지에 로그 파일 경로가 포함돼야 함: %q", msg)
		}
	})

	// 부정 케이스: rclone이 stderr에 아무것도 안 남긴 경우에도 메시지가
	// 텅 비면 안 되고, 대체 안내 문구가 나와야 한다.
	Scenario(t, "GIVEN rclone이 오류 메시지를 전혀 출력하지 않음 WHEN 실패 메시지 구성 THEN 대체 안내 문구를 보여준다 (부정 케이스)", func(t *testing.T) {
		msg := mountFailureMessage(m, "", "/tmp/RcloneManager.log")
		if !strings.Contains(msg, "별도 오류 메시지를 출력하지 않았습니다") {
			t.Errorf("상세 내용이 없을 때의 대체 문구가 있어야 함: %q", msg)
		}
	})
}

func TestMountFromForm(t *testing.T) {
	Scenario(t, "GIVEN 새 마운트를 추가하는 경우(existing이 nil) WHEN 조립 THEN AutoMount와 Schedules는 빈 상태로 시작한다", func(t *testing.T) {
		m := mountFromForm(nil, "gds", "video", "E:", "", "off", "")
		if m.AutoMount {
			t.Errorf("새 마운트인데 AutoMount가 true로 나옴")
		}
		if len(m.Schedules) != 0 {
			t.Errorf("새 마운트인데 Schedules가 비어있지 않음: %+v", m.Schedules)
		}
	})

	// 회귀 테스트: 실제로 있었던 버그 — 일정이 있는 마운트를 편집(캐시 모드만
	// 바꾸는 것도 포함)하면 Schedules가 조용히 사라졌었다.
	Scenario(t, "GIVEN 기존 마운트에 일정이 등록돼 있음 WHEN 다른 필드만 바꿔서 편집 THEN 일정이 그대로 유지된다 (회귀 테스트)", func(t *testing.T) {
		existing := &engine.Mount{
			ID: "m1", Remote: "gds", RemotePath: "video", Drive: "E:",
			AutoMount: false,
			Schedules: []engine.Schedule{{StartHour: 9, StartMinute: 0, EndHour: 18, EndMinute: 0, Days: []int{1, 2, 3, 4, 5}}},
		}

		m := mountFromForm(existing, "gds", "video", "E:", "", "writes", "") // 캐시 모드만 바꿔서 저장

		if len(m.Schedules) != 1 {
			t.Fatalf("일정이 사라짐: %+v", m.Schedules)
		}
		if m.Schedules[0].StartHour != 9 {
			t.Errorf("일정 내용이 바뀜: %+v", m.Schedules[0])
		}
	})

	Scenario(t, "GIVEN 기존 마운트의 자동 마운트가 켜져 있음 WHEN 편집 THEN 자동 마운트 상태가 그대로 유지된다", func(t *testing.T) {
		existing := &engine.Mount{ID: "m1", Remote: "gds", AutoMount: true}
		m := mountFromForm(existing, "gds", "", "", "", "", "")
		if !m.AutoMount {
			t.Errorf("AutoMount가 유지돼야 하는데 false가 됨")
		}
	})

	Scenario(t, "GIVEN 편집 중임 WHEN 조립 THEN ID는 기존 것을 그대로 쓴다 (경계 케이스)", func(t *testing.T) {
		existing := &engine.Mount{ID: "keep-me", Remote: "gds"}
		m := mountFromForm(existing, "gds2", "", "", "", "", "")
		if m.ID != "keep-me" {
			t.Errorf("ID가 바뀌면 안 되는데 %q", m.ID)
		}
	})
}
