package main

import (
	"sync"
	"testing"
	"time"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestAutoMountBackoffDelay(t *testing.T) {
	Scenario(t, "GIVEN 실패 0회 이하 WHEN 대기 시간 계산 THEN 0이다", func(t *testing.T) {
		if autoMountBackoffDelay(0) != 0 || autoMountBackoffDelay(-3) != 0 {
			t.Errorf("실패가 없으면 대기도 없어야 함")
		}
	})

	Scenario(t, "GIVEN 연속 실패 1,2,3회 WHEN 대기 시간 계산 THEN 기본값에서 두 배씩 늘어난다", func(t *testing.T) {
		if got := autoMountBackoffDelay(1); got != autoMountBackoffBase {
			t.Errorf("1회: got %v", got)
		}
		if got := autoMountBackoffDelay(2); got != 2*autoMountBackoffBase {
			t.Errorf("2회: got %v", got)
		}
		if got := autoMountBackoffDelay(3); got != 4*autoMountBackoffBase {
			t.Errorf("3회: got %v", got)
		}
	})

	// 경계 케이스: 아주 많이 실패해도 상한을 넘거나 오버플로로 음수가 되면 안 된다.
	Scenario(t, "GIVEN 실패가 매우 많음 WHEN 대기 시간 계산 THEN 상한에서 멈추고 줄어들지 않는다 (경계 케이스)", func(t *testing.T) {
		prev := time.Duration(0)
		for n := 1; n <= 200; n++ {
			d := autoMountBackoffDelay(n)
			if d < prev {
				t.Fatalf("%d회에서 대기 시간이 줄어듦: %v < %v", n, d, prev)
			}
			if d > autoMountBackoffMax {
				t.Fatalf("%d회에서 상한 초과: %v", n, d)
			}
			prev = d
		}
		if autoMountBackoffDelay(200) != autoMountBackoffMax {
			t.Errorf("충분히 많이 실패하면 상한이어야 함")
		}
	})
}

func TestBackoffTracker(t *testing.T) {
	t0 := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	Scenario(t, "GIVEN 기록이 없는 마운트 WHEN shouldSkip THEN false다", func(t *testing.T) {
		var tr backoffTracker
		if tr.shouldSkip("m1", t0) {
			t.Errorf("기록 없으면 건너뛰면 안 됨")
		}
	})

	Scenario(t, "GIVEN 첫 실패를 기록함 WHEN 대기 시간 안/밖에서 shouldSkip THEN 안에서는 true, 지나면 false", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("m1", true, 0, t0)
		if !tr.shouldSkip("m1", t0.Add(autoMountBackoffBase/2)) {
			t.Errorf("대기 시간 안에서는 건너뛰어야 함")
		}
		if tr.shouldSkip("m1", t0.Add(autoMountBackoffBase)) {
			t.Errorf("대기 시간이 지나면 다시 시도해야 함")
		}
	})

	Scenario(t, "GIVEN 연속 두 번 실패 WHEN 두 번째 대기 시간 확인 THEN 첫 번째보다 길다", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("m1", true, 0, t0)
		tr.recordExit("m1", true, 0, t0)
		if !tr.shouldSkip("m1", t0.Add(autoMountBackoffBase+time.Second)) {
			t.Errorf("두 번째 실패의 대기(2배)가 아직 안 지났어야 함")
		}
		if tr.shouldSkip("m1", t0.Add(2*autoMountBackoffBase)) {
			t.Errorf("2배 대기가 지나면 다시 시도해야 함")
		}
	})

	Scenario(t, "GIVEN 실패 후 정상 종료가 기록됨 WHEN shouldSkip THEN 초기화되어 false다", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("m1", true, 0, t0)
		tr.recordExit("m1", false, 0, t0)
		if tr.shouldSkip("m1", t0) {
			t.Errorf("정상 종료 뒤엔 초기화돼야 함")
		}
	})

	// 마운트가 한참 정상 동작하다가 실패한 건 '새로운 사건'이다 — 이전 실패 누적을 이어가면 안 된다.
	Scenario(t, "GIVEN 연속 실패 중이었는데 안정적으로 오래 실행된 뒤 실패 WHEN 기록 THEN 1회 실패부터 다시 센다", func(t *testing.T) {
		var tr backoffTracker
		for i := 0; i < 5; i++ {
			tr.recordExit("m1", true, 0, t0)
		}
		tr.recordExit("m1", true, autoMountStableRun+time.Second, t0)
		if tr.shouldSkip("m1", t0.Add(autoMountBackoffBase)) {
			t.Errorf("안정 실행 뒤 첫 실패라면 기본 대기(%v)만 적용돼야 함", autoMountBackoffBase)
		}
	})

	Scenario(t, "GIVEN 여러 마운트가 실패 중 WHEN resetAll THEN 전부 초기화된다 (네트워크 재연결 시 즉시 재시도 보장)", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("m1", true, 0, t0)
		tr.recordExit("m2", true, 0, t0)
		tr.resetAll()
		if tr.shouldSkip("m1", t0) || tr.shouldSkip("m2", t0) {
			t.Errorf("resetAll 뒤엔 모두 즉시 시도 가능해야 함")
		}
	})

	Scenario(t, "GIVEN 마운트 하나만 실패 WHEN 다른 마운트 shouldSkip THEN 영향이 없다", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("m1", true, 0, t0)
		if tr.shouldSkip("m2", t0) {
			t.Errorf("다른 마운트는 영향받으면 안 됨")
		}
	})

	Scenario(t, "GIVEN 여러 고루틴이 동시에 기록하고 조회 WHEN -race THEN 데이터 레이스가 없다", func(t *testing.T) {
		var tr backoffTracker
		var wg sync.WaitGroup
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 300; i++ {
					tr.recordExit("m", i%3 != 0, 0, t0)
					_ = tr.shouldSkip("m", t0)
					if i%100 == 0 {
						tr.resetAll()
					}
				}
			}()
		}
		wg.Wait()
	})
}

func TestMountsDueForAutoMount(t *testing.T) {
	t0 := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	mounts := []engine.Mount{
		{ID: "a", AutoMount: true},
		{ID: "b", AutoMount: true},
		{ID: "c", AutoMount: false},
	}

	Scenario(t, "GIVEN 자동 마운트 2개 중 b가 백오프 대기 중 WHEN 대상 선별 THEN a만 나온다", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("b", true, 0, t0)
		got := mountsDueForAutoMount(mounts, &tr, t0)
		if len(got) != 1 || got[0].ID != "a" {
			t.Errorf("a만 나와야 하는데 %+v", got)
		}
	})

	// 부정 케이스: 자동 마운트가 꺼진 마운트는 백오프와 무관하게 절대 대상이 아니다.
	Scenario(t, "GIVEN 자동 마운트가 꺼진 마운트 WHEN 대상 선별 THEN 포함되지 않는다 (부정 케이스)", func(t *testing.T) {
		var tr backoffTracker
		for _, m := range mountsDueForAutoMount(mounts, &tr, t0) {
			if m.ID == "c" {
				t.Errorf("자동 마운트가 꺼진 c가 포함됨")
			}
		}
	})

	Scenario(t, "GIVEN 대기 시간이 지남 WHEN 대상 선별 THEN 다시 포함된다", func(t *testing.T) {
		var tr backoffTracker
		tr.recordExit("b", true, 0, t0)
		got := mountsDueForAutoMount(mounts, &tr, t0.Add(autoMountBackoffBase))
		if len(got) != 2 {
			t.Errorf("a,b 둘 다 나와야 하는데 %+v", got)
		}
	})
}
