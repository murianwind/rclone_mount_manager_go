package engine

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultDownloadClient has a bounded timeout — http.DefaultClient's
// Timeout is 0 (unlimited), which meant a stalled connection would hang
// this call forever with no error and nothing to log. 2 minutes is
// generous for a few-MB exe/zip even on a slow connection.
var defaultDownloadClient = &http.Client{Timeout: 2 * time.Minute}

// Download status values mirror download_rclone()'s return convention
// (True / "manual" / error string), just as Go constants + an error
// instead of Python's mixed-type return.
const (
	DownloadStatusInstalled = "installed" // replaced rclone.exe directly
	DownloadStatusManual    = "manual"    // saved alongside; needs manual swap
)

// rcloneDownloadURLFn builds the wiserain fork release asset URL for a
// given version. It's a package var (not a plain function) so tests can
// redirect it at an httptest server instead of the real GitHub releases
// CDN.
var rcloneDownloadURLFn = func(version string) string {
	return fmt.Sprintf(
		"https://github.com/wiserain/rclone/releases/download/v%s/rclone-v%s-windows-amd64.zip",
		version, version)
}

// DownloadRclone downloads the given rclone version and installs it as
// rclone.exe in destDir, mirroring download_rclone().
//
// If destDir/rclone.exe can't be written directly — e.g. it's locked by a
// running mount process, the situation the Python version specifically
// handles — the new binary is saved as destDir/rclone_new.exe instead and
// DownloadStatusManual is returned so the caller can prompt the user for a
// manual swap (same behavior as the Python PermissionError fallback).
func DownloadRclone(client *http.Client, destDir, version string) (status string, err error) {
	if client == nil {
		client = defaultDownloadClient
	}

	data, err := httpGetBytes(client, rcloneDownloadURLFn(version))
	if err != nil {
		return "", err
	}

	exeData, err := extractRcloneExeFromZip(data)
	if err != nil {
		return "", err
	}

	target := filepath.Join(destDir, "rclone.exe")
	if werr := os.WriteFile(target, exeData, 0o755); werr != nil {
		fallback := filepath.Join(destDir, "rclone_new.exe")
		if ferr := os.WriteFile(fallback, exeData, 0o755); ferr != nil {
			return "", ferr
		}
		return DownloadStatusManual, nil
	}
	return DownloadStatusInstalled, nil
}

func extractRcloneExeFromZip(data []byte) ([]byte, error) {
	return extractFileFromZip(data, "rclone.exe")
}

func extractFileFromZip(data []byte, nameSuffix string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, nameSuffix) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not found in downloaded archive", nameSuffix)
}

func httpGetBytes(client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = defaultDownloadClient
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
