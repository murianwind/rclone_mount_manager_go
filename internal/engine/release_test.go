package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestReleaseTag(t *testing.T) {
	Scenario(t, "GIVEN a release API returning a v-prefixed tag WHEN fetched THEN the v prefix is stripped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"tag_name": "v1.68.2"}`))
		}))
		defer srv.Close()

		got, err := FetchLatestReleaseTag(srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "1.68.2" {
			t.Errorf("FetchLatestReleaseTag = %q, want %q", got, "1.68.2")
		}
	})

	// Negative case: mirrors what a private repo (no auth token) or a
	// rate-limited GitHub API looks like — a non-200 status with no body
	// worth parsing.
	Scenario(t, "GIVEN the release API returns 403 (e.g. rate-limited or a private repo with no auth) WHEN fetched THEN it returns an error, not a crash or empty success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		if _, err := FetchLatestReleaseTag(srv.Client(), srv.URL); err == nil {
			t.Errorf("expected an error on non-200 status")
		}
	})
}

func TestFetchLatestRelease(t *testing.T) {
	Scenario(t, "GIVEN a release with a download asset WHEN fetched THEN both the version and the asset URL are parsed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"tag_name": "v2.1.0",
				"body": "- 뭔가 고쳤습니다\n- 뭔가 더 고쳤습니다",
				"assets": [
					{"name": "RcloneManager.zip", "browser_download_url": "https://example.com/RcloneManager.zip"}
				]
			}`))
		}))
		defer srv.Close()

		rel, err := FetchLatestRelease(srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rel.Version != "2.1.0" {
			t.Errorf("Version = %q, want %q", rel.Version, "2.1.0")
		}
		if rel.Body == "" {
			t.Errorf("release body should have been parsed but is empty")
		}
		if len(rel.Assets) != 1 || rel.Assets[0].Name != "RcloneManager.zip" {
			t.Fatalf("unexpected assets: %+v", rel.Assets)
		}
		if rel.Assets[0].DownloadURL != "https://example.com/RcloneManager.zip" {
			t.Errorf("unexpected download URL: %q", rel.Assets[0].DownloadURL)
		}
	})
}

func TestParseLocalRcloneVersion(t *testing.T) {
	Scenario(t, "GIVEN rclone version output with a wiserain build suffix WHEN parsed THEN the full version+build string is extracted", func(t *testing.T) {
		output := "rclone v1.74.4-302\n- os/version: windows 10\n- os/kernel: ...\n"
		got, ok := ParseLocalRcloneVersion(output)
		if !ok {
			t.Fatalf("expected a match")
		}
		if got != "1.74.4-302" {
			t.Errorf("ParseLocalRcloneVersion = %q, want %q", got, "1.74.4-302")
		}
	})

	// Negative case: e.g. rclone.exe missing/corrupt and the "version"
	// command produced garbage instead of real output.
	Scenario(t, "GIVEN output that isn't a version string at all WHEN parsed THEN it reports no match rather than a wrong guess", func(t *testing.T) {
		if _, ok := ParseLocalRcloneVersion("not a version string"); ok {
			t.Errorf("expected no match")
		}
	})
}
