package main

import (
	"os"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// pickerMode is what a picker dialog is choosing: a file, or a folder.
type pickerMode int

const (
	pickFile pickerMode = iota // click a file, then confirm
	pickDir                    // confirm picks whatever folder is currently open
)

// showFilePicker is a minimal, fully self-built file browser — built with
// the same dialog.ShowCustomConfirm pattern already proven to place its
// buttons correctly elsewhere in this app (see showRemoteSelectDialog).
// Fyne's built-in FileDialog's Cancel/Open button placement has been
// unreliable across several attempts to fix it from the outside, so this
// sidesteps that entirely by not using it.
func (rm *rcloneManager) showFilePicker(title, startDir string, onSelected func(path string)) {
	rm.showPicker(title, "선택", startDir, pickFile, onSelected)
}

// showDirPicker is showFilePicker's directory-only counterpart: clicking a
// folder navigates into it (same as showFilePicker), but confirming picks
// whatever directory is currently open rather than requiring a file
// click. Used for 캐시 디렉토리, where a folder itself is the answer.
func (rm *rcloneManager) showDirPicker(title, startDir string, onSelected func(path string)) {
	rm.showPicker(title, "이 폴더 선택", startDir, pickDir, onSelected)
}

// showPicker is the one implementation behind both pickers — they differ
// only in mode (which rows are listed, and what confirming returns).
func (rm *rcloneManager) showPicker(title, confirmText, startDir string, mode pickerMode, onSelected func(path string)) {
	if startDir == "" || !dirExists(startDir) {
		if home, err := os.UserHomeDir(); err == nil {
			startDir = home
		}
	}
	currentDir := startDir

	var entries []fileEntry
	var selected string
	var list *widget.List

	pathLabel := widget.NewLabel(currentDir)
	pathLabel.Wrapping = fyne.TextWrapBreak

	driveSelect := widget.NewSelect(availableDrives(), nil)
	driveSelect.PlaceHolder = "드라이브"

	refresh := func() {
		entries = pickerEntries(currentDir, mode)
		selected = ""
		pathLabel.SetText(currentDir)
		list.UnselectAll()
		list.Refresh()
	}
	driveSelect.OnChanged = func(d string) {
		currentDir = d
		refresh()
	}

	list = widget.NewList(
		func() int { return len(entries) + 1 }, // +1 for ".." (parent dir)
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.SetText(fileEntryLabel(id, entries))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		newDir, file := pickerClick(currentDir, entries, id)
		if file != "" {
			selected = file // 파일 선택은 목록을 다시 그리지 않는다
			return
		}
		currentDir = newDir
		refresh()
	}

	refresh()

	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(480, 320))
	header := container.NewBorder(nil, nil, driveSelect, nil, pathLabel)
	content := container.NewBorder(header, nil, nil, nil, scroll)

	dialog.ShowCustomConfirm(title, confirmText, "취소", content, func(ok bool) {
		if !ok {
			return
		}
		if path, has := pickerResult(mode, currentDir, selected); has {
			onSelected(path)
		}
	}, rm.win)
}

// pickerEntries lists dir's rows for the given mode (folders first; in
// folder mode, files aren't listed at all).
func pickerEntries(dir string, mode pickerMode) []fileEntry {
	entries := listDir(dir)
	if mode == pickDir {
		return dirsOnly(entries)
	}
	return entries
}

// pickerClick resolves a click on row id (0 is the ".." row). It returns
// the directory to show next, or — when a file row was clicked — the
// selected file path (and the directory is left as it was).
func pickerClick(currentDir string, entries []fileEntry, id int) (newDir, selectedFile string) {
	if id == 0 {
		return filepath.Dir(currentDir), ""
	}
	e := entries[id-1]
	full := filepath.Join(currentDir, e.name)
	if e.isDir {
		return full, ""
	}
	return currentDir, full
}

// pickerResult is what confirming the picker returns: the open folder in
// folder mode, or the selected file in file mode (nothing if none was
// picked).
func pickerResult(mode pickerMode, currentDir, selected string) (path string, ok bool) {
	if mode == pickDir {
		return currentDir, true
	}
	return selected, selected != ""
}

// dirsOnly filters a listDir() result down to directories — showDirPicker
// has no use for files. Pure function for testing.
func dirsOnly(entries []fileEntry) []fileEntry {
	dirs := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		if e.isDir {
			dirs = append(dirs, e)
		}
	}
	return dirs
}

type fileEntry struct {
	name  string
	isDir bool
}

// fileEntryLabel is the pure formatting rule for one list row: index 0 is
// always the ".." parent-directory entry; everything else maps to
// entries[id-1] with a folder/file icon.
func fileEntryLabel(id widget.ListItemID, entries []fileEntry) string {
	if id == 0 {
		return "📁 .."
	}
	e := entries[id-1]
	if e.isDir {
		return "📁 " + e.name
	}
	return "📄 " + e.name
}

// listDir reads one directory's entries, already sorted (folders first,
// then alphabetical within each group). Returns nil (not an error) on any
// read failure — an unreadable directory just shows as empty rather than
// crashing the picker.
func listDir(dir string) []fileEntry {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	entries := make([]fileEntry, 0, len(des))
	for _, d := range des {
		entries = append(entries, fileEntry{name: d.Name(), isDir: d.IsDir()})
	}
	sortFileEntries(entries)
	return entries
}

// sortFileEntries orders folders before files, alphabetically within each
// group. Pulled out as a pure function (no os I/O) for testing.
func sortFileEntries(entries []fileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return entries[i].name < entries[j].name
	})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
