package main

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// defaultNewSchedule is what a freshly added schedule entry starts as —
// weekday business hours, the most common case, so most users only need
// to adjust rather than fill in from a blank slate.
func defaultNewSchedule() engine.Schedule {
	return engine.Schedule{StartHour: 9, StartMinute: 0, EndHour: 18, EndMinute: 0, Days: []int{1, 2, 3, 4, 5}}
}

var weekdayShortLabels = [7]string{"일", "월", "화", "수", "목", "금", "토"}

// showScheduleDialog lets the user add/remove/edit this mount's schedule
// entries. Each entry gets its own day-toggle row + start/end time
// pickers; "저장" validates every entry (see engine.ValidateSchedule and
// engine.SchedulesOverlap) before actually closing — an invalid entry
// shows what's wrong and leaves the dialog open instead of discarding
// everything the user just entered.
func (rm *rcloneManager) showScheduleDialog(m engine.Mount) {
	schedules := append([]engine.Schedule(nil), m.Schedules...) // 작업용 복사본

	rowsBox := container.NewVBox()
	var refreshRows func()

	addRow := func(s engine.Schedule) {
		idx := len(schedules)
		schedules = append(schedules, s)

		selectedDays := func() map[int]bool {
			set := make(map[int]bool, 7)
			for _, d := range schedules[idx].Days {
				set[d] = true
			}
			return set
		}

		var dayButtons [7]*widget.Button
		refreshDayStyles := func() {
			sel := selectedDays()
			for d, b := range dayButtons {
				if sel[d] {
					b.Importance = widget.HighImportance
				} else {
					b.Importance = widget.MediumImportance
				}
				b.Refresh()
			}
		}
		toggleDay := func(d int) {
			sel := selectedDays()
			if sel[d] {
				kept := schedules[idx].Days[:0]
				for _, x := range schedules[idx].Days {
					if x != d {
						kept = append(kept, x)
					}
				}
				schedules[idx].Days = kept
			} else {
				schedules[idx].Days = append(schedules[idx].Days, d)
			}
			refreshDayStyles()
		}

		dayRow := container.NewHBox()
		for d := 0; d < 7; d++ {
			d := d
			b := widget.NewButton(weekdayShortLabels[d], func() { toggleDay(d) })
			dayButtons[d] = b
			dayRow.Add(b)
		}
		allBtn := widget.NewButton("매일", func() {
			schedules[idx].Days = []int{0, 1, 2, 3, 4, 5, 6}
			refreshDayStyles()
		})
		dayRow.Add(allBtn)
		refreshDayStyles()

		startHour := widget.NewEntry()
		startHour.SetText(fmt.Sprintf("%02d", schedules[idx].StartHour))
		startHour.OnChanged = func(v string) { schedules[idx].StartHour, _ = strconv.Atoi(v) }

		startMinute := widget.NewEntry()
		startMinute.SetText(fmt.Sprintf("%02d", schedules[idx].StartMinute))
		startMinute.OnChanged = func(v string) { schedules[idx].StartMinute, _ = strconv.Atoi(v) }

		endHour := widget.NewEntry()
		endHour.SetText(fmt.Sprintf("%02d", schedules[idx].EndHour))
		endHour.OnChanged = func(v string) { schedules[idx].EndHour, _ = strconv.Atoi(v) }

		endMinute := widget.NewEntry()
		endMinute.SetText(fmt.Sprintf("%02d", schedules[idx].EndMinute))
		endMinute.OnChanged = func(v string) { schedules[idx].EndMinute, _ = strconv.Atoi(v) }

		removeBtn := widget.NewButton("삭제", func() {
			schedules = append(schedules[:idx], schedules[idx+1:]...)
			refreshRows()
		})

		timeRow := container.NewHBox(
			widget.NewLabel("시작"), startHour, widget.NewLabel(":"), startMinute,
			widget.NewLabel("~ 종료"), endHour, widget.NewLabel(":"), endMinute,
			removeBtn,
		)
		rowsBox.Add(container.NewVBox(dayRow, timeRow, widget.NewSeparator()))
	}

	refreshRows = func() {
		rowsBox.Objects = nil
		saved := schedules
		schedules = nil
		for _, s := range saved {
			addRow(s)
		}
		rowsBox.Refresh()
	}

	if len(schedules) == 0 {
		addRow(defaultNewSchedule())
	} else {
		refreshRows()
	}

	addEntryBtn := widget.NewButton("+ 일정 추가", func() {
		addRow(defaultNewSchedule())
	})

	hint := widget.NewLabel("시·분은 24시간제 숫자로 입력해 주세요 (예: 오후 6시는 18로)")
	hint.TextStyle = fyne.TextStyle{Italic: true}

	scroll := container.NewVScroll(rowsBox)
	scroll.SetMinSize(fyne.NewSize(480, 260))
	content := container.NewBorder(hint, addEntryBtn, nil, nil, scroll)

	var d dialog.Dialog
	saveBtn := widget.NewButton("저장", func() {
		if msg := validateScheduleList(schedules); msg != "" {
			dialog.ShowInformation("일정 확인", msg, rm.win)
			return
		}
		m.Schedules = schedules
		rm.saveMount(m)
		d.Hide()
	})
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton("취소", func() { d.Hide() })

	fullContent := container.NewBorder(nil, container.NewHBox(cancelBtn, saveBtn), nil, nil, content)
	d = dialog.NewCustomWithoutButtons(fmt.Sprintf("일정 — %s:%s", m.Remote, m.RemotePath), fullContent, rm.win)
	d.Resize(fyne.NewSize(520, 420))
	d.Show()
}

// validateScheduleList checks every entry individually (engine.ValidateSchedule)
// and every pair against each other (engine.SchedulesOverlap). Returns ""
// if the whole list is fine, else a message naming the first problem found.
func validateScheduleList(schedules []engine.Schedule) string {
	for i, s := range schedules {
		if msg := engine.ValidateSchedule(s); msg != "" {
			return fmt.Sprintf("%d번째 항목: %s", i+1, msg)
		}
	}
	for i := 0; i < len(schedules); i++ {
		for j := i + 1; j < len(schedules); j++ {
			if engine.SchedulesOverlap(schedules[i], schedules[j]) {
				return fmt.Sprintf("%d번째와 %d번째 항목의 요일·시간대가 겹칩니다.", i+1, j+1)
			}
		}
	}
	return ""
}
