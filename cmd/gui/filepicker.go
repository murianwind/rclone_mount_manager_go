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

// showFilePicker is a minimal, fully self-built file browser — built with
// the same dialog.ShowCustomConfirm pattern already proven to place its
// buttons correctly elsewhere in this app (see showRemoteSelectDialog).
// Fyne's built-in FileDialog's Cancel/Open button placement has been
// unreliable across several attempts to fix it from the outside, so this
// sidesteps that entirely by not using it.
func (rm *rcloneManager) showFilePicker(title, startDir string, onSelected func(path string)) {
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
		entries = listDir(currentDir)
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
		if id == 0 {
			currentDir = filepath.Dir(currentDir)
			refresh()
			return
		}
		e := entries[id-1]
		full := filepath.Join(currentDir, e.name)
		if e.isDir {
			currentDir = full
			refresh()
			return
		}
		selected = full
	}

	refresh()

	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(480, 320))
	header := container.NewBorder(nil, nil, driveSelect, nil, pathLabel)
	content := container.NewBorder(header, nil, nil, nil, scroll)

	dialog.ShowCustomConfirm(title, "선택", "취소", content, func(ok bool) {
		if ok && selected != "" {
			onSelected(selected)
		}
	}, rm.win)
}

// showDirPicker is showFilePicker's directory-only counterpart: clicking a
// folder navigates into it (same as showFilePicker), but confirming picks
// whatever directory is currently open rather than requiring a file
// click. Used for 캐시 디렉토리, where a folder itself is the answer.
func (rm *rcloneManager) showDirPicker(title, startDir string, onSelected func(path string)) {
	if startDir == "" || !dirExists(startDir) {
		if home, err := os.UserHomeDir(); err == nil {
			startDir = home
		}
	}
	currentDir := startDir

	var entries []fileEntry
	var list *widget.List

	pathLabel := widget.NewLabel(currentDir)
	pathLabel.Wrapping = fyne.TextWrapBreak

	driveSelect := widget.NewSelect(availableDrives(), nil)
	driveSelect.PlaceHolder = "드라이브"

	refresh := func() {
		entries = dirsOnly(listDir(currentDir))
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
		if id == 0 {
			currentDir = filepath.Dir(currentDir)
		} else {
			currentDir = filepath.Join(currentDir, entries[id-1].name)
		}
		refresh()
	}

	refresh()

	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(480, 320))
	header := container.NewBorder(nil, nil, driveSelect, nil, pathLabel)
	content := container.NewBorder(header, nil, nil, nil, scroll)

	dialog.ShowCustomConfirm(title, "이 폴더 선택", "취소", content, func(ok bool) {
		if ok {
			onSelected(currentDir)
		}
	}, rm.win)
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
