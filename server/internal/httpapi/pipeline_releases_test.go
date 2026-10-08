package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestPipelineReleaseNotesSync(t *testing.T) {
	s, st, _ := testUpdateServer(t, "https://www.niuma007.com/niuma/cloud")
	s.cfg.PipelineToken = "pipeline-token-16"
	ctx := t.Context()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	sha := strings.Repeat("ab", 32)
	insert := func(id, channel, platform, arch, status, notes string) {
		t.Helper()
		rel := &store.AppRelease{
			ID: id, Product: "niuma", Version: "1.2.0", Channel: channel,
			Platform: platform, Arch: arch, Title: "old", NotesMd: notes,
			DownloadURL: "https://cdn.example.com/" + id + ".bin", SHA256: sha,
			Status: status, CreatedBy: "staff", CreatedAt: now, UpdatedAt: now,
		}
		if status == "published" {
			rel.PublishedAt = &now
		}
		if err := st.InsertRelease(ctx, rel); err != nil {
			t.Fatal(err)
		}
	}
	insert("rel_lin", "beta", "linux", "x64", "published", "旧说明")
	insert("rel_mac", "beta", "macos", "arm64", "yanked", "撤回说明")

	post := func(notes string) (int, int) {
		t.Helper()
		body := `{
			"product":"niuma","version":"1.2.0","channel":"stable",
			"title":"NiuMa 1.2.0","notesMd":` + jsonString(notes) + `,
			"publish":true,
			"items":[{
				"platform":"windows","arch":"x64",
				"name":"NiuMa-1.2.0-windows-x64-Setup.exe",
				"downloadUrl":"https://cdn.example.com/niuma-1.2.0-setup.exe",
				"sha256":"` + sha + `","fileSize":12
			}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/niuma/cloud/api/v1/pipeline/releases", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer pipeline-token-16")
		rec := httptest.NewRecorder()
		s.handlePipelineReleases(rec, req)
		var out struct {
			NotesSynced int `json:"notesSynced"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil && rec.Code == http.StatusOK {
			t.Fatal(err)
		}
		return rec.Code, out.NotesSynced
	}
	if code, n := post("## 修复\n- 新说明"); code != http.StatusOK || n != 2 {
		t.Fatalf("sync: code=%d notesSynced=%d", code, n)
	}
	lin, err := st.GetReleaseByID(ctx, "rel_lin")
	if err != nil || lin.NotesMd != "## 修复\n- 新说明" || lin.Title != "old" {
		t.Fatalf("linux notes: %+v %v", lin, err)
	}
	mac, err := st.GetReleaseByID(ctx, "rel_mac")
	if err != nil || mac.NotesMd != "撤回说明" || mac.Title != "old" {
		t.Fatalf("yanked notes: %+v %v", mac, err)
	}
	win, err := st.GetReleaseByDim(ctx, "niuma", "stable", "windows", "x64", "1.2.0")
	if err != nil || win.NotesMd != "## 修复\n- 新说明" {
		t.Fatalf("windows notes: %+v %v", win, err)
	}

	if code, n := post(""); code != http.StatusOK || n != 0 {
		t.Fatalf("empty notes: code=%d notesSynced=%d", code, n)
	}
	lin, err = st.GetReleaseByID(ctx, "rel_lin")
	if err != nil || lin.NotesMd != "## 修复\n- 新说明" {
		t.Fatalf("linux kept: %+v %v", lin, err)
	}
	win, err = st.GetReleaseByDim(ctx, "niuma", "stable", "windows", "x64", "1.2.0")
	if err != nil || win.NotesMd != "## 修复\n- 新说明" {
		t.Fatalf("windows kept: %+v %v", win, err)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
