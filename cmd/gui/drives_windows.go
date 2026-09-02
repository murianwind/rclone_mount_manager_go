//go:build windows

package main

import "golang.org/x/sys/windows"

// availableDrives returns every drive letter currently accessible (e.g.
// ["C:\\", "D:\\"]) — the same Windows API Explorer itself uses to list
// drives. Used by the file/folder picker to jump between drives, since
// Windows has no single filesystem root; ".." at a drive's own root just
// stays there (filepath.Dir("C:\\") == "C:\\"), so there's no way to
// "navigate up" into a different drive the way there is on a
// single-rooted filesystem.
func availableDrives() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var drives []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) != 0 {
			drives = append(drives, string(rune('A'+i))+":\\")
		}
	}
	return drives
}
