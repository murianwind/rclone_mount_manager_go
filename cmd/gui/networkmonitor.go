package main

import (
	"time"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// autoMountFailureGrace is how long a known network outage has to persist
// before an auto-triggered mount failure is worth interrupting the user
// for — a brief blip is expected to resolve itself on the next retry.
const autoMountFailureGrace = 5 * time.Minute

// shouldSuppressAutoMountFailure decides whether an auto-triggered mount
// failure should stay silent. offlineSince is when connectivity was last
// observed down (zero if we currently believe we're connected). Suppressed
// only while a *known* outage is still within the grace period — a failure
// while we think we're online, or after an outage that's dragged on past
// the grace period, is shown normally, since at that point it's more
// likely a real, unrelated problem worth surfacing. Pulled out as a pure
// function for testing — see mount_test.go.
func shouldSuppressAutoMountFailure(offlineSince, now time.Time, grace time.Duration) bool {
	if offlineSince.IsZero() {
		return false
	}
	return now.Sub(offlineSince) < grace
}

// getOfflineSince/setOfflineSince guard offlineSince — written from the
// network-monitor goroutine, read from whichever goroutine is finishing up
// a mount attempt in waitForMountExit.
func (rm *rcloneManager) getOfflineSince() time.Time {
	rm.offlineMu.Lock()
	defer rm.offlineMu.Unlock()
	return rm.offlineSince
}

func (rm *rcloneManager) setOfflineSince(t time.Time) {
	rm.offlineMu.Lock()
	rm.offlineSince = t
	rm.offlineMu.Unlock()
}

// autoMountAll starts every mount flagged AutoMount. The network monitor owns
// the initial connectivity transition as well as later reconnects, so there
// is exactly one startup trigger for this operation.
//
// Skipped entirely while an rclone.exe update is in progress (see
// rcloneupdate.go): the network monitor now retries this every 10s while
// connected, and without this guard it could start a fresh mount with the
// *old* rclone.exe in the brief window right after installOrUpdateRclone
// unmounts everything to free the file — locking rclone.exe again right
// before we try to overwrite it, and turning what should be an automatic
// replace into "rclone.exe가 사용 중이라 자동 교체하지 못했습니다."
func (rm *rcloneManager) autoMountAll() {
	if rm.isUpdatingRclone() {
		return
	}
	for _, m := range rm.cfgSnapshot().Mounts {
		if m.AutoMount {
			rm.mountWithOrigin(m, true)
		}
	}
}

// unmountAllOnDisconnect unmounts every currently-active mount — regardless
// of its AutoMount flag — since a mount whose remote just went unreachable
// can hang the drive if left alone.
func (rm *rcloneManager) unmountAllOnDisconnect() {
	rm.activeMu.Lock()
	ids := make([]string, 0, len(rm.active))
	for id := range rm.active {
		ids = append(ids, id)
	}
	rm.activeMu.Unlock()

	for _, id := range ids {
		rm.unmount(id)
	}
}

// startNetworkMonitor polls connectivity every 10s. While connected, it
// repeatedly calls autoMountAll() (not just once on the reconnect edge) —
// mount() already no-ops instantly for anything already active, so this is
// cheap, and it matters because a genuine retry can otherwise be lost:
// rclone can take well over a minute to actually fail when there's no
// network, so a mount slot can still be "reserved" (occupied by the old,
// not-yet-failed attempt) at the exact moment connectivity returns. A
// one-shot edge trigger would silently lose that retry forever; polling
// while connected picks it up on the next cycle instead. Disconnection is
// still handled once per edge, since there's nothing to retry there.
// It is called from the Fyne app-started lifecycle hook so all UI work is safe.
func (rm *rcloneManager) startNetworkMonitor() {
	go func() {
		connected := engine.IsInternetAvailable("8.8.8.8", 53, 3*time.Second)
		if connected {
			rm.setOfflineSince(time.Time{})
			rm.autoMountAll()
		} else {
			rm.setOfflineSince(time.Now())
		}
		wasConnected := connected

		for {
			time.Sleep(10 * time.Second)
			connected = engine.IsInternetAvailable("8.8.8.8", 53, 3*time.Second)

			if connected {
				if !wasConnected {
					rm.setOfflineSince(time.Time{})
				}
				rm.autoMountAll()
			} else {
				if wasConnected {
					rm.setOfflineSince(time.Now())
					rm.unmountAllOnDisconnect()
				}
			}
			wasConnected = connected
		}
	}()
}
