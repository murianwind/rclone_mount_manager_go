package engine

import (
	"testing"
	"time"
)

// ── ValidateSchedule ──

func TestValidateSchedule(t *testing.T) {
	Scenario(t, "GIVEN 종료 시각이 시작 시각보다 나중이고 요일이 하나 이상 선택됨 WHEN 검증 THEN 문제 없음(빈 문자열)을 반환한다", func(t *testing.T) {
		s := Schedule{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1, 3, 5}}
		if msg := ValidateSchedule(s); msg != "" {
			t.Errorf("문제 없어야 하는데 %q", msg)
		}
	})

	// 부정 케이스: 종료 시각이 시작 시각과 같음 — 자정을 안 넘기기로 했으니
	// 빈 구간은 무의미하다.
	Scenario(t, "GIVEN 종료 시각이 시작 시각과 같음 WHEN 검증 THEN 오류 메시지를 반환한다 (부정 케이스)", func(t *testing.T) {
		s := Schedule{StartHour: 9, StartMinute: 0, EndHour: 9, EndMinute: 0, Days: []int{1}}
		if msg := ValidateSchedule(s); msg == "" {
			t.Errorf("종료==시작인데 문제 없다고 나옴")
		}
	})

	Scenario(t, "GIVEN 종료 시각이 시작 시각보다 이전임 WHEN 검증 THEN 오류 메시지를 반환한다 (부정 케이스)", func(t *testing.T) {
		s := Schedule{StartHour: 12, StartMinute: 0, EndHour: 9, EndMinute: 0, Days: []int{1}}
		if msg := ValidateSchedule(s); msg == "" {
			t.Errorf("종료<시작인데 문제 없다고 나옴")
		}
	})

	Scenario(t, "GIVEN 요일이 하나도 선택 안 됨 WHEN 검증 THEN 오류 메시지를 반환한다 (부정 케이스)", func(t *testing.T) {
		s := Schedule{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{}}
		if msg := ValidateSchedule(s); msg == "" {
			t.Errorf("요일 없는데 문제 없다고 나옴")
		}
	})

	// 경계 케이스: 분 단위 차이만 있어도(9:00~9:01) 유효해야 한다.
	Scenario(t, "GIVEN 시작과 종료가 1분 차이임 WHEN 검증 THEN 문제 없음을 반환한다 (경계 케이스)", func(t *testing.T) {
		s := Schedule{StartHour: 9, StartMinute: 0, EndHour: 9, EndMinute: 1, Days: []int{1}}
		if msg := ValidateSchedule(s); msg != "" {
			t.Errorf("1분 차이는 유효해야 하는데 %q", msg)
		}
	})

	// 부정 케이스: 화면 입력값이 잘못 넘어온 방어적 상황(시/분 범위 밖).
	Scenario(t, "GIVEN 시가 24 이상이거나 분이 60 이상임 WHEN 검증 THEN 오류 메시지를 반환한다 (부정 케이스)", func(t *testing.T) {
		s := Schedule{StartHour: 24, StartMinute: 0, EndHour: 25, EndMinute: 0, Days: []int{1}}
		if msg := ValidateSchedule(s); msg == "" {
			t.Errorf("시가 범위 밖인데 문제 없다고 나옴")
		}
	})
}

// ── DaysOverlap ──

func TestDaysOverlap(t *testing.T) {
	Scenario(t, "GIVEN 두 요일 목록이 하나라도 겹침 WHEN 판정 THEN true를 반환한다", func(t *testing.T) {
		if !DaysOverlap([]int{1, 3, 5}, []int{5, 6}) {
			t.Errorf("5(금)가 겹치는데 false가 나옴")
		}
	})

	Scenario(t, "GIVEN 두 요일 목록이 전혀 안 겹침 WHEN 판정 THEN false를 반환한다 (부정 케이스)", func(t *testing.T) {
		if DaysOverlap([]int{1, 2, 3}, []int{4, 5, 6}) {
			t.Errorf("안 겹치는데 true가 나옴")
		}
	})

	Scenario(t, "GIVEN 둘 중 하나가 빈 목록임 WHEN 판정 THEN false를 반환한다 (경계 케이스)", func(t *testing.T) {
		if DaysOverlap([]int{}, []int{1, 2, 3}) {
			t.Errorf("빈 목록인데 true가 나옴")
		}
	})
}

