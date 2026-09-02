package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// table column indices — keep in sync with buildTable's header labels.
const (
	colKind = iota
	colAuto
	colDrive
	colRemote
	colStatus
	colActions
	colCount
)

// tableContentWidth is the sum of every column's fixed width — used
// elsewhere to size the window sensibly, since Table itself doesn't
// factor column widths into its own MinSize.
const tableContentWidth = 80 + 46 + 70 + 230 + 70 + 240

// columnWidths holds each column's fixed pixel width, indexed by the col*
// constants — shared by buildTable (SetColumnWidth) and buildTableHeader
// (matching fixed-width labels), so the two never drift out of sync.
var columnWidths = [colCount]float32{80, 46, 70, 230, 70, 240}

// columnHeaderLabels are the column titles, indexed the same way.
var columnHeaderLabels = [colCount]string{"구분", "자동", "드라이브", "리모트(서브경로)", "상태", ""}

func (rm *rcloneManager) buildTable() {
	rm.table = widget.NewTable(
		func() (int, int) { return len(rm.rows()), colCount },
		func() fyne.CanvasObject {
			// Table은 이 함수가 만든 "빈 템플릿"의 MinSize()로 기본 행
			// 높이를 스스로 결정한다(templateSize()). 빈 Stack만 반환하면
			// 그 크기가 0이 되어 모든 행이 높이 0으로 겹쳐 그려진다 — 실제
			// 내용은 UpdateCell에서 바로 이 자리를 교체하므로, 여기서는
			// "적당한 높이가 있다"는 것만 알려주면 된다.
			placeholder := canvas.NewRectangle(color.Transparent)
			placeholder.SetMinSize(fyne.NewSize(0, 34))
			return container.NewStack(placeholder)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			rm.updateTableCell(id, cell.(*fyne.Container))
		},
	)
	// Table의 내장 헤더(ShowHeaderRow)는 본문과 간격 없이 항상 딱
	// 붙어버려서(HideSeparators를 켜든 꺼든 마찬가지) 안 쓴다 — 대신
	// buildTableHeader()로 직접 만든 헤더를 표 위에 별도로 놓고, 그
	// 사이에 우리가 원하는 만큼 여백을 둔다.
	rm.table.HideSeparators = true
	for col, w := range columnWidths {
		rm.table.SetColumnWidth(col, w)
	}

	rm.table.OnSelected = func(id widget.TableCellID) {
		// Fyne이 "선택된 셀"에 자체적으로 그리는 테두리(흰 줄처럼 보이던
		// 것)를 바로 지운다 — 그 표시는 우리가 직접 굵은 글씨로 대신한다.
		rm.table.Unselect(id)
		rm.selectedRow = toggleSelection(rm.selectedRow, id.Row)
		rm.table.Refresh()
	}
}

// toggleSelection decides the next selectedRow value: clicking the
// already-selected row deselects it (-1); clicking any other row selects
// that one instead.
func toggleSelection(current, clicked int) int {
	if current == clicked {
		return -1
	}
	return clicked
}

// buildTableHeader is our own header row, column-width-matched to the
// table below it, with a real visible gap between the two (see build()).
func buildTableHeader() fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, colCount)
	for col, title := range columnHeaderLabels {
		label := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		cells[col] = container.New(layout.NewGridWrapLayout(fyne.NewSize(columnWidths[col], label.MinSize().Height)), label)
	}
	return container.NewHBox(cells...)
}

// updateTableCell fills in one cell. CreateCell can't know in advance
// which column a recycled template will be asked to render, so each
// helper (cellCheck/cellLabel/cellActionButtons) replaces the cell's
// content if it isn't already the right widget type.
func (rm *rcloneManager) updateTableCell(id widget.TableCellID, content *fyne.Container) {
	rows := rm.rows()
	if id.Row >= len(rows) {
		return
	}
	row := rows[id.Row]
	selected := id.Row == rm.selectedRow

	if id.Col == colKind {
		rm.setCellText(content, kindLabel(row.kind), selected)
		return
	}

	if row.kind == rowKindRemote {
		rm.updateRemoteRowCell(id.Col, content, row.remote, selected)
		return
	}
	rm.updateMountRowCell(id.Col, content, row.mount, selected)
}

func (rm *rcloneManager) updateRemoteRowCell(col int, cell *fyne.Container, r engine.Remote, selected bool) {
	switch col {
	case colAuto, colDrive, colStatus:
		rm.setCellText(cell, "", selected)
	case colRemote:
		rm.setCellText(cell, remoteDisplayText(r), selected)
	case colActions:
		importBtn, blank1, blank2, delBtn := rm.cellActionButtons(cell)
		importBtn.SetText("가져오기")
		importBtn.OnTapped = func() { rm.showMountDialog(nil, r.Name) }
		blank1.Hide() // 원본 행에는 편집 개념이 없어서 안 씀
		blank2.Hide() // 원본 행에는 일정 개념이 없어서 안 씀
		delBtn.SetText("삭제")
		delBtn.OnTapped = func() { rm.confirmDeleteRemote(r) }
	}
}

