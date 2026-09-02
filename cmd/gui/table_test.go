package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDisplayDrive(t *testing.T) {
	Scenario(t, "GIVEN 드라이브 문자가 비어있음 WHEN 표시 텍스트 결정 THEN '(자동)'으로 보여준다", func(t *testing.T) {
		if got := displayDrive(""); got != "(자동)" {
			t.Errorf("displayDrive(\"\") = %q, 기대값 %q", got, "(자동)")
		}
	})
	Scenario(t, "GIVEN 드라이브 문자가 공백뿐임 WHEN 표시 텍스트 결정 THEN '(자동)'으로 보여준다", func(t *testing.T) {
		if got := displayDrive("  "); got != "(자동)" {
			t.Errorf("displayDrive(\"  \") = %q, 기대값 %q", got, "(자동)")
		}
	})
	Scenario(t, "GIVEN 드라이브 문자가 지정돼 있음(경로 구분자 없음) WHEN 표시 텍스트 결정 THEN 그 값을 그대로 보여준다", func(t *testing.T) {
		if got := displayDrive("E:"); got != "E:" {
			t.Errorf("displayDrive(\"E:\") = %q, 기대값 %q", got, "E:")
		}
	})

	// 핵심 규칙: 폴더 경로면 길든 짧든 항상 마지막 폴더명만 남기고 그
	// 앞은 "..."으로 — 실제로 겹치던 옆 컬럼과의 충돌도 이걸로 해결된다.
	Scenario(t, "GIVEN 폴더 경로가 짧음 WHEN 표시 텍스트 결정 THEN 그래도 마지막 폴더명만 남기고 앞은 ...으로 보여준다", func(t *testing.T) {
		if got := displayDrive(`D:\Plugin\rclone`); got != `...\rclone` {
			t.Errorf("got %q, 기대값 %q", got, `...\rclone`)
		}
	})

	Scenario(t, "GIVEN 폴더 경로가 김 WHEN 표시 텍스트 결정 THEN 마지막 폴더명 쪽이 남고 표시 길이 제한을 넘지 않는다", func(t *testing.T) {
		long := "D:\\Users\\누군가\\Documents\\정말길고긴폴더이름모음집입니다"
		got := displayDrive(long)
		if !strings.HasPrefix(got, "...") {
			t.Errorf("잘렸으면 앞에 ...이 붙어야 하는데 %q", got)
		}
		if len([]rune(got)) > maxDisplayDriveLength {
			t.Errorf("표시 길이 제한을 넘음: %q (%d자)", got, len([]rune(got)))
		}
	})

	// 경계 케이스: 마지막 폴더명 자체가 표시 길이 제한보다 긴 경우 —
	// 그때는 안전장치로 그 조각의 끝부분만 문자 수로 자른다.
	Scenario(t, "GIVEN 마지막 폴더명 자체가 표시 길이 제한보다 김 WHEN 표시 텍스트 결정 THEN 그 조각의 뒷부분만 남기고 문자 수로 자른다 (경계 케이스)", func(t *testing.T) {
		long := "D:\\" + strings.Repeat("가", maxDisplayDriveLength+10)
		got := displayDrive(long)
		if len([]rune(got)) > maxDisplayDriveLength {
			t.Errorf("표시 길이 제한을 넘음: %q (%d자)", got, len([]rune(got)))
		}
		if !strings.HasPrefix(got, "...") {
			t.Errorf("잘렸으면 앞에 ...이 붙어야 하는데 %q", got)
		}
	})

	// 부정/경계 케이스: 경로가 구분자로 끝나서 마지막 조각이 빈 경우.
	Scenario(t, "GIVEN 경로가 구분자로 끝나서 마지막 폴더명 조각이 없음 WHEN 표시 텍스트 결정 THEN panic 없이 문자 수로만 자른다 (부정 케이스)", func(t *testing.T) {
		got := displayDrive(`D:\Plugin\`)
		if !utf8.ValidString(got) {
			t.Errorf("결과가 깨진 UTF-8임: %q", got)
		}
	})

	// 부정 케이스: 한글처럼 멀티바이트 문자가 섞인 경로를 잘라도
	// 깨진(잘못된 UTF-8) 문자가 나오면 안 된다.
	Scenario(t, "GIVEN 한글이 섞인 긴 경로 WHEN 표시 텍스트 결정 THEN 잘린 결과가 유효한 UTF-8이다 (부정 케이스)", func(t *testing.T) {
		long := "D:\\Users\\누군가\\Documents\\정말길고긴폴더이름모음집입니다"
		got := displayDrive(long)
		if !utf8.ValidString(got) {
			t.Errorf("잘린 결과가 깨진 UTF-8임: %q", got)
		}
	})
}

func TestStatusLabel(t *testing.T) {
	Scenario(t, "GIVEN 마운트가 실행 중임 WHEN 상태 라벨 결정 THEN '연결됨'을 보여준다", func(t *testing.T) {
		if got := statusLabel(true); got != "연결됨" {
			t.Errorf("statusLabel(true) = %q, 기대값 %q", got, "연결됨")
		}
	})
	Scenario(t, "GIVEN 마운트가 실행 중이 아님 WHEN 상태 라벨 결정 THEN '해제됨'을 보여준다", func(t *testing.T) {
		if got := statusLabel(false); got != "해제됨" {
			t.Errorf("statusLabel(false) = %q, 기대값 %q", got, "해제됨")
		}
	})
}

func TestToggleLabel(t *testing.T) {
	Scenario(t, "GIVEN 마운트가 실행 중임 WHEN 토글 버튼 라벨 결정 THEN '해제'를 보여준다", func(t *testing.T) {
		if got := toggleLabel(true); got != "해제" {
			t.Errorf("toggleLabel(true) = %q, 기대값 %q", got, "해제")
		}
	})
	Scenario(t, "GIVEN 마운트가 실행 중이 아님 WHEN 토글 버튼 라벨 결정 THEN '마운트'를 보여준다", func(t *testing.T) {
		if got := toggleLabel(false); got != "마운트" {
			t.Errorf("toggleLabel(false) = %q, 기대값 %q", got, "마운트")
		}
	})
}

func TestToggleSelection(t *testing.T) {
	Scenario(t, "GIVEN 선택된 행이 없음(-1) WHEN 어떤 행을 클릭 THEN 그 행이 선택된다", func(t *testing.T) {
		if got := toggleSelection(-1, 2); got != 2 {
			t.Errorf("got %d, 기대값 2", got)
		}
	})

	Scenario(t, "GIVEN 이미 선택된 행을 WHEN 다시 클릭 THEN 선택이 해제된다(-1)", func(t *testing.T) {
		if got := toggleSelection(2, 2); got != -1 {
			t.Errorf("got %d, 기대값 -1", got)
		}
	})

	Scenario(t, "GIVEN 다른 행이 선택된 상태에서 WHEN 새 행을 클릭 THEN 선택이 그 새 행으로 옮겨간다", func(t *testing.T) {
		if got := toggleSelection(1, 3); got != 3 {
			t.Errorf("got %d, 기대값 3", got)
		}
	})
}
