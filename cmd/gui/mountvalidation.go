package main

import (
	"os"
	"strings"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// validateMount checks a mount for save-blocking conflicts against the
// rest of cfg.Mounts (ignoring the mount itself when editing, matched by
// ID). Returns "" if there's no conflict, else a user-facing message.
// Pulled out as a pure function for testing — see mountvalidation_test.go.
func validateMount(m engine.Mount, existingMounts []engine.Mount) string {
	drive := m.Drive
	for _, other := range existingMounts {
		if other.ID == m.ID {
			continue
		}
		if drive != "" && other.Drive == drive {
			return "이미 사용 중인 드라이브 문자입니다."
		}
		if other.Remote == m.Remote && other.RemotePath == m.RemotePath {
			return "동일한 리모트/경로가 이미 등록되어 있습니다."
		}
	}
	return ""
}

// validateMountLocation checks a folder-path mount location against
// rclone's own requirement: the folder must NOT already exist — rclone
// creates it itself at mount time, and refuses ("mountpoint path already
// exists") if it's already there. A plain drive letter ("Z:") has no such
// requirement and always passes. Touches the filesystem, so it's kept
// separate from validateMount (which stays pure) — tested with a real
// temp directory instead, see mountvalidation_test.go.
func validateMountLocation(drive string) string {
	drive = strings.TrimSpace(drive)
	if drive == "" || !strings.ContainsAny(drive, `\/`) {
		return "" // 비어있음(자동) 또는 드라이브 문자뿐 — 확인할 폴더가 없음
	}
	info, err := os.Stat(drive)
	if err != nil {
		return "" // 없음 — rclone이 마운트 시점에 만들 수 있으니 정상
	}
	if info.IsDir() {
		return "이 폴더가 이미 존재합니다. rclone은 폴더가 없는 상태여야 그 자리에 마운트할 수 있습니다 — 폴더를 지우거나 다른 경로를 써 주세요."
	}
	return "이 경로에 이미 파일이 있습니다. 다른 경로를 써 주세요."
}