// ── SchedulesOverlap ──

func TestSchedulesOverlap(t *testing.T) {
	mon9to12 := Schedule{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1}}

	Scenario(t, "GIVEN 같은 요일에 시간대가 겹침 WHEN 판정 THEN true를 반환한다", func(t *testing.T) {
		other := Schedule{StartHour: 10, StartMinute: 0, EndHour: 14, EndMinute: 0, Days: []int{1}}
		if !SchedulesOverlap(mon9to12, other) {
			t.Errorf("9~12시와 10~14시는 겹쳐야 함")
		}
	})

	// 부정 케이스: 시간대가 정확히 맞닿기만 함 — 겹치는 게 아니라 이어붙인
	// 것이므로, 뒤이어 등록하는 걸 막으면 안 된다(반열린구간 [시작,종료)).
	Scenario(t, "GIVEN 같은 요일에 시간대가 정확히 맞닿기만 함(12시에 끝나고 12시에 시작) WHEN 판정 THEN 겹친다고 보지 않는다 (부정 케이스)", func(t *testing.T) {
		other := Schedule{StartHour: 12, StartMinute: 0, EndHour: 15, EndMinute: 0, Days: []int{1}}
		if SchedulesOverlap(mon9to12, other) {
			t.Errorf("맞닿기만 한 건 겹치는 게 아닌데 true가 나옴 — 뒤이어 등록이 막히면 안 됨")
		}
	})

	Scenario(t, "GIVEN 같은 요일이지만 시간대가 전혀 안 겹침 WHEN 판정 THEN false를 반환한다 (부정 케이스)", func(t *testing.T) {
		other := Schedule{StartHour: 14, StartMinute: 0, EndHour: 16, EndMinute: 0, Days: []int{1}}
		if SchedulesOverlap(mon9to12, other) {
			t.Errorf("시간대가 안 겹치는데 true가 나옴")
		}
	})

	Scenario(t, "GIVEN 시간대는 겹치지만 요일이 다름 WHEN 판정 THEN false를 반환한다 (경계 케이스)", func(t *testing.T) {
		tue9to12 := Schedule{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{2}}
		if SchedulesOverlap(mon9to12, tue9to12) {
			t.Errorf("월요일과 화요일은 요일이 다른데 true가 나옴")
		}
	})

	Scenario(t, "GIVEN 요일이 일부만 겹치고 그 요일의 시간대도 겹침 WHEN 판정 THEN true를 반환한다", func(t *testing.T) {
		monWed := Schedule{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1, 3}}
		wedFri := Schedule{StartHour: 10, StartMinute: 0, EndHour: 13, EndMinute: 0, Days: []int{3, 5}}
		if !SchedulesOverlap(monWed, wedFri) {
			t.Errorf("수요일이 겹치고 시간대도 겹치는데 false가 나옴")
		}
	})
}

// ── IsWithinSchedule ──

func TestIsWithinSchedule(t *testing.T) {
	// 2026-08-31은 월요일이다.
	mon9to12 := Schedule{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1}}

	Scenario(t, "GIVEN 지금이 스케줄 구간 한가운데임 WHEN 판정 THEN true를 반환한다", func(t *testing.T) {
		now := time.Date(2026, 8, 31, 10, 30, 0, 0, time.UTC)
		if !IsWithinSchedule([]Schedule{mon9to12}, now) {
			t.Errorf("구간 안인데 false가 나옴")
		}
	})

	Scenario(t, "GIVEN 지금이 시작 시각과 정확히 같음 WHEN 판정 THEN true를 반환한다 (경계 케이스, 시작은 포함)", func(t *testing.T) {
		now := time.Date(2026, 8, 31, 9, 0, 0, 0, time.UTC)
		if !IsWithinSchedule([]Schedule{mon9to12}, now) {
			t.Errorf("시작 시각 정각인데 false가 나옴")
		}
	})

	Scenario(t, "GIVEN 지금이 종료 시각과 정확히 같음 WHEN 판정 THEN false를 반환한다 (경계 케이스, 종료는 제외)", func(t *testing.T) {
		now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
		if IsWithinSchedule([]Schedule{mon9to12}, now) {
			t.Errorf("종료 시각 정각인데 true가 나옴 — 종료는 제외돼야 함")
		}
	})

	// 부정 케이스: 요일 자체가 다름.
	Scenario(t, "GIVEN 지금이 스케줄에 없는 요일임 WHEN 판정 THEN false를 반환한다 (부정 케이스)", func(t *testing.T) {
		tue := time.Date(2026, 9, 1, 10, 30, 0, 0, time.UTC) // 화요일
		if IsWithinSchedule([]Schedule{mon9to12}, tue) {
			t.Errorf("화요일인데 true가 나옴")
		}
	})

	Scenario(t, "GIVEN 스케줄 목록이 비어있음 WHEN 판정 THEN false를 반환한다 (경계 케이스)", func(t *testing.T) {
		now := time.Date(2026, 8, 31, 10, 30, 0, 0, time.UTC)
		if IsWithinSchedule(nil, now) {
			t.Errorf("스케줄이 없는데 true가 나옴")
		}
	})

	Scenario(t, "GIVEN 여러 스케줄 중 하나에만 해당함 WHEN 판정 THEN true를 반환한다", func(t *testing.T) {
		evening := Schedule{StartHour: 18, StartMinute: 0, EndHour: 22, EndMinute: 0, Days: []int{1}}
		now := time.Date(2026, 8, 31, 19, 0, 0, 0, time.UTC)
		if !IsWithinSchedule([]Schedule{mon9to12, evening}, now) {
			t.Errorf("두 번째 스케줄 구간인데 false가 나옴")
		}
	})
}