func (rm *rcloneManager) updateMountRowCell(col int, cell *fyne.Container, m engine.Mount, selected bool) {
	switch col {
	case colAuto:
		check := rm.cellCheck(cell)
		check.SetChecked(m.AutoMount)
		if len(m.Schedules) > 0 {
			check.Disable()
		} else {
			check.Enable()
		}
		check.OnChanged = func(checked bool) {
			m.AutoMount = checked
			rm.saveMount(m)
		}
	case colDrive:
		rm.setCellText(cell, displayDrive(m.Drive), selected)
	case colRemote:
		rm.setCellText(cell, fmt.Sprintf("%s:%s", m.Remote, m.RemotePath), selected)
	case colStatus:
		rm.setCellText(cell, statusLabel(rm.isRunning(m.ID)), selected)
	case colActions:
		toggle, editBtn, scheduleBtn, delBtn := rm.cellActionButtons(cell)
		running := rm.isRunning(m.ID)
		toggle.SetText(toggleLabel(running))
		toggle.OnTapped = func() {
			if running {
				if len(m.Schedules) > 0 && engine.IsWithinSchedule(m.Schedules, time.Now()) {
					rm.setScheduleSkip(m.ID, true)
				}
				rm.unmount(m.ID)
			} else {
				rm.mount(m)
			}
		}
		editBtn.Show()
		editBtn.SetText("편집")
		editBtn.OnTapped = func() { rm.showMountDialog(&m, "") }

		scheduleBtn.SetText("일정")
		if m.AutoMount {
			scheduleBtn.Hide() // "자동"이 켜져 있으면 일정은 의미가 없다 — 서로 배타적
		} else {
			scheduleBtn.Show()
			if len(m.Schedules) > 0 {
				scheduleBtn.Importance = widget.HighImportance // 일정이 등록돼 있음을 한눈에
			}
			scheduleBtn.Refresh()
			scheduleBtn.OnTapped = func() { rm.showScheduleDialog(m) }
		}

		delBtn.SetText("삭제")
		delBtn.OnTapped = func() { rm.confirmDelete(m) }
	}
}

func (rm *rcloneManager) cellCheck(cell *fyne.Container) *widget.Check {
	if len(cell.Objects) == 1 {
		if c, ok := cell.Objects[0].(*widget.Check); ok {
			return c
		}
	}
	c := widget.NewCheck("", nil)
	cell.Objects = []fyne.CanvasObject{c}
	return c
}

func (rm *rcloneManager) cellLabel(cell *fyne.Container) *widget.Label {
	if len(cell.Objects) == 1 {
		if l, ok := cell.Objects[0].(*widget.Label); ok {
			return l
		}
	}
	l := widget.NewLabel("")
	cell.Objects = []fyne.CanvasObject{l}
	return l
}

// setCellText sets a label cell's text and — instead of any background
// color, which either got hidden behind opaque widgets or clashed with
// Fyne's own selection outline — bolds it when its row is selected.
func (rm *rcloneManager) setCellText(cell *fyne.Container, text string, bold bool) {
	label := rm.cellLabel(cell)
	label.SetText(text)
	if label.TextStyle.Bold != bold {
		label.TextStyle.Bold = bold
		label.Refresh()
	}
}

// cellActionButtons returns a 4-button slot shared by both row kinds:
// mount rows use all four (토글/편집/일정/삭제); remote rows only use the
// first and last (가져오기/삭제) — the middle two are left blank rather
// than removed, so the recycled widget shape stays consistent.
func (rm *rcloneManager) cellActionButtons(cell *fyne.Container) (first, second, third, last *widget.Button) {
	if len(cell.Objects) == 1 {
		if row, ok := cell.Objects[0].(*fyne.Container); ok && len(row.Objects) == 4 {
			if firstWrap, ok := row.Objects[0].(*fyne.Container); ok && len(firstWrap.Objects) == 1 {
				if f, ok := firstWrap.Objects[0].(*widget.Button); ok {
					if s, ok := row.Objects[1].(*widget.Button); ok {
						if t, ok := row.Objects[2].(*widget.Button); ok {
							if l, ok := row.Objects[3].(*widget.Button); ok {
								s.SetText("")
								s.OnTapped = nil
								t.SetText("")
								t.OnTapped = nil
								t.Importance = widget.MediumImportance
								return f, s, t, l
							}
						}
					}
				}
			}
		}
	}
	first = widget.NewButton("", nil)
	second = widget.NewButton("", nil)
	third = widget.NewButton("", nil)
	last = widget.NewButton("", nil)
	// 버튼 라벨 길이가 바뀌어도(마운트/해제/가져오기 등) 폭이 흔들리지
	// 않도록 첫 번째 버튼만 고정 폭으로 감싼다 — 뒤따르는 버튼들의
	// 위치가 흔들리는 걸 막기 위함.
	firstFixed := container.New(layout.NewGridWrapLayout(fyne.NewSize(64, 34)), first)
	cell.Objects = []fyne.CanvasObject{container.NewHBox(firstFixed, second, third, last)}
	return first, second, third, last
}

// displayDrive is the pure formatting rule shared by the table's 드라이브
// column: "" reads as "자동" (rclone picks an unused drive letter itself).
func displayDrive(drive string) string {
	drive = strings.TrimSpace(drive)
	if drive == "" {
		return "(자동)"
	}
	return drive
}

func statusLabel(running bool) string {
	if running {
		return "연결됨"
	}
	return "해제됨"
}

func toggleLabel(running bool) string {
	if running {
		return "해제"
	}
	return "마운트"
}
