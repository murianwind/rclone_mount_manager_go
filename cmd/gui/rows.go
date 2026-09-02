package main

import "github.com/Murianwind/rclone-manager-go/internal/engine"

// rowKind은 테이블의 한 행이 "원본 리모트"인지 "마운트 설정"인지 구분한다.
// 원본 행은 마운트로 아직 등록 안 된 rclone.conf의 리모트를 보여주고,
// "마운트" 버튼으로 마운트 추가 다이얼로그에 그대로 넘길 수 있다.
type rowKind int

const (
	rowKindRemote rowKind = iota
	rowKindMount
)

// tableRow는 원본/마운트 두 목록을 한 화면에 같이 보여주기 위한
// 통합 뷰다. kind에 따라 remote 또는 mount 필드만 유효하다.
type tableRow struct {
	kind   rowKind
	remote engine.Remote
	mount  engine.Mount
}

// rows는 원본 리모트 목록을 먼저, 마운트 목록을 그 뒤에 이어붙인
// 전체 테이블 행을 만든다 — 기존 Python 버전의 트리 구성 순서와 동일하다.
func (rm *rcloneManager) rows() []tableRow {
	cfg := rm.cfgSnapshot()
	rows := make([]tableRow, 0, len(cfg.Remotes)+len(cfg.Mounts))
	for _, r := range cfg.Remotes {
		rows = append(rows, tableRow{kind: rowKindRemote, remote: r})
	}
	for _, m := range cfg.Mounts {
		rows = append(rows, tableRow{kind: rowKindMount, mount: m})
	}
	return rows
}

// kindLabel은 구분 컬럼에 보여줄 텍스트다.
func kindLabel(kind rowKind) string {
	if kind == rowKindMount {
		return "💾 마운트"
	}
	return "☁️ 원본"
}

// remoteDisplayText는 원본 행의 리모트 컬럼에 보여줄 "[타입] 이름" 형식이다.
func remoteDisplayText(r engine.Remote) string {
	return "[" + r.Type + "] " + r.Name
}
