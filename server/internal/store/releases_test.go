package store_test

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Blair-Shang/ensis-cloud/server/internal/db"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

func TestUpdateReleaseNotesByVersion(t *testing.T) {
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "releases.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	ctx := t.Context()
	now := "2026-08-31T00:00:00Z"
	sha := strings.Repeat("a", 64)

	mustInsert := func(id, platform, arch, version, status, notes string) {
		t.Helper()
		rel := &store.AppRelease{
			ID: id, Product: "niuma", Version: version, Channel: "stable",
			Platform: platform, Arch: arch, Title: "old title", NotesMd: notes,
			DownloadURL: "https://cdn.example.com/" + id + ".exe", SHA256: sha,
			Status: status, CreatedBy: "staff", CreatedAt: now, UpdatedAt: now,
		}
		if err := st.InsertRelease(ctx, rel); err != nil {
			t.Fatal(err)
		}
	}
	mustInsert("rel_win", "windows", "x64", "1.2.0", "published", "old win")
	mustInsert("rel_lin", "linux", "x64", "1.2.0", "draft", "old lin")
	mustInsert("rel_mac", "macos", "arm64", "1.2.0", "yanked", "old mac")
	mustInsert("rel_old", "windows", "x64", "1.1.0", "published", "other ver")

	title := "1.2.0 稳定性"
	notes := "## 修复\n- 同步说明"
	n, err := st.UpdateReleaseNotesByVersion(ctx, "niuma", "1.2.0", "2026-08-31T01:00:00Z", store.ReleaseNotesPatch{
		Title:   &title,
		NotesMd: &notes,
	})
	if err != nil || n != 2 {
		t.Fatalf("synced %d %v", n, err)
	}

	win, err := st.GetReleaseByID(ctx, "rel_win")
	if err != nil || win.Title != title || win.NotesMd != notes {
		t.Fatalf("windows: %+v %v", win, err)
	}
	lin, err := st.GetReleaseByID(ctx, "rel_lin")
	if err != nil || lin.Title != title || lin.NotesMd != notes {
		t.Fatalf("linux: %+v %v", lin, err)
	}
	mac, err := st.GetReleaseByID(ctx, "rel_mac")
	if err != nil || mac.Title != "old title" || mac.NotesMd != "old mac" {
		t.Fatalf("yanked should stay: %+v %v", mac, err)
	}
	old, err := st.GetReleaseByID(ctx, "rel_old")
	if err != nil || old.NotesMd != "other ver" {
		t.Fatalf("other version: %+v %v", old, err)
	}
	if win.DownloadURL == lin.DownloadURL {
		t.Fatal("installer url must stay per kernel")
	}
}

func TestPublicReleaseCardOmitsNotes(t *testing.T) {
	now := "2026-08-31T00:00:00Z"
	card := store.PublicReleaseCard(&store.AppRelease{
		Version: "1.2.0", Title: "fix", NotesMd: strings.Repeat("很长的更新说明", 200),
		DownloadURL: "https://cdn.example.com/a.exe", SHA256: strings.Repeat("a", 64),
		PublishedAt: &now,
	})
	if _, ok := card["notesMd"]; ok {
		t.Fatal("card must not include notesMd")
	}
	if _, ok := card["downloadUrl"]; ok {
		t.Fatal("card must not include downloadUrl")
	}
	if card["version"] != "1.2.0" || card["publishedAt"] != now {
		t.Fatalf("card: %+v", card)
	}
}

func TestListPublishedReleasesClampsLimit(t *testing.T) {
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "releases-clamp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	now := "2026-08-31T00:00:00Z"
	sha := strings.Repeat("b", 64)
	for i, ver := range []string{"1.0.0", "1.0.1", "1.0.2"} {
		rel := &store.AppRelease{
			ID: "rel_c" + string(rune('a'+i)), Product: "niuma", Version: ver, Channel: "stable",
			Platform: "windows", Arch: "x64", Title: ver, NotesMd: "n",
			DownloadURL: "https://cdn.example.com/" + ver + ".exe", SHA256: sha,
			Status: "published", PublishedAt: &now, CreatedBy: "staff", CreatedAt: now, UpdatedAt: now,
		}
		if err := st.InsertRelease(t.Context(), rel); err != nil {
			t.Fatal(err)
		}
	}
	items, total, err := st.ListPublishedReleases(t.Context(), "niuma", "stable", "windows", "x64", 999, 0)
	if err != nil || total != 3 || len(items) != 3 {
		t.Fatalf("clamp 999: n=%d total=%d err=%v", len(items), total, err)
	}
}

func TestNotesExcerpt(t *testing.T) {
	if got, trunc := store.NotesExcerpt(""); got != "" || trunc {
		t.Fatalf("empty: %q %v", got, trunc)
	}
	short := "修复启动崩溃。"
	if got, trunc := store.NotesExcerpt(short); got != short || trunc {
		t.Fatalf("short: %q %v", got, trunc)
	}
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("这是一段更新说明，用来撑过公开摘要上限。\n\n")
	}
	long := b.String()
	got, trunc := store.NotesExcerpt(long)
	if !trunc || !strings.HasSuffix(got, "…") {
		t.Fatalf("long: %q trunc=%v", got, trunc)
	}
	if utf8.RuneCountInString(strings.TrimSuffix(got, "…")) > store.PublicNotesExcerptRunes {
		t.Fatalf("excerpt too long: %d", utf8.RuneCountInString(got))
	}
	pub := store.PublicRelease(&store.AppRelease{NotesMd: long, Version: "1.2.0"})
	if pub["notesTruncated"] != true {
		t.Fatalf("public truncated: %+v", pub["notesTruncated"])
	}
	if pub["notesMd"] != long {
		t.Fatal("public must keep full notesMd")
	}
}

func TestUpdateReleaseNotesByVersionEmptyPatch(t *testing.T) {
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "releases-empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	n, err := st.UpdateReleaseNotesByVersion(t.Context(), "niuma", "1.0.0", "2026-08-31T01:00:00Z", store.ReleaseNotesPatch{})
	if err != nil || n != 0 {
		t.Fatalf("empty patch: %d %v", n, err)
	}
	if _, err := st.UpdateReleaseNotesByVersion(t.Context(), "", "1.0.0", "", store.ReleaseNotesPatch{}); err != store.ErrInvalidRelease {
		t.Fatalf("want invalid: %v", err)
	}
}
