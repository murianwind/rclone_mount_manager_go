package main

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestClassifyMountExit(t *testing.T) {
	boom := errors.New("exit status 1")

	Scenario(t, "GIVEN 오류 없이 종료 WHEN 분류 THEN 정상 종료다", func(t *testing.T) {
		if got := classifyMountExit(nil, false); got != exitClean {
			t.Errorf("got %v", got)
		}
	})

	// 우리가 직접 멈춘 프로세스는 0이 아닌 코드로 끝나는 게 흔하다 — 실패가 아니다.
	Scenario(t, "GIVEN 오류 종료인데 우리가 직접 멈춘 것 WHEN 분류 THEN 요청에 의한 종료다 (실패로 보면 안 됨)", func(t *testing.T) {
		if got := classifyMountExit(boom, true); got != exitRequested {
			t.Errorf("got %v", got)
		}
	})

	Scenario(t, "GIVEN 오류 종료이고 우리가 멈춘 게 아님 WHEN 분류 THEN 실패다 (부정 케이스)", func(t *testing.T) {
		if got := classifyMountExit(boom, false); got != exitFailed {
			t.Errorf("got %v", got)
		}
	})

	Scenario(t, "GIVEN 어떤 조합이든 WHEN 분류와 shouldReportMountFailure를 비교 THEN 항상 일치한다 (기존 규칙 보존)", func(t *testing.T) {
		for _, err := range []error{nil, boom} {
			for _, stopped := range []bool{false, true} {
				want := classifyMountExit(err, stopped) == exitFailed
				if got := shouldReportMountFailure(err, stopped); got != want {
					t.Errorf("err=%v stopped=%v: shouldReport=%v, 분류 기준=%v", err, stopped, got, want)
				}
			}
		}
	})
}

func TestCollectMountExit(t *testing.T) {
	Scenario(t, "GIVEN 우리가 멈춘 자동 마운트가 45초 실행됨 WHEN 종료 정보 수집 THEN 상태·실행 시간·오류 상세를 모으고 active에서 제거한다", func(t *testing.T) {
		rm := &rcloneManager{active: map[string]*runningMount{}}
		rm.active["m1"] = &runningMount{stoppedByUs: true, autoTriggered: true, startedAt: time.Now().Add(-45 * time.Second)}
		buf := newCappedBuffer(100)
		_, _ = buf.Write([]byte("  boom \n"))

		ex := rm.collectMountExit("m1", errors.New("x"), buf)

		if !ex.stoppedByUs || !ex.autoTriggered {
			t.Errorf("플래그가 수집돼야 함: %+v", ex)
		}
		if ex.ranFor < 44*time.Second {
			t.Errorf("실행 시간이 약 45초여야 하는데 %v", ex.ranFor)
		}
		if ex.detail != "boom" {
			t.Errorf("오류 상세는 공백이 정리된 'boom'이어야 하는데 %q", ex.detail)
		}
		if _, still := rm.active["m1"]; still {
			t.Errorf("수집 뒤엔 active에서 지워져야 함")
		}
	})

	// 부정 케이스: 이미 목록에서 사라진 마운트여도(예: 중복 종료 처리) panic이 나면 안 된다.
	Scenario(t, "GIVEN active에 없는 마운트 WHEN 종료 정보 수집 THEN panic 없이 기본값을 돌려준다 (부정 케이스)", func(t *testing.T) {
		rm := &rcloneManager{active: map[string]*runningMount{}}
		ex := rm.collectMountExit("ghost", nil, newCappedBuffer(10))
		if ex.stoppedByUs || ex.autoTriggered || ex.ranFor != 0 {
			t.Errorf("기본값이어야 하는데 %+v", ex)
		}
	})
}

