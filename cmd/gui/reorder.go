package main

import (
	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// swapRemotesUp/Down, swapMountsUp/Down는 슬라이스 안에서 인접한 두
// 원소를 맞바꾼다. 인덱스가 경계 밖이면 아무 일도 하지 않고 false를
// 반환한다 — rm.cfg 없이도 경계 조건을 테스트할 수 있도록 순수 함수로
// 뽑아뒀다. 원본과 마운트를 같은 함수로 합치지 않은 이유는 두 슬라이스가
// 서로 다른 타입([]engine.Remote / []engine.Mount)이기 때문이다.

func swapRemotesUp(remotes []engine.Remote, i int) bool {
	if i <= 0 || i >= len(remotes) {
		return false
	}
	remotes[i], remotes[i-1] = remotes[i-1], remotes[i]
	return true
}

func swapRemotesDown(remotes []engine.Remote, i int) bool {
	if i < 0 || i >= len(remotes)-1 {
		return false
	}
	remotes[i], remotes[i+1] = remotes[i+1], remotes[i]
	return true
}

func swapMountsUp(mounts []engine.Mount, i int) bool {
	if i <= 0 || i >= len(mounts) {
		return false
	}
	mounts[i], mounts[i-1] = mounts[i-1], mounts[i]
	return true
}

func swapMountsDown(mounts []engine.Mount, i int) bool {
	if i < 0 || i >= len(mounts)-1 {
		return false
	}
	mounts[i], mounts[i+1] = mounts[i+1], mounts[i]
	return true
}

func indexOfRemote(remotes []engine.Remote, name string) int {
	for i, r := range remotes {
		if r.Name == name {
			return i
		}
	}
	return -1
}

func indexOfMount(mounts []engine.Mount, id string) int {
	for i, m := range mounts {
		if m.ID == id {
			return i
		}
	}
	return -1
}

// moveSelectedUp/Down은 현재 테이블에서 선택된 행이 속한 목록(원본 또는
// 마운트)만 재정렬한다 — 기존 Python 버전의 _move_up/_move_down과 동일하게,
// 원본과 마운트 경계를 넘어서는 이동은 없다. 이동 후에도 선택이 옮겨진
// 항목을 계속 따라가도록 유지한다(연속으로 위/아래를 눌러도 항상 같은
// 항목이 움직이게 하기 위함) — Python 버전의 selection_set(sel[0])에 대응.
func (rm *rcloneManager) moveSelectedUp() { rm.moveSelected(swapRemotesUp, swapMountsUp, -1) }

func (rm *rcloneManager) moveSelectedDown() { rm.moveSelected(swapRemotesDown, swapMountsDown, +1) }

func (rm *rcloneManager) moveSelected(
	swapRemote func([]engine.Remote, int) bool,
	swapMount func([]engine.Mount, int) bool,
	delta int,
) {
	rows := rm.rows()
	if rm.selectedRow < 0 || rm.selectedRow >= len(rows) {
		return
	}
	row := rows[rm.selectedRow]

	moved := false
	rm.withCfg(func(cfg *engine.Config) {
		switch row.kind {
		case rowKindRemote:
			idx := indexOfRemote(cfg.Remotes, row.remote.Name)
			if idx < 0 || !swapRemote(cfg.Remotes, idx) {
				return
			}
		default:
			idx := indexOfMount(cfg.Mounts, row.mount.ID)
			if idx < 0 || !swapMount(cfg.Mounts, idx) {
				return
			}
		}
		moved = true
	})
	if !moved {
		return
	}

	rm.selectedRow += delta
	rm.persist()
}
