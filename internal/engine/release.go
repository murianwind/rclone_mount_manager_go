package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// defaultAPIClient has a bounded timeout, unlike http.DefaultClient
// (Timeout: 0 / unlimited) — a stalled GitHub API response would otherwise
// hang the calling goroutine forever with no error.
var defaultAPIClient = &http.Client{Timeout: 15 * time.Second}

// RcloneReleaseAPI / AppReleaseAPI mirror the two GitHub "latest release"
// endpoints polled by _check_versions_async(): rclone's wiserain fork, and
// this app's own repo.
const (
	RcloneReleaseAPI = "https://api.github.com/repos/wiserain/rclone/releases/latest"
	AppReleaseAPI    = "https://api.github.com/repos/Murianwind/rclone_mount_manager_go/releases/latest"
)

// githubRelease is the subset of the GitHub "latest release" response this
// package needs.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// FetchLatestReleaseTag calls a GitHub "releases/latest"-shaped API URL
// and returns the tag name with any leading "v" stripped (e.g. "v1.68.2"
// -> "1.68.2"). client may be nil to use http.DefaultClient.
func FetchLatestReleaseTag(client *http.Client, apiURL string) (string, error) {
	if client == nil {
		client = defaultAPIClient
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, apiURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", err
	}
	return strings.TrimPrefix(rel.TagName, "v"), nil
}

// ReleaseAsset is one downloadable file attached to a GitHub release.
type ReleaseAsset struct {
	Name        string
	DownloadURL string
}

// Release is the subset of a GitHub "latest release" response needed to
// check for and download an app update: the version, and its assets.
type Release struct {
	Version string
	Body    string
	Assets  []ReleaseAsset
}

type githubReleaseFull struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// FetchLatestRelease is like FetchLatestReleaseTag but also returns the
// release's downloadable assets, needed to locate the app's own update
// package (e.g. "RcloneManager.zip").
func FetchLatestRelease(client *http.Client, apiURL string) (Release, error) {
	if client == nil {
		client = defaultAPIClient
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, apiURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Release{}, err
	}

	var full githubReleaseFull
	if err := json.Unmarshal(body, &full); err != nil {
		return Release{}, err
	}

	rel := Release{Version: strings.TrimPrefix(full.TagName, "v"), Body: full.Body}
	for _, a := range full.Assets {
		rel.Assets = append(rel.Assets, ReleaseAsset{Name: a.Name, DownloadURL: a.BrowserDownloadURL})
	}
	return rel, nil
}

var rcloneVersionOutputRe = regexp.MustCompile(`rclone v([\d.\-]+)`)

// ParseLocalRcloneVersion extracts the version string from `rclone
// version`'s stdout (e.g. "rclone v1.74.4-302\n- os/version: ..." ->
// "1.74.4-302"). The hyphenated build suffix used by the wiserain fork is
// kept, matching the comment in _check_versions_async: capturing the full
// string keeps the displayed version aligned with the GitHub tag, while
// VerTuple() is what actually strips/compares the build number.
func ParseLocalRcloneVersion(rcloneVersionOutput string) (version string, ok bool) {
	m := rcloneVersionOutputRe.FindStringSubmatch(rcloneVersionOutput)
	if m == nil {
		return "", false
	}
	return m[1], true
}