func TestRecordAutoMountOutcome(t *testing.T) {
	Scenario(t, "GIVEN 자동 마운트가 온라인에서 실패 WHEN 결과 기록 THEN 백오프 대기 상태가 된다", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.recordAutoMountOutcome("m1", exitFailed, mountExit{autoTriggered: true})
		if !rm.backoff.shouldSkip("m1", time.Now()) {
			t.Errorf("실패 직후엔 건너뛰어야 함")
		}
	})

	// 오프라인 중 실패는 '곧 해결될' 정상 현상 — 이걸 세면 재연결 직후에도 한참 안 붙는다.
	Scenario(t, "GIVEN 자동 마운트가 오프라인 중 실패 WHEN 결과 기록 THEN 백오프에 세지 않는다 (재연결 즉시 재시도 보장)", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.setOfflineSince(time.Now().Add(-time.Minute))
		rm.recordAutoMountOutcome("m1", exitFailed, mountExit{autoTriggered: true})
		if rm.backoff.shouldSkip("m1", time.Now()) {
			t.Errorf("오프라인 중 실패는 백오프에 세면 안 됨")
		}
	})

	Scenario(t, "GIVEN 자동 마운트가 실패 중이었는데 사용자가 수동으로 마운트했다가 종료됨 WHEN 결과 기록 THEN 그 마운트의 백오프가 초기화된다", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.recordAutoMountOutcome("m1", exitFailed, mountExit{autoTriggered: true})
		rm.recordAutoMountOutcome("m1", exitClean, mountExit{autoTriggered: false})
		if rm.backoff.shouldSkip("m1", time.Now()) {
			t.Errorf("수동 마운트 종료 뒤엔 초기화돼야 함")
		}
	})

	Scenario(t, "GIVEN 자동 마운트가 요청에 의해 정상 종료 WHEN 결과 기록 THEN 초기화된다", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.recordAutoMountOutcome("m1", exitFailed, mountExit{autoTriggered: true})
		rm.recordAutoMountOutcome("m1", exitRequested, mountExit{autoTriggered: true, stoppedByUs: true})
		if rm.backoff.shouldSkip("m1", time.Now()) {
			t.Errorf("요청에 의한 종료 뒤엔 초기화돼야 함")
		}
	})

	Scenario(t, "GIVEN 수동 마운트가 실패 WHEN 결과 기록 THEN 백오프에 세지 않는다 (사용자가 직접 다시 누르는 경우는 막지 않음)", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.recordAutoMountOutcome("m1", exitFailed, mountExit{autoTriggered: false})
		if rm.backoff.shouldSkip("m1", time.Now()) {
			t.Errorf("수동 실패는 백오프 대상이 아님")
		}
	})
}

// ── 리팩토링(waitForMountExit 분해) 전후 동작이 같음을 고정하는 특성화 테스트 ──

