package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Blair-Shang/ensis-cloud/server/internal/config"
	"github.com/Blair-Shang/ensis-cloud/server/internal/db"
	"github.com/Blair-Shang/ensis-cloud/server/internal/releasefiles"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

func testUpdateServer(t *testing.T, publicBase string) (*Server, *store.Store, *releasefiles.Store) {
	t.Helper()
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "cloud.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	relDir := filepath.Join(t.TempDir(), "releases")
	s := New(config.Config{
		PublicBase:     publicBase,
		ReleasesDir:    relDir,
		TrustProxy:     false,
		JWTSecret:      "test-jwt-secret-16",
		AccessTokenTTL: time.Minute,
	}, st, nil, nil)
	return s, st, s.files
}

func insertPublished(t *testing.T, st *store.Store, version, downloadURL string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rel := &store.AppRelease{
		ID:          "rel_" + strings.ReplaceAll(version, ".", ""),
		Product:     "niuma",
		Version:     version,
		Channel:     "stable",
		Platform:    "windows",
		Arch:        "x64",
		Title:       "NiuMa " + version,
		DownloadURL: downloadURL,
		SHA256:      strings.Repeat("ab", 32),
		Status:      "published",
		PublishedAt: &now,
		CreatedBy:   "test",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := st.InsertRelease(t.Context(), rel); err != nil {
		t.Fatal(err)
	}
}

func TestHandleUpdatesDownloadLatestRedirect(t *testing.T) {
	s, st, files := testUpdateServer(t, "https://www.niuma007.com/niuma/cloud")
	d := releasefiles.Dim{
		Product: "niuma", Channel: "stable", Platform: "windows",
		Arch: "x64", Version: "1.2.3", Name: "NiuMa-1.2.3-windows-x64-Setup.exe",
	}
	if _, _, err := files.Save(d, strings.NewReader("setup-bytes"), 1024, ""); err != nil {
		t.Fatal(err)
	}
	insertPublished(t, st, "1.0.0", "https://www.niuma007.com/old.exe")
	insertPublished(t, st, "1.2.3", releasefiles.PublicURL("https://www.niuma007.com/niuma/cloud", d))

	req := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/download?platform=windows&arch=x64", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/api/v1/updates/files/niuma/stable/windows/x64/1.2.3/NiuMa-1.2.3-windows-x64-Setup.exe") {
		t.Fatalf("location=%s", loc)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("cache=%s", rec.Header().Get("Cache-Control"))
	}
	n, err := st.CountUpdateHits(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("hits=%d", n)
	}
}

func TestHandleUpdatesDownloadFallbackURL(t *testing.T) {
	s, st, _ := testUpdateServer(t, "")
	insertPublished(t, st, "2.0.0", "https://cdn.niuma007.com/NiuMa-2.0.0-windows-x64-Setup.exe")
	s.cfg.DownloadURLHosts = []string{"cdn.niuma007.com"}

	req := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/download?product=niuma&channel=stable&platform=windows&arch=x64", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "https://cdn.niuma007.com/NiuMa-2.0.0-windows-x64-Setup.exe" {
		t.Fatalf("location=%s", loc)
	}
}

