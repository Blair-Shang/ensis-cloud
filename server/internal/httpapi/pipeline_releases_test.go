package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Blair-Shang/niuma-cloud/server/internal/config"
	"github.com/Blair-Shang/niuma-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

func TestSecureTokenEqual(t *testing.T) {
	if !secureTokenEqual("same-secret-token", "same-secret-token") {
		t.Fatal("equal tokens")
	}
	if secureTokenEqual("same-secret-token", "other-secret-token") {
		t.Fatal("different tokens")
	}
	if secureTokenEqual("same-secret-token", "") {
		t.Fatal("empty got")
	}
}

func TestPipelineReleasesAuth(t *testing.T) {
	s := &Server{cfg: config.Config{}}
	req := httptest.NewRequest(http.MethodPost, "/niuma/cloud/api/v1/pipeline/releases", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	s.handlePipelineReleases(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("empty token: got %d", rec.Code)
	}

	s.cfg.PipelineToken = "pipeline-token-16"
	req = httptest.NewRequest(http.MethodPost, "/niuma/cloud/api/v1/pipeline/releases", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	s.handlePipelineReleases(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing header: got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/niuma/cloud/api/v1/pipeline/releases", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer wrong-token-xxxx")
	rec = httptest.NewRecorder()
	s.handlePipelineReleases(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong bearer: got %d", rec.Code)
	}
}

func TestPipelineAssetAuth(t *testing.T) {
	s := &Server{cfg: config.Config{}}
	req := httptest.NewRequest(http.MethodPost, "/niuma/cloud/api/v1/pipeline/releases/assets", nil)
	rec := httptest.NewRecorder()
	s.handlePipelineReleaseAsset(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("empty token: got %d", rec.Code)
	}

	s.cfg.PipelineToken = "pipeline-token-16"
	req = httptest.NewRequest(http.MethodPost, "/niuma/cloud/api/v1/pipeline/releases/assets", nil)
	rec = httptest.NewRecorder()
	s.handlePipelineReleaseAsset(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing header: got %d", rec.Code)
	}
}

func TestAttachLocalInstaller(t *testing.T) {
	root := t.TempDir()
	files := releasefiles.New(root)
	s := &Server{
		cfg:   config.Config{PublicBase: "https://www.niuma007.com/niuma/cloud"},
		files: files,
	}
	rel := &store.AppRelease{
		Product: "niuma", Channel: "stable", Platform: "windows", Arch: "x64", Version: "1.0.2",
	}
	if err := s.attachLocalInstaller(rel, ""); err == nil || err.Error() != "missing_installer" {
		t.Fatalf("want missing_installer, got %v", err)
	}

	d := releasefiles.Dim{
		Product: "niuma", Channel: "stable", Platform: "windows",
		Arch: "x64", Version: "1.0.2", Name: "NiuMa-1.0.2-windows-x64-Setup.exe",
	}
	if _, _, err := files.Save(d, strings.NewReader("installer-bytes"), 1024, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.attachLocalInstaller(rel, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rel.DownloadURL, "/api/v1/updates/files/niuma/stable/windows/x64/1.0.2/") {
		t.Fatalf("url=%s", rel.DownloadURL)
	}
	if rel.FileSize != int64(len("installer-bytes")) {
		t.Fatalf("size=%d", rel.FileSize)
	}
}
