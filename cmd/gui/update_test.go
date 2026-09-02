package main

import (
	"strings"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestFindAsset(t *testing.T) {
	assets := []engine.ReleaseAsset{
		{Name: "checksums.txt", DownloadURL: "https://example.com/checksums.txt"},
		{Name: "RcloneManager.zip", DownloadURL: "https://example.com/RcloneManager.zip"},
	}

	Scenario(t, "GIVEN 릴리스 자산 목록에 원하는 이름이 있음 WHEN 자산 검색 THEN 해당 다운로드 URL을 반환한다", func(t *testing.T) {
		got := findAsset(assets, "RcloneManager.zip")
		if got != "https://example.com/RcloneManager.zip" {
			t.Errorf("got %q, RcloneManager.zip의 URL이어야 함", got)
		}
	})

	// 부정 케이스: 릴리스가 발행됐지만 원하는 자산 이름이 없는 상황
	// (예: 빌드 워크플로가 다른 이름으로 올린 경우).
	Scenario(t, "GIVEN 원하는 이름의 자산이 목록에 없음 WHEN 자산 검색 THEN 빈 문자열을 반환한다 (부정 케이스)", func(t *testing.T) {
		got := findAsset(assets, "does-not-exist.zip")
		if got != "" {
			t.Errorf("got %q, 찾는 자산이 없으면 빈 문자열이어야 함", got)
		}
	})

	Scenario(t, "GIVEN 자산 목록 자체가 nil임(릴리스에 자산이 아예 없음) WHEN 자산 검색 THEN 빈 문자열을 반환한다 (부정 케이스)", func(t *testing.T) {
		got := findAsset(nil, "RcloneManager.zip")
		if got != "" {
			t.Errorf("got %q, 자산이 없으면 빈 문자열이어야 함", got)
		}
	})
}

func TestFormatUpdateConfirmMessage(t *testing.T) {
	Scenario(t, "GIVEN 릴리스 본문이 있음 WHEN 메시지 구성 THEN 그 내용이 포함된다", func(t *testing.T) {
		msg := formatUpdateConfirmMessage("1.2.0", "- 뭔가 고쳤습니다")
		if !strings.Contains(msg, "1.2.0") {
			t.Errorf("버전이 포함돼야 하는데 없음: %q", msg)
		}
		if !strings.Contains(msg, "뭔가 고쳤습니다") {
			t.Errorf("릴리스 본문이 포함돼야 하는데 없음: %q", msg)
		}
	})

	// 부정 케이스: 릴리스 본문이 비어있는 경우(과거 릴리스 등) — 버전
	// 안내만 나오고 깨지면 안 된다.
	Scenario(t, "GIVEN 릴리스 본문이 비어있음 WHEN 메시지 구성 THEN 버전 안내만 나온다 (부정 케이스)", func(t *testing.T) {
		msg := formatUpdateConfirmMessage("1.2.0", "")
		if !strings.Contains(msg, "1.2.0") {
			t.Errorf("버전이 포함돼야 하는데 없음: %q", msg)
		}
		if strings.Count(msg, "\n\n") > 1 {
			t.Errorf("본문이 없는데 빈 문단이 남아있는 것으로 보임: %q", msg)
		}
	})

	// 경계 케이스: 다이얼로그가 이제 스크롤되므로, 아주 긴 본문도 잘리지
	// 않고 그대로 다 포함돼야 한다.
	Scenario(t, "GIVEN 릴리스 본문이 매우 긺 WHEN 메시지 구성 THEN 잘리지 않고 전부 포함된다 (경계 케이스 — 다이얼로그가 스크롤되므로)", func(t *testing.T) {
		longBody := strings.Repeat("가", 2000)
		msg := formatUpdateConfirmMessage("1.2.0", longBody)
		if !strings.Contains(msg, longBody) {
			t.Errorf("긴 본문이 잘린 것으로 보임 (메시지 길이 %d, 본문 길이 %d)", len(msg), len(longBody))
		}
	})
}
