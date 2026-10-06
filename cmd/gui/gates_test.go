package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedGate(t *testing.T) {
	t0 := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	Scenario(t, "GIVEN 아무도 잡지 않은 키 WHEN 획득 THEN 성공한다", func(t *testing.T) {
		var g keyedGate
		if !g.tryAcquire("k") {
			t.Errorf("처음 획득은 성공해야 함")
		}
	})

	// 부정 케이스: 같은 알림이 이미 떠 있는데 또 뜨면 창이 계속 쌓인다.
	Scenario(t, "GIVEN 이미 잡힌 키 WHEN 다시 획득 THEN 실패한다 (부정 케이스)", func(t *testing.T) {
		var g keyedGate
		g.tryAcquire("k")
		if g.tryAcquire("k") {
			t.Errorf("이미 잡힌 키인데 또 획득됨")
		}
	})

	Scenario(t, "GIVEN 서로 다른 키 WHEN 각각 획득 THEN 서로 영향이 없다", func(t *testing.T) {
		var g keyedGate
		g.tryAcquire("a")
		if !g.tryAcquire("b") {
			t.Errorf("다른 키는 독립적이어야 함")
		}
	})

	Scenario(t, "GIVEN 잡았다가 해제한 키 WHEN 다시 획득 THEN 성공한다", func(t *testing.T) {
		var g keyedGate
		g.tryAcquire("k")
		g.release("k")
		if !g.tryAcquire("k") {
			t.Errorf("해제 후엔 다시 획득돼야 함")
		}
	})

	// 안전장치: 닫힘 콜백이 어떤 이유로 오지 않아도 영구히 잠기면 안 된다.
	Scenario(t, "GIVEN 해제 없이 유효 시간(maxHold)이 지남 WHEN 획득 THEN 성공한다 (영구 잠금 방지)", func(t *testing.T) {
		g := keyedGate{maxHold: time.Minute}
		g.tryAcquireAt("k", t0)
		if g.tryAcquireAt("k", t0.Add(30*time.Second)) {
			t.Errorf("유효 시간 안에는 계속 잡혀 있어야 함")
		}
		if !g.tryAcquireAt("k", t0.Add(2*time.Minute)) {
			t.Errorf("유효 시간이 지났으면 다시 획득돼야 함")
		}
	})

	Scenario(t, "GIVEN maxHold가 0 WHEN 오래 지나도 THEN 만료 없이 계속 잡혀 있다 (경계 케이스)", func(t *testing.T) {
		var g keyedGate
		g.tryAcquireAt("k", t0)
		if g.tryAcquireAt("k", t0.Add(24*time.Hour)) {
			t.Errorf("maxHold=0이면 만료되면 안 됨")
		}
	})

	Scenario(t, "GIVEN 획득한 적 없는 키 WHEN release THEN panic 없이 무시한다 (부정 케이스)", func(t *testing.T) {
		var g keyedGate
		g.release("never-acquired")
	})

	Scenario(t, "GIVEN 100개 고루틴이 동시에 같은 키를 획득 시도 WHEN 실행 THEN 정확히 하나만 성공한다 (-race)", func(t *testing.T) {
		var g keyedGate
		var wins atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if g.tryAcquire("k") {
					wins.Add(1)
				}
			}()
		}
		wg.Wait()
		if wins.Load() != 1 {
			t.Errorf("성공이 %d번 — 정확히 1번이어야 함", wins.Load())
		}
	})
}
