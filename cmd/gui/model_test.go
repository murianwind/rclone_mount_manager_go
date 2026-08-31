package main

import (
	"sync"
	"testing"

	"github.com/Murianwind/rclone-manager-go/internal/engine"
)

// TestCfgConcurrentAccess is a direct regression test for a real, confirmed
// data race: rm.cfg.Mounts used to be read directly (no lock) by the
// network-monitor goroutine (autoMountAll) and the rclone-update-check
// goroutine (activeMountsSnapshot), while UI actions (add/edit/delete/
// reorder a mount) mutated the same slice on the Fyne main thread with no
// synchronization at all. This simulates exactly that: one goroutine
// repeatedly reading via cfgSnapshot() while another repeatedly mutates
// via withCfg() — run with -race, this must report nothing.
func TestCfgConcurrentAccess(t *testing.T) {
	Scenario(t, "GIVEN 한 고루틴은 계속 cfgSnapshot()으로 읽고 다른 고루틴은 계속 withCfg()로 mounts를 수정함 WHEN 동시에 오래 실행 THEN -race 기준으로 데이터 레이스가 없다 (회귀 테스트)", func(t *testing.T) {
		rm := &rcloneManager{}
		rm.cfg.Mounts = []engine.Mount{{ID: "seed", Remote: "gds", RemotePath: "x"}}

		const iterations = 2000
		var wg sync.WaitGroup
		wg.Add(2)

		// 네트워크 감시/업데이트 확인 고루틴을 흉내내는 쪽 — 계속 읽기만 한다.
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				snap := rm.cfgSnapshot()
				_ = len(snap.Mounts) // 슬라이스를 실제로 훑어봐야 레이스가 드러남
				for _, m := range snap.Mounts {
					_ = m.ID
				}
			}
		}()

		// UI 액션(추가/삭제/편집)을 흉내내는 쪽 — 계속 mounts를 바꾼다.
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				rm.withCfg(func(cfg *engine.Config) {
					cfg.Mounts = append(cfg.Mounts, engine.Mount{ID: "x"})
					if len(cfg.Mounts) > 5 {
						cfg.Mounts = cfg.Mounts[:1]
					}
				})
			}
		}()

		wg.Wait()
	})
}
