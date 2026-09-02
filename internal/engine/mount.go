package engine

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
)

// Mount mirrors the fields of a single mount entry in mounts.json.
type Mount struct {
	ID         string     `json:"id"`
	Remote     string     `json:"remote"`
	RemotePath string     `json:"remote_path"`
	Drive      string     `json:"drive"`
	CacheDir   string     `json:"cache_dir"`
	CacheMode  string     `json:"cache_mode"`
	ExtraFlags string     `json:"extra_flags"`
	AutoMount  bool       `json:"auto_mount"`
	Schedules  []Schedule `json:"schedules,omitempty"`
}

// NewMountID generates a new random v4-UUID-shaped identifier for a mount
// entry, mirroring the Python version's use of uuid.uuid4() as each
// mount's "id". Used to target a specific entry for edit/delete/mount
// operations regardless of remote name changes.
func NewMountID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

var volnameFlagRe = regexp.MustCompile(`^--volname=(.+)$`)

// GetVolname determines the drive volume label, mirroring _get_volname.
//
// Priority:
//  1. an explicit --volname=VALUE inside ExtraFlags
//  2. the last path element of RemotePath
//     PLEX:KODI        -> KODI
//     gds:GDRIVE/VIDEO -> VIDEO
//  3. the remote name itself
//     nas:              -> nas
func GetVolname(m Mount) string {
	if m.ExtraFlags != "" {
		for _, token := range strings.Split(m.ExtraFlags, ";") {
			token = strings.TrimSpace(token)
			if match := volnameFlagRe.FindStringSubmatch(token); match != nil {
				return strings.TrimSpace(match[1])
			}
		}
	}

	rpath := strings.Trim(strings.ReplaceAll(strings.TrimSpace(m.RemotePath), "\\", "/"), "/")
	if rpath != "" {
		segs := strings.Split(rpath, "/")
		return segs[len(segs)-1]
	}

	return strings.Split(m.Remote, ":")[0]
}

// BuildCmd builds the rclone mount command-line argument list, mirroring
// build_cmd(). exePath is the path to rclone.exe (as a plain string —
// callers decide how to resolve/quote it for exec).
func BuildCmd(exePath string, m Mount) []string {
	rpath := strings.Trim(strings.ReplaceAll(strings.TrimSpace(m.RemotePath), "\\", "/"), "/")
	driveTarget := strings.TrimSpace(m.Drive)
	if driveTarget == "" {
		driveTarget = " "
	}

	volname := GetVolname(m)
	cmd := []string{
		exePath, "mount", m.Remote + ":" + rpath, driveTarget,
		"--volname", volname,
	}

	if m.CacheDir != "" {
		cmd = append(cmd, "--cache-dir", m.CacheDir)
	}
	if m.CacheMode != "" {
		cmd = append(cmd, "--vfs-cache-mode", m.CacheMode)
	}

	extra := strings.TrimSpace(m.ExtraFlags)
	if extra != "" {
		for _, f := range strings.Split(extra, ";") {
			f = strings.TrimSpace(f)
			if f != "" && !strings.HasPrefix(f, "--volname") {
				cmd = append(cmd, f)
			}
		}
	}

	return cmd
}