func TestResolveFailureReport(t *testing.T) {
	boom := errors.New("exit status 1")
	m := engine.Mount{ID: "m1", Remote: "PLEX", RemotePath: "KODI"}

	Scenario(t, "GIVEN 수동 마운트가 오류로 종료 WHEN 알림 여부 결정 THEN 알린다", func(t *testing.T) {
		rm, _ := newGuardTestManager(t)
		got := rm.resolveFailureReport(m, exitFailed, mountExit{err: boom})
		if !got {
			t.Errorf("수동 마운트 실패는 알려야 함")
		}
	})

	Scenario(t, "GIVEN 자동 마운트가 온라인에서 오류로 종료 WHEN 알림 여부 결정 THEN 알리고 백오프도 기록한다", func(t *testing.T) {
		rm, _ := newGuardTestManager(t)
		got := rm.resolveFailureReport(m, exitFailed, mountExit{err: boom, autoTriggered: true})
		if !got {
			t.Errorf("온라인 자동 마운트 실패는 알려야 함")
		}
		if !rm.backoff.shouldSkip("m1", time.Now()) {
			t.Errorf("실패가 백오프에 기록돼야 함")
		}
	})

	Scenario(t, "GIVEN 오프라인 유예 기간 안의 자동 마운트 실패 WHEN 알림 여부 결정 THEN 알리지 않는다 (기존 규칙 보존)", func(t *testing.T) {
		rm, _ := newGuardTestManager(t)
		rm.setOfflineSince(time.Now().Add(-time.Minute))
		if rm.resolveFailureReport(m, exitFailed, mountExit{err: boom, autoTriggered: true}) {
			t.Errorf("유예 기간 안에서는 알림을 생략해야 함")
		}
	})

	// 부정 케이스: 유예 기간(5분)을 넘겨서도 계속 실패하면 더는 숨기면 안 된다.
	Scenario(t, "GIVEN 유예 기간을 넘겨서도 계속 오프라인인 자동 마운트 실패 WHEN 알림 여부 결정 THEN 알린다 (부정 케이스)", func(t *testing.T) {
		rm, _ := newGuardTestManager(t)
		rm.setOfflineSince(time.Now().Add(-2 * autoMountFailureGrace))
		if !rm.resolveFailureReport(m, exitFailed, mountExit{err: boom, autoTriggered: true}) {
			t.Errorf("유예 기간이 지났으면 알려야 함")
		}
	})

	Scenario(t, "GIVEN 요청에 의한 종료나 정상 종료 WHEN 알림 여부 결정 THEN 알리지 않는다", func(t *testing.T) {
		rm, _ := newGuardTestManager(t)
		if rm.resolveFailureReport(m, exitRequested, mountExit{err: boom, stoppedByUs: true}) {
			t.Errorf("요청에 의한 종료는 알리면 안 됨")
		}
		if rm.resolveFailureReport(m, exitClean, mountExit{}) {
			t.Errorf("정상 종료는 알리면 안 됨")
		}
	})
}

func TestLogMountExit(t *testing.T) {
	boom := errors.New("exit status 0x40010004")
	m := engine.Mount{ID: "m1", Remote: "PLEX", RemotePath: "KODI"}
	read := func(path string) string { b, _ := os.ReadFile(path); return string(b) }

	Scenario(t, "GIVEN 요청에 의한 종료 WHEN 로그 THEN WARN이 아니라 INFO로 남는다 (0x40010004 같은 예상된 종료 코드로 로그가 오염되지 않게)", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		rm.logMountExit(m, exitRequested, mountExit{err: boom, stoppedByUs: true})
		got := read(path)
		if !strings.Contains(got, "INFO") || strings.Contains(got, "WARN") || !strings.Contains(got, "요청에 의한 종료") {
			t.Errorf("INFO '요청에 의한 종료'여야 하는데: %s", got)
		}
	})

	Scenario(t, "GIVEN 실패 종료와 rclone 오류 상세 WHEN 로그 THEN WARN과 ERROR(상세) 두 줄이 남는다", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		rm.logMountExit(m, exitFailed, mountExit{err: boom, detail: "CRITICAL: bad credentials"})
		got := read(path)
		if !strings.Contains(got, "WARN") || !strings.Contains(got, "ERROR") || !strings.Contains(got, "bad credentials") {
			t.Errorf("WARN+ERROR 상세가 있어야 하는데: %s", got)
		}
	})

	Scenario(t, "GIVEN 실패 종료인데 오류 상세가 비어있음 WHEN 로그 THEN ERROR 상세 줄은 남기지 않는다 (경계 케이스)", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		rm.logMountExit(m, exitFailed, mountExit{err: boom})
		if strings.Contains(read(path), "ERROR") {
			t.Errorf("상세가 없으면 ERROR 줄이 없어야 함: %s", read(path))
		}
	})

	Scenario(t, "GIVEN 정상 종료 WHEN 로그 THEN INFO 한 줄이다", func(t *testing.T) {
		rm, path := newGuardTestManager(t)
		rm.logMountExit(m, exitClean, mountExit{})
		got := read(path)
		if !strings.Contains(got, "INFO") || strings.Contains(got, "WARN") {
			t.Errorf("INFO여야 하는데: %s", got)
		}
	})
}
