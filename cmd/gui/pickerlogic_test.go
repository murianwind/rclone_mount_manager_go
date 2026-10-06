package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPickerEntries(t *testing.T) {
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, "폴더B"), 0o755)
	_ = os.Mkdir(filepath.Join(dir, "폴더A"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "파일.txt"), []byte("x"), 0o644)

	Scenario(t, "GIVEN 파일 선택 모드 WHEN 항목 목록 조회 THEN 폴더가 먼저, 그다음 파일이 나온다", func(t *testing.T) {
		got := pickerEntries(dir, pickFile)
		if len(got) != 3 || !got[0].isDir || !got[1].isDir || got[2].isDir {
			t.Errorf("폴더 2개 → 파일 1개 순서여야 하는데 %+v", got)
		}
	})

	Scenario(t, "GIVEN 폴더 선택 모드 WHEN 항목 목록 조회 THEN 폴더만 나온다 (파일은 목록에 없다)", func(t *testing.T) {
		got := pickerEntries(dir, pickDir)
		if len(got) != 2 {
			t.Fatalf("폴더 2개만 나와야 하는데 %+v", got)
		}
		for _, e := range got {
			if !e.isDir {
				t.Errorf("파일이 섞임: %+v", e)
			}
		}
	})

	// 부정 케이스: 읽을 수 없는 경로 — 선택창이 죽지 않고 빈 목록이어야 한다.
	Scenario(t, "GIVEN 존재하지 않는 경로 WHEN 항목 목록 조회 THEN panic 없이 빈 목록이다 (부정 케이스)", func(t *testing.T) {
		if got := pickerEntries(filepath.Join(dir, "없는경로"), pickFile); len(got) != 0 {
			t.Errorf("빈 목록이어야 하는데 %+v", got)
		}
	})
}

func TestPickerClick(t *testing.T) {
	cur := filepath.Join("base", "dir")
	entries := []fileEntry{{name: "sub", isDir: true}, {name: "a.exe", isDir: false}}

	Scenario(t, "GIVEN '..' 행(0번)을 클릭 WHEN 처리 THEN 상위 폴더로 이동한다", func(t *testing.T) {
		newDir, file := pickerClick(cur, entries, 0)
		if newDir != "base" || file != "" {
			t.Errorf("got (%q, %q)", newDir, file)
		}
	})

	Scenario(t, "GIVEN 폴더 행을 클릭 WHEN 처리 THEN 그 폴더로 들어간다", func(t *testing.T) {
		newDir, file := pickerClick(cur, entries, 1)
		if newDir != filepath.Join(cur, "sub") || file != "" {
			t.Errorf("got (%q, %q)", newDir, file)
		}
	})

	Scenario(t, "GIVEN 파일 행을 클릭 WHEN 처리 THEN 폴더는 그대로고 그 파일이 선택된다", func(t *testing.T) {
		newDir, file := pickerClick(cur, entries, 2)
		if newDir != cur || file != filepath.Join(cur, "a.exe") {
			t.Errorf("got (%q, %q)", newDir, file)
		}
	})
}

func TestPickerResult(t *testing.T) {
	Scenario(t, "GIVEN 폴더 선택 모드 WHEN 확인 THEN 선택 없이도 현재 열린 폴더가 결과다", func(t *testing.T) {
		if p, ok := pickerResult(pickDir, "d:/x", ""); !ok || p != "d:/x" {
			t.Errorf("got (%q, %v)", p, ok)
		}
	})

	Scenario(t, "GIVEN 파일 선택 모드에서 파일을 골랐음 WHEN 확인 THEN 그 파일이 결과다", func(t *testing.T) {
		if p, ok := pickerResult(pickFile, "d:/x", "d:/x/rclone.exe"); !ok || p != "d:/x/rclone.exe" {
			t.Errorf("got (%q, %v)", p, ok)
		}
	})

	// 부정 케이스: 파일을 안 고르고 확인을 누르면 아무 콜백도 호출되면 안 된다.
	Scenario(t, "GIVEN 파일 선택 모드에서 아무것도 안 골랐음 WHEN 확인 THEN 결과 없음이다 (부정 케이스)", func(t *testing.T) {
		if _, ok := pickerResult(pickFile, "d:/x", ""); ok {
			t.Errorf("선택이 없으면 결과도 없어야 함")
		}
	})
}
