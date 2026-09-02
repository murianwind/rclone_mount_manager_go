package main

import (
	"os"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestValidateMount(t *testing.T) {
	existing := []engine.Mount{
		{ID: "a", Remote: "PLEX", RemotePath: "KODI", Drive: "E:"},
		{ID: "b", Remote: "gds", RemotePath: "VIDEO", Drive: "G:"},
	}

	Scenario(t, "GIVEN 다른 마운트와 드라이브 문자가 겹침 WHEN 검증 THEN 오류 메시지를 반환한다", func(t *testing.T) {
		m := engine.Mount{ID: "new", Remote: "onedrive", RemotePath: "x", Drive: "E:"}
		if got := validateMount(m, existing); got == "" {
			t.Errorf("드라이브 중복인데 오류가 없음")
		}
	})

	Scenario(t, "GIVEN 다른 마운트와 리모트+서브경로가 완전히 같음 WHEN 검증 THEN 오류 메시지를 반환한다", func(t *testing.T) {
		m := engine.Mount{ID: "new", Remote: "PLEX", RemotePath: "KODI", Drive: "Z:"}
		if got := validateMount(m, existing); got == "" {
			t.Errorf("리모트/경로 중복인데 오류가 없음")
		}
	})

	Scenario(t, "GIVEN 겹치는 게 전혀 없음 WHEN 검증 THEN 빈 문자열(문제 없음)을 반환한다", func(t *testing.T) {
		m := engine.Mount{ID: "new", Remote: "onedrive", RemotePath: "x", Drive: "Z:"}
		if got := validateMount(m, existing); got != "" {
			t.Errorf("문제 없어야 하는데 %q", got)
		}
	})

	// 부정/경계 케이스: 자기 자신(편집 중인 마운트)은 비교 대상에서 제외돼야 함.
	Scenario(t, "GIVEN 편집 중인 마운트가 자기 자신의 예전 값과 비교됨 WHEN 검증 THEN 자기 자신은 충돌로 안 잡는다 (경계 케이스)", func(t *testing.T) {
		m := engine.Mount{ID: "a", Remote: "PLEX", RemotePath: "KODI", Drive: "E:"} // 기존 "a"를 그대로 저장(편집 후 변경 없음)
		if got := validateMount(m, existing); got != "" {
			t.Errorf("자기 자신과는 충돌로 잡히면 안 되는데 %q", got)
		}
	})

	// 부정 케이스: 드라이브 문자가 비어있으면(자동 배정) 다른 빈 드라이브와도 충돌로 안 잡음.
	Scenario(t, "GIVEN 드라이브 문자가 둘 다 비어있음(자동) WHEN 검증 THEN 드라이브 중복으로 안 잡는다 (부정 케이스)", func(t *testing.T) {
		existingAuto := []engine.Mount{{ID: "a", Remote: "PLEX", RemotePath: "KODI", Drive: ""}}
		m := engine.Mount{ID: "new", Remote: "gds", RemotePath: "x", Drive: ""}
		if got := validateMount(m, existingAuto); got != "" {
			t.Errorf("빈 드라이브끼리는 충돌 아니어야 하는데 %q", got)
		}
	})
}

func TestValidateMountLocation(t *testing.T) {
	Scenario(t, "GIVEN 마운트 위치가 비어있음(자동) WHEN 검증 THEN 문제 없음을 반환한다", func(t *testing.T) {
		if msg := validateMountLocation(""); msg != "" {
			t.Errorf("빈 값은 문제 없어야 하는데 %q", msg)
		}
	})

	Scenario(t, "GIVEN 드라이브 문자뿐임(경로 구분자 없음) WHEN 검증 THEN 문제 없음을 반환한다", func(t *testing.T) {
		if msg := validateMountLocation("Z:"); msg != "" {
			t.Errorf("드라이브 문자는 문제 없어야 하는데 %q", msg)
		}
	})

	Scenario(t, "GIVEN 폴더 경로인데 아직 존재하지 않음 WHEN 검증 THEN 문제 없음을 반환한다 (rclone이 마운트 시점에 직접 만듦)", func(t *testing.T) {
		dir := t.TempDir()
		notYetCreated := dir + string(os.PathSeparator) + "새로만들어질폴더"
		if msg := validateMountLocation(notYetCreated); msg != "" {
			t.Errorf("아직 없는 폴더는 문제 없어야 하는데 %q", msg)
		}
	})

	// 회귀 테스트: 실제로 겪은 문제 — 사용자가 폴더를 미리 만들어두면
	// rclone이 "mountpoint path already exists"로 항상 실패했다.
	Scenario(t, "GIVEN 폴더 경로가 이미 존재함(미리 만들어둔 빈 폴더) WHEN 검증 THEN 오류 메시지를 반환한다 (회귀 테스트)", func(t *testing.T) {
		dir := t.TempDir()
		if msg := validateMountLocation(dir); msg == "" {
			t.Errorf("이미 존재하는 폴더인데 문제 없다고 나옴")
		}
	})

	Scenario(t, "GIVEN 그 경로에 이미 파일이 있음(폴더가 아님) WHEN 검증 THEN 오류 메시지를 반환한다 (부정 케이스)", func(t *testing.T) {
		dir := t.TempDir()
		filePath := dir + string(os.PathSeparator) + "이미있는파일"
		if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if msg := validateMountLocation(filePath); msg == "" {
			t.Errorf("이미 파일이 있는데 문제 없다고 나옴")
		}
	})
}
