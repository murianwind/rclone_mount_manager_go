package main

import (
	"time"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// startScheduleMonitor checks every mount's schedules once a minute and
// acts on engine.DecideScheduleAction's verdict. Runs an immediate check
// on startup too, so a mount whose window is already open when the app
// launches doesn't wait up to a minute for its first mount.
func (rm *rcloneManager) startScheduleMonitor() {
	go func() {
		rm.tickSchedules()
		for {
			time.Sleep(time.Minute)
			rm.tickSchedules()
		}
	}()
}

func (rm *rcloneManager) tickSchedules() {
	now := time.Now()
	for _, m := range rm.cfgSnapshot().Mounts {
		if len(m.Schedules) == 0 {
			continue
		}
		running := rm.isRunning(m.ID)
		skip := rm.getScheduleSkip(m.ID)

		switch engine.DecideScheduleAction(m.Schedules, now, running, skip) {
		case engine.ScheduleActionMount:
			rm.mountWithOrigin(m, true)
		case engine.ScheduleActionUnmount:
			rm.unmount(m.ID)
		case engine.ScheduleActionResetSkip:
			rm.setScheduleSkip(m.ID, false)
		}
	}
}

// getScheduleSkip/setScheduleSkip guard rm.scheduleSkip — set when the
// user manually unmounts a scheduled mount *during* its own window (see
// table.go's toggle handler), cleared once that window ends (tickSchedules,
// via ScheduleActionResetSkip) so the *next* window works normally again.
func (rm *rcloneManager) getScheduleSkip(mountID string) bool {
	rm.scheduleSkipMu.Lock()
	defer rm.scheduleSkipMu.Unlock()
	return rm.scheduleSkip[mountID]
}

func (rm *rcloneManager) setScheduleSkip(mountID string, v bool) {
	rm.scheduleSkipMu.Lock()
	defer rm.scheduleSkipMu.Unlock()
	if rm.scheduleSkip == nil {
		rm.scheduleSkip = map[string]bool{}
	}
	if v {
		rm.scheduleSkip[mountID] = true
	} else {
		delete(rm.scheduleSkip, mountID)
	}
}
