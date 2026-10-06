package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func newGuardTestManager(t *testing.T) (*rcloneManager, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.log")
	return &rcloneManager{log: engine.RotatingLog{Path: path, MaxLines: 200}}, path
}

func TestGuard(t *testing.T) {
	Scenario(t, "GIVEN 정상적으로 끝나는 함수 WHEN guard로 실행 THEN 그대로 실행되고 오류 로그는 없다", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		ran := false
		rm.guard("test", func() { ran = true })
		if !ran {
			t.Errorf("함수가 실행돼야 함")
		}
		if data, _ := os.ReadFile(path); strings.Contains(string(data), "패닉") {
			t.Errorf("정상 실행인데 패닉 로그가 남음: %s", data)
		}
	})

	// 핵심: 백그라운드 고루틴의 panic 하나가 프로세스 전체를 죽이면 rclone만 고아로 남는다.
	Scenario(t, "GIVEN panic하는 함수 WHEN guard로 실행 THEN 프로세스가 죽지 않고 위치와 원인이 로그에 남는다", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		rm.guard("network-monitor", func() { panic("boom") })
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "network-monitor") || !strings.Contains(string(data), "boom") {
			t.Errorf("위치와 원인이 로그에 있어야 하는데: %s", data)
		}
	})

	Scenario(t, "GIVEN 한 번 panic을 복구한 뒤 WHEN guard를 다시 호출 THEN 계속 정상 동작한다", func(t *testing.T) {
		rm, _ := newGuardTestManager(t)
		rm.guard("a", func() { panic("first") })
		ran := false
		rm.guard("b", func() { ran = true })
		if !ran {
			t.Errorf("복구 뒤에도 다음 호출이 실행돼야 함")
		}
	})

	// 부정 케이스: nil 포인터 역참조 같은 런타임 오류도 복구 대상이어야 한다.
	Scenario(t, "GIVEN nil 포인터 역참조가 발생하는 함수 WHEN guard로 실행 THEN 복구된다 (부정 케이스)", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		var m *engine.Mount
		rm.guard("nil-deref", func() { _ = m.ID })
		if data, _ := os.ReadFile(path); !strings.Contains(string(data), "nil-deref") {
			t.Errorf("런타임 오류도 로그에 남아야 함: %s", data)
		}
	})
}

func TestFirstLines(t *testing.T) {
	Scenario(t, "GIVEN 줄 수가 n보다 많은 문자열 WHEN firstLines THEN 앞 n줄만 남는다", func(t *testing.T) {
		if got := firstLines("a\nb\nc\nd", 2); got != "a\nb" {
			t.Errorf("got %q", got)
		}
	})
	Scenario(t, "GIVEN 줄 수가 n 이하인 문자열 WHEN firstLines THEN 그대로다", func(t *testing.T) {
		if got := firstLines("a\nb", 5); got != "a\nb" {
			t.Errorf("got %q", got)
		}
	})
	Scenario(t, "GIVEN n이 0 이하 WHEN firstLines THEN 빈 문자열이다 (경계 케이스)", func(t *testing.T) {
		if got := firstLines("a\nb", 0); got != "" {
			t.Errorf("got %q", got)
		}
	})
}