// ── DecideScheduleAction ──

func TestDecideScheduleAction(t *testing.T) {
	schedules := []Schedule{{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1}}}
	within := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)  // 월요일 10시 — 구간 안
	outside := time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC) // 월요일 15시 — 구간 밖

	Scenario(t, "GIVEN 구간 안이고 아직 안 켜져 있고 건너뛰기도 아님 WHEN 판정 THEN 마운트하라고 나온다", func(t *testing.T) {
		if got := DecideScheduleAction(schedules, within, false, false); got != ScheduleActionMount {
			t.Errorf("got %v, 기대값 mount", got)
		}
	})

	Scenario(t, "GIVEN 구간 안이고 이미 켜져 있음 WHEN 판정 THEN 아무것도 안 하라고 나온다", func(t *testing.T) {
		if got := DecideScheduleAction(schedules, within, true, false); got != ScheduleActionNone {
			t.Errorf("got %v, 기대값 none", got)
		}
	})

	// 핵심 시나리오: 구간 안에서 사용자가 수동으로 해제해서 skip=true가 된 상태.
	Scenario(t, "GIVEN 구간 안인데 사용자가 수동 해제해서 건너뛰기 상태임 WHEN 판정 THEN 아무것도 안 하라고 나온다 (이번 구간은 건너뜀)", func(t *testing.T) {
		if got := DecideScheduleAction(schedules, within, false, true); got != ScheduleActionNone {
			t.Errorf("got %v, 기대값 none (건너뛰기 유지)", got)
		}
	})

	Scenario(t, "GIVEN 구간 밖인데 켜져 있음 WHEN 판정 THEN 해제하라고 나온다 (건너뛰기 여부와 무관하게 스케줄이 우선)", func(t *testing.T) {
		if got := DecideScheduleAction(schedules, outside, true, false); got != ScheduleActionUnmount {
			t.Errorf("got %v, 기대값 unmount", got)
		}
		if got := DecideScheduleAction(schedules, outside, true, true); got != ScheduleActionUnmount {
			t.Errorf("건너뛰기 상태여도 구간 밖에서 켜져 있으면 해제해야 하는데 got %v", got)
		}
	})

	Scenario(t, "GIVEN 구간 밖이고 꺼져 있는데 건너뛰기 상태가 남아있음 WHEN 판정 THEN 건너뛰기를 해제하라고 나온다 (다음 구간을 위해)", func(t *testing.T) {
		if got := DecideScheduleAction(schedules, outside, false, true); got != ScheduleActionResetSkip {
			t.Errorf("got %v, 기대값 resetSkip", got)
		}
	})

	Scenario(t, "GIVEN 구간 밖이고 꺼져 있고 건너뛰기도 아님 WHEN 판정 THEN 아무것도 안 하라고 나온다 (부정 케이스, 이미 정상 상태)", func(t *testing.T) {
		if got := DecideScheduleAction(schedules, outside, false, false); got != ScheduleActionNone {
			t.Errorf("got %v, 기대값 none", got)
		}
	})
}
