package main

import "testing"

func TestIsStaleMountpointError(t *testing.T) {
	Scenario(t, "GIVEN rclone 오류 상세에 'mountpoint path already exists'가 포함됨 WHEN 판정 THEN stale 마운트포인트 오류로 인식한다", func(t *testing.T) {
		detail := "2026/08/28 09:00:00 CRITICAL: Fatal error: failed to mount FUSE fs: mountpoint path already exists: E:"
		if !isStaleMountpointError(detail) {
			t.Errorf("stale 마운트포인트 오류로 인식했어야 함")
		}
	})

	// 부정 케이스: 자격 증명 오류 등 완전히 다른, 재시도해도 절대 안 풀리는 문제.
	Scenario(t, "GIVEN 전혀 다른 종류의 rclone 오류임(자격 증명 문제 등) WHEN 판정 THEN stale 마운트포인트 오류가 아니라고 본다 (부정 케이스)", func(t *testing.T) {
		detail := "2026/08/28 09:00:00 ERROR: failed to authenticate: invalid credentials"
		if isStaleMountpointError(detail) {
			t.Errorf("무관한 오류인데 stale 마운트포인트로 잘못 인식함")
		}
	})

	Scenario(t, "GIVEN 오류 상세가 비어있음 WHEN 판정 THEN false다 (경계 케이스)", func(t *testing.T) {
		if isStaleMountpointError("") {
			t.Errorf("빈 문자열은 항상 false여야 함")
		}
	})
}

func TestNoteStaleMountRetry(t *testing.T) {
	Scenario(t, "GIVEN 첫 번째 재시도 WHEN 기록 THEN 허용된다고 나온다", func(t *testing.T) {
		rm := &rcloneManager{staleRetries: map[string]int{}}
		used, allowed := rm.noteStaleMountRetry("m1")
		if used != 1 || !allowed {
			t.Errorf("got (%d, %v), 기대값 (1, true)", used, allowed)
		}
	})

	Scenario(t, "GIVEN 최대 횟수(3)까지 재시도함 WHEN 기록 THEN 그 횟수까지는 허용된다", func(t *testing.T) {
		rm := &rcloneManager{staleRetries: map[string]int{}}
		var lastAllowed bool
		for i := 0; i < maxStaleMountRetries; i++ {
			_, lastAllowed = rm.noteStaleMountRetry("m1")
		}
		if !lastAllowed {
			t.Errorf("최대 허용 횟수(%d)까지는 허용돼야 함", maxStaleMountRetries)
		}
	})

	// 부정/경계 케이스: 최대 횟수를 넘어서면 더 이상 허용하면 안 된다 —
	// 그래야 진짜로 고장난 마운트포인트에 무한 재시도하지 않는다.
	Scenario(t, "GIVEN 최대 횟수를 넘겨서 재시도함 WHEN 기록 THEN 더 이상 허용하지 않는다 (부정 케이스)", func(t *testing.T) {
		rm := &rcloneManager{staleRetries: map[string]int{}}
		var lastAllowed bool
		for i := 0; i < maxStaleMountRetries+1; i++ {
			_, lastAllowed = rm.noteStaleMountRetry("m1")
		}
		if lastAllowed {
			t.Errorf("최대 횟수(%d)를 넘겼는데도 허용됨", maxStaleMountRetries)
		}
	})

	Scenario(t, "GIVEN 서로 다른 마운트 ID WHEN 각각 기록 THEN 서로 독립적으로 카운트된다 (경계 케이스)", func(t *testing.T) {
		rm := &rcloneManager{staleRetries: map[string]int{}}
		rm.noteStaleMountRetry("m1")
		rm.noteStaleMountRetry("m1")
		used, _ := rm.noteStaleMountRetry("m2")
		if used != 1 {
			t.Errorf("m2는 m1과 무관하게 1이어야 하는데 %d", used)
		}
	})
}

func TestClearStaleMountRetries(t *testing.T) {
	Scenario(t, "GIVEN 재시도 기록이 있던 마운트 WHEN 초기화 THEN 이후 다시 1부터 센다", func(t *testing.T) {
		rm := &rcloneManager{staleRetries: map[string]int{}}
		rm.noteStaleMountRetry("m1")
		rm.noteStaleMountRetry("m1")
		rm.clearStaleMountRetries("m1")

		used, _ := rm.noteStaleMountRetry("m1")
		if used != 1 {
			t.Errorf("초기화 후엔 1부터 다시 세야 하는데 %d", used)
		}
	})
}