func TestHandleUpdatesHitStats(t *testing.T) {
	s, st, _ := testUpdateServer(t, "")
	if _, err := st.IncrementUpdateHit(t.Context(), "niuma", "stable", "windows", "x64", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.IncrementUpdateHit(t.Context(), "niuma", "beta", "linux", "x64", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/hits?product=niuma", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"total":2`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleUpdatesDownloadNotFound(t *testing.T) {
	s, _, _ := testUpdateServer(t, "")
	req := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/download?platform=windows&arch=x64", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandleUpdatesDownloadRejectsBadQuery(t *testing.T) {
	s, _, _ := testUpdateServer(t, "")
	req := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/download", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestLatestDownloadLocationRelative(t *testing.T) {
	s := &Server{cfg: config.Config{}, files: releasefiles.New(t.TempDir())}
	rel := &store.AppRelease{
		Product: "niuma", Channel: "stable", Platform: "windows", Arch: "x64", Version: "1.0.1",
		DownloadURL: "https://www.niuma007.com/niuma/cloud/api/v1/updates/files/niuma/stable/windows/x64/1.0.1/NiuMa-1.0.1-windows-x64-Setup.exe",
	}
	if loc := s.latestDownloadLocation(rel); loc != rel.DownloadURL {
		t.Fatalf("no local file: %s", loc)
	}
	d := releasefiles.Dim{
		Product: "niuma", Channel: "stable", Platform: "windows",
		Arch: "x64", Version: "1.0.1", Name: "NiuMa-1.0.1-windows-x64-Setup.exe",
	}
	if _, _, err := s.files.Save(d, strings.NewReader("x"), 64, ""); err != nil {
		t.Fatal(err)
	}
	loc := s.latestDownloadLocation(rel)
	if loc != "/niuma/cloud/api/v1/updates/files/niuma/stable/windows/x64/1.0.1/NiuMa-1.0.1-windows-x64-Setup.exe" {
		t.Fatalf("relative=%s", loc)
	}
}

func TestSafeDownloadRedirect(t *testing.T) {
	s := &Server{}
	if got := s.safeDownloadRedirect("http://evil.example/setup.exe"); got != "" {
		t.Fatalf("http: %s", got)
	}
	if got := s.safeDownloadRedirect("https://evil.example/setup.exe"); got != "https://evil.example/setup.exe" {
		t.Fatalf("https open host without allowlist: %s", got)
	}
	s.cfg.DownloadURLHosts = []string{"niuma007.com"}
	if got := s.safeDownloadRedirect("https://evil.example/setup.exe"); got != "" {
		t.Fatalf("allowlist: %s", got)
	}
	rel := "/niuma/cloud/api/v1/updates/files/niuma/stable/windows/x64/1.0.0/NiuMa-1.0.0-windows-x64-Setup.exe"
	if got := s.safeDownloadRedirect(rel); got != rel {
		t.Fatalf("managed relative: %s", got)
	}
}

func TestOldAndNewDesktopSeeEnsisOpsRelease(t *testing.T) {
	s, st, _ := testUpdateServer(t, "https://www.niuma007.com/niuma/cloud")
	insertPublished(t, st, "1.0.0", "https://cdn.niuma007.com/NiuMa-1.0.0-windows-x64-Setup.exe")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := st.InsertRelease(t.Context(), &store.AppRelease{
		ID: "rel_ensis200", Product: "ensis-ops", Version: "2.0.0", Channel: "stable",
		Platform: "windows", Arch: "x64", Title: "ensis-ops 2.0.0",
		DownloadURL: "https://cdn.niuma007.com/ensis-ops-2.0.0-windows-x64-Setup.exe",
		SHA256:      strings.Repeat("cd", 32), Status: "published", PublishedAt: &now,
		CreatedBy: "test", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	oldReq := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/check?current=1.0.0&platform=windows&arch=x64", nil)
	oldRec := httptest.NewRecorder()
	s.Handler().ServeHTTP(oldRec, oldReq)
	if oldRec.Code != http.StatusOK || !strings.Contains(oldRec.Body.String(), `"version":"2.0.0"`) || !strings.Contains(oldRec.Body.String(), `"product":"ensis-ops"`) {
		t.Fatalf("old client status=%d body=%s", oldRec.Code, oldRec.Body.String())
	}

	newReq := httptest.NewRequest(http.MethodGet, "/niuma/cloud/api/v1/updates/check?product=ensis-ops&current=1.0.0&platform=windows&arch=x64", nil)
	newRec := httptest.NewRecorder()
	s.Handler().ServeHTTP(newRec, newReq)
	if newRec.Code != http.StatusOK || !strings.Contains(newRec.Body.String(), `"updateAvailable":true`) || !strings.Contains(newRec.Body.String(), `"product":"ensis-ops"`) {
		t.Fatalf("new client status=%d body=%s", newRec.Code, newRec.Body.String())
	}
}
