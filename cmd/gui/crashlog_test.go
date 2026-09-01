package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchivePreviousCrashLog(t *testing.T) {
	fixedNow := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	Scenario(t, "GIVEN 이전 실행이 남긴 크래시 로그가 있음(내용 있음) WHEN 확인 THEN 타임스탬프 붙은 이름으로 옮기고 true를 반환한다", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "RcloneManager.crash.log")
		if err := os.WriteFile(path, []byte("panic: boom\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		archived, had := archivePreviousCrashLog(path, fixedNow)
		if !had {
			t.Fatalf("크래시가 있었는데 hadCrash=false가 나옴")
		}
		if archived == path {
			t.Errorf("원래 자리에서 옮겨졌어야 하는데 그대로임: %v", archived)
		}
		if _, err := os.Stat(path); err == nil {
			t.Errorf("원래 자리엔 파일이 남아있으면 안 됨(옮겨져야 함)")
		}
		if data, err := os.ReadFile(archived); err != nil || string(data) != "panic: boom\n" {
			t.Errorf("옮겨진 파일 내용이 보존돼야 하는데 %q, err=%v", data, err)
		}
	})

	// 부정 케이스: 크래시 로그 자체가 없는(정상 종료만 있었던) 흔한 상황.
	Scenario(t, "GIVEN 크래시 로그 파일이 아예 없음 WHEN 확인 THEN false를 반환하고 아무 일도 안 한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "RcloneManager.crash.log")

		archived, had := archivePreviousCrashLog(path, fixedNow)
		if had || archived != "" {
			t.Errorf("파일이 없으면 (\"\", false)여야 하는데 (%q, %v)", archived, had)
		}
	})

	// 부정/경계 케이스: 파일은 있지만 비어있는 경우 (redirectStderrToFile이
	// 만들어만 두고 실제 panic은 한 번도 없었던 정상적인 상황) — 크래시로
	// 취급하면 안 된다.
	Scenario(t, "GIVEN 크래시 로그 파일이 있지만 내용이 비어있음 WHEN 확인 THEN false를 반환한다 (경계 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "RcloneManager.crash.log")
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}

		archived, had := archivePreviousCrashLog(path, fixedNow)
		if had || archived != "" {
			t.Errorf("빈 파일은 크래시로 취급하면 안 되는데 (%q, %v)", archived, had)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("크래시가 아니라고 판단했으면 원래 파일을 건드리면 안 됨")
		}
	})
}
