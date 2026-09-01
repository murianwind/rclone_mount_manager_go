package main

import (
	"strings"
	"time"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// maxStaleMountRetries caps how many times a single mount is automatically
// retried after a "mountpoint path already exists" failure before giving
// up and showing the normal failure dialog.
const maxStaleMountRetries = 3

// staleMountRetryDelay is how long to wait before each retry — enough for
// WinFsp/Windows to finish releasing a drive letter's previous mountpoint
// registration on its own.
const staleMountRetryDelay = 3 * time.Second

// isStaleMountpointError reports whether detail (rclone's captured stderr)
// looks like the specific "the drive letter is still registered as a
// mountpoint from before" failure, rather than a genuine problem with the
// remote itself (bad credentials, unreachable, wrong path, ...).
//
// This is exactly what a fresh PC reboot can leave behind: a reboot kills
// every process at once, including any running rclone.exe — there's no
// window for a graceful unmount at all, no matter how our own shutdown
// timeouts are tuned. WinFsp/Windows sometimes hasn't fully released the
// drive letter's mountpoint registration by the moment this app
// auto-mounts again at startup, even though nothing is actually still
// using it — a boot-timing issue, not a real failure. Pulled out as a
// pure function for testing — see staleretry_test.go.
func isStaleMountpointError(detail string) bool {
	return strings.Contains(detail, "mountpoint path already exists")
}

// noteStaleMountRetry records one more stale-mountpoint retry for mountID
// and reports whether another retry is still allowed (i.e. the count,
// after this one, is within maxStaleMountRetries).
func (rm *rcloneManager) noteStaleMountRetry(mountID string) (retriesUsed int, allowed bool) {
	rm.staleMu.Lock()
	defer rm.staleMu.Unlock()
	rm.staleRetries[mountID]++
	n := rm.staleRetries[mountID]
	return n, n <= maxStaleMountRetries
}

// clearStaleMountRetries resets mountID's retry count — called once a
// mount actually starts cleanly, so a much later, unrelated failure
// doesn't inherit an old retry streak.
func (rm *rcloneManager) clearStaleMountRetries(mountID string) {
	rm.staleMu.Lock()
	delete(rm.staleRetries, mountID)
	rm.staleMu.Unlock()
}

// retryStaleMount waits staleMountRetryDelay, then tries the mount again
// with the same auto/manual origin as the attempt that just failed.
func (rm *rcloneManager) retryStaleMount(m engine.Mount, auto bool) {
	time.Sleep(staleMountRetryDelay)
	rm.mountWithOrigin(m, auto)
}
