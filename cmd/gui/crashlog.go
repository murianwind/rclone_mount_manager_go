package main

import (
	"fmt"
	"os"
	"time"
)

// archivePreviousCrashLog checks whether a crash log from a previous run
// exists and actually has content, and if so renames it out of the way
// (so this session's redirectStderrToFile starts a fresh, empty file
// instead of appending onto an old, possibly unrelated crash forever).
//
// Returns the path to check for details and true, or ("", false) if
// there was nothing to report. If the rename itself fails, the original
// path is returned unchanged — still useful to point the user at, even if
// it couldn't be moved out of the way this time.
func archivePreviousCrashLog(path string, now time.Time) (archivedPath string, hadCrash bool) {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return "", false
	}

	archived := fmt.Sprintf("%s.%s", path, now.Format("20060102-150405"))
	if err := os.Rename(path, archived); err != nil {
		return path, true
	}
	return archived, true
}
