package engine

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// buildFakeRcloneZip은 실제 wiserain 릴리스 자산과 같은 구조
// ("rclone-v<버전>-windows-amd64/rclone.exe")의 zip을 만든다.
func buildFakeRcloneZip(t *testing.T, exeContent []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("rclone-v1.65.0-windows-amd64/rclone.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(exeContent); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// givenDownloadURLRedirectsTo는 rcloneDownloadURLFn을 테스트 서버로
// 임시 교체한다 (실제 GitHub 대신 httptest 서버를 쓰기 위한 대역).
func givenDownloadURLRedirectsTo(t *testing.T, url string) {
	t.Helper()
	orig := rcloneDownloadURLFn
	rcloneDownloadURLFn = func(version string) string { return url }
	t.Cleanup(func() { rcloneDownloadURLFn = orig })
}

func TestDownloadRclone(t *testing.T) {
	Scenario(t, "GIVEN 정상적인 rclone.exe가 든 zip WHEN 다운로드 THEN rclone.exe로 바로 설치된다", func(t *testing.T) {
		zipData := buildFakeRcloneZip(t, []byte("fake_exe"))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(zipData)
		}))
		defer srv.Close()
		givenDownloadURLRedirectsTo(t, srv.URL)

		destDir := t.TempDir()
		status, err := DownloadRclone(srv.Client(), destDir, "1.65.0")
		thenNoError(t, err)
		if status != DownloadStatusInstalled {
			t.Errorf("상태 = %q, 기대값 %q", status, DownloadStatusInstalled)
		}

		data, err := os.ReadFile(filepath.Join(destDir, "rclone.exe"))
		if err != nil {
			t.Fatalf("rclone.exe가 기록됐어야 함: %v", err)
		}
		if string(data) != "fake_exe" {
			t.Errorf("rclone.exe 내용 = %q, 기대값 %q", data, "fake_exe")
		}
	})

	// 부정 케이스: 실행 중인 마운트 때문에 rclone.exe가 잠겨있어 바로
	// 교체할 수 없는 상황 — Python 버전의 PermissionError 폴백에 대응.
	Scenario(t, "GIVEN 대상 rclone.exe 자리가 잠겨있어 직접 쓸 수 없음 WHEN 다운로드 THEN rclone_new.exe로 대신 저장하고 수동 교체 상태를 반환한다 (부정 케이스)", func(t *testing.T) {
		zipData := buildFakeRcloneZip(t, []byte("fake_exe_v2"))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(zipData)
		}))
		defer srv.Close()
		givenDownloadURLRedirectsTo(t, srv.URL)

		destDir := t.TempDir()
		// rclone.exe 자리에 디렉터리를 만들어, "지금은 교체 불가"
		// 상황을 OS/사용자 권한과 무관하게 재현한다.
		if err := os.Mkdir(filepath.Join(destDir, "rclone.exe"), 0o755); err != nil {
			t.Fatal(err)
		}

		status, err := DownloadRclone(srv.Client(), destDir, "1.65.0")
		thenNoError(t, err)
		if status != DownloadStatusManual {
			t.Errorf("상태 = %q, 기대값 %q", status, DownloadStatusManual)
		}

		data, err := os.ReadFile(filepath.Join(destDir, "rclone_new.exe"))
		if err != nil {
			t.Fatalf("rclone_new.exe 폴백 파일이 기록됐어야 함: %v", err)
		}
		if string(data) != "fake_exe_v2" {
			t.Errorf("rclone_new.exe 내용 = %q, 기대값 %q", data, "fake_exe_v2")
		}
	})

	// 부정 케이스: 다운로드한 zip 안에 정작 rclone.exe가 없는 손상/변조 상황.
	Scenario(t, "GIVEN 다운로드한 zip 안에 rclone.exe가 없음 WHEN 다운로드 THEN 오류를 반환한다 (부정 케이스)", func(t *testing.T) {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		_, _ = zw.Create("readme.txt")
		zw.Close()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(buf.Bytes())
		}))
		defer srv.Close()
		givenDownloadURLRedirectsTo(t, srv.URL)

		if _, err := DownloadRclone(srv.Client(), t.TempDir(), "1.65.0"); err == nil {
			t.Errorf("zip에 rclone.exe가 없으면 오류가 나야 함")
		}
	})
}
