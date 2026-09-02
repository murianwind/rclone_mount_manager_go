package engine

import "time"

// Schedule is one "이 요일들의 이 시간대엔 마운트돼 있어야 한다" rule.
// Intentionally day-scoped (never crosses midnight) — the recommended way
// to express an overnight window is two adjacent Schedules, e.g.
// 22:00–23:59 + 00:00–06:00, which keeps every comparison a same-day
// time-of-day comparison with no wraparound arithmetic anywhere.
type Schedule struct {
	StartHour   int   `json:"start_hour"`
	StartMinute int   `json:"start_minute"`
	EndHour     int   `json:"end_hour"`
	EndMinute   int   `json:"end_minute"`
	Days        []int `json:"days"` // time.Weekday values: 0=Sunday..6=Saturday
}

// startMinutes/endMinutes convert a Schedule's start/end into
// minutes-since-midnight, the common unit every comparison below works in.
func (s Schedule) startMinutes() int { return s.StartHour*60 + s.StartMinute }
func (s Schedule) endMinutes() int   { return s.EndHour*60 + s.EndMinute }

// ValidateSchedule checks one schedule in isolation (not against any
// others — see SchedulesOverlap for that). Returns "" if valid, else a
// user-facing message naming the problem.
func ValidateSchedule(s Schedule) string {
	if s.StartHour < 0 || s.StartHour > 23 || s.EndHour < 0 || s.EndHour > 23 {
		return "시는 0~23 사이여야 합니다."
	}
	if s.StartMinute < 0 || s.StartMinute > 59 || s.EndMinute < 0 || s.EndMinute > 59 {
		return "분은 0~59 사이여야 합니다."
	}
	if len(s.Days) == 0 {
		return "요일을 하나 이상 선택해 주세요."
	}
	if s.endMinutes() <= s.startMinutes() {
		return "종료 시각은 시작 시각보다 나중이어야 합니다."
	}
	return ""
}

// DaysOverlap reports whether a and b share at least one day.
func DaysOverlap(a, b []int) bool {
	set := make(map[int]bool, len(a))
	for _, d := range a {
		set[d] = true
	}
	for _, d := range b {
		if set[d] {
			return true
		}
	}
	return false
}

// SchedulesOverlap reports whether a and b could both be "active" at the
// same moment — they share a day AND their time-of-day ranges overlap.
// Ranges are half-open ([start,end)), so back-to-back schedules that meet
// exactly at a boundary (one ends 12:00, the next starts 12:00) are NOT
// considered overlapping — that's the normal way to chain schedules
// across a day, not a conflict.
func SchedulesOverlap(a, b Schedule) bool {
	if !DaysOverlap(a.Days, b.Days) {
		return false
	}
	return a.startMinutes() < b.endMinutes() && b.startMinutes() < a.endMinutes()
}

// IsWithinSchedule reports whether now falls inside any of schedules —
// its weekday matches one of that schedule's Days, and its time-of-day is
// within [start,end).
func IsWithinSchedule(schedules []Schedule, now time.Time) bool {
	weekday := int(now.Weekday())
	nowMinutes := now.Hour()*60 + now.Minute()

	for _, s := range schedules {
		inDay := false
		for _, d := range s.Days {
			if d == weekday {
				inDay = true
				break
			}
		}
		if !inDay {
			continue
		}
		if nowMinutes >= s.startMinutes() && nowMinutes < s.endMinutes() {
			return true
		}
	}
	return false
}

// ScheduleAction is what schedule.go's periodic tick should do for one
// scheduled mount, decided by DecideScheduleAction.
type ScheduleAction int

const (
	ScheduleActionNone ScheduleAction = iota
	ScheduleActionMount
	ScheduleActionUnmount
	ScheduleActionResetSkip
)

func (a ScheduleAction) String() string {
	switch a {
	case ScheduleActionMount:
		return "mount"
	case ScheduleActionUnmount:
		return "unmount"
	case ScheduleActionResetSkip:
		return "resetSkip"
	default:
		return "none"
	}
}

// DecideScheduleAction is the whole scheduler's decision rule, isolated
// from any process/state management so it can be tested without actually
// mounting anything:
//
//   - inside a window, not running, not skipped -> mount it
//   - outside every window but still running -> unmount it (the schedule
//     always wins on the way out, regardless of skip — skip only ever
//     suppresses an *auto-mount*, never an active unmount)
//   - outside every window, not running, but skip is still set -> clear
//     skip so the *next* window works normally
//   - anything else -> nothing to do
//
// skip is how "the user manually unmounted this during its window" is
// remembered until that window ends — see cmd/gui/schedule.go.
func DecideScheduleAction(schedules []Schedule, now time.Time, running, skip bool) ScheduleAction {
	within := IsWithinSchedule(schedules, now)

	switch {
	case within && !running && !skip:
		return ScheduleActionMount
	case !within && running:
		return ScheduleActionUnmount
	case !within && skip:
		return ScheduleActionResetSkip
	default:
		return ScheduleActionNone
	}
}
