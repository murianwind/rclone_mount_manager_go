package main

import (
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

func TestValidateScheduleList(t *testing.T) {
	Scenario(t, "GIVEN 서로 겹치지 않는 여러 항목 WHEN 검증 THEN 문제 없음(빈 문자열)을 반환한다", func(t *testing.T) {
		list := []engine.Schedule{
			{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1, 2, 3, 4, 5}},
			{StartHour: 18, StartMinute: 0, EndHour: 22, EndMinute: 0, Days: []int{0, 6}},
		}
		if msg := validateScheduleList(list); msg != "" {
			t.Errorf("문제 없어야 하는데 %q", msg)
		}
	})

	Scenario(t, "GIVEN 항목 하나가 그 자체로 잘못됨(종료<=시작) WHEN 검증 THEN 몇 번째 항목이 문제인지 알려준다", func(t *testing.T) {
		list := []engine.Schedule{
			{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1}},
			{StartHour: 15, StartMinute: 0, EndHour: 10, EndMinute: 0, Days: []int{2}}, // 2번째가 잘못됨
		}
		msg := validateScheduleList(list)
		if msg == "" {
			t.Fatalf("문제 있는데 통과함")
		}
		if !containsDigit(msg, "2") {
			t.Errorf("2번째 항목이 문제라고 알려줘야 하는데 %q", msg)
		}
	})

	// 부정 케이스: 두 항목이 서로 요일·시간대가 겹침.
	Scenario(t, "GIVEN 두 항목이 요일·시간대가 서로 겹침 WHEN 검증 THEN 겹친다는 오류를 반환한다 (부정 케이스)", func(t *testing.T) {
		list := []engine.Schedule{
			{StartHour: 9, StartMinute: 0, EndHour: 12, EndMinute: 0, Days: []int{1}},
			{StartHour: 10, StartMinute: 0, EndHour: 14, EndMinute: 0, Days: []int{1}},
		}
		if msg := validateScheduleList(list); msg == "" {
			t.Errorf("겹치는데 문제 없다고 나옴")
		}
	})

	Scenario(t, "GIVEN 목록이 비어있음 WHEN 검증 THEN 문제 없음을 반환한다 (경계 케이스 — 일정을 전부 지운 상태로 저장 가능해야 함)", func(t *testing.T) {
		if msg := validateScheduleList(nil); msg != "" {
			t.Errorf("빈 목록은 문제 없어야 하는데 %q", msg)
		}
	})
}

func containsDigit(s, digit string) bool {
	for i := 0; i+len(digit) <= len(s); i++ {
		if s[i:i+len(digit)] == digit {
			return true
		}
	}
	return false
}
