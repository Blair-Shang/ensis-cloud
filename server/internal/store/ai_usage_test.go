package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Blair-Shang/niuma-cloud/server/internal/db"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

func TestTryAddAIUsageCap(t *testing.T) {
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "usage.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	ctx := t.Context()
	at := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	got, err := st.TryAddAIUsage(ctx, "user_a", 1, 0, 2, 0, at)
	if err != nil || got.RequestCount != 1 {
		t.Fatalf("first: %+v %v", got, err)
	}
	got, err = st.TryAddAIUsage(ctx, "user_a", 1, 0, 2, 0, at)
	if err != nil || got.RequestCount != 2 {
		t.Fatalf("second: %+v %v", got, err)
	}
	if _, err = st.TryAddAIUsage(ctx, "user_a", 1, 0, 2, 0, at); err != store.ErrQuotaExceeded {
		t.Fatalf("third: %v", err)
	}
	used, err := st.GetAIUsage(ctx, "user_a", at)
	if err != nil || used.RequestCount != 2 {
		t.Fatalf("stuck at cap: %+v %v", used, err)
	}
}

func TestListAIUsage(t *testing.T) {
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "usage-list.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	ctx := t.Context()
	at := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	now := at.Format(time.RFC3339Nano)

	if err := st.CreateUser(ctx, store.User{
		ID: "user_a", Email: "a@example.com", PasswordHash: "x",
		DisplayName: "Alice", Status: "active", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(ctx, store.User{
		ID: "user_b", Email: "b@example.com", PasswordHash: "x",
		DisplayName: "Bob", Status: "active", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.TryAddAIUsage(ctx, "user_a", 3, 0, 80, 0, at); err != nil {
		t.Fatal(err)
	}
	if _, err := st.TryAddAIUsage(ctx, "user_b", 1, 0, 80, 0, at); err != nil {
		t.Fatal(err)
	}

	sum, err := st.SummarizeAIUsage(ctx, "2026-08-31")
	if err != nil || sum.UserCount != 2 || sum.RequestCount != 4 {
		t.Fatalf("summary: %+v %v", sum, err)
	}

	items, total, err := st.ListAIUsage(ctx, store.AIUsageListFilter{Day: "2026-08-31"})
	if err != nil || total != 2 || len(items) != 2 {
		t.Fatalf("list: %d %+v %v", total, items, err)
	}
	if items[0].Email != "a@example.com" || items[0].RequestCount != 3 {
		t.Fatalf("order: %+v", items[0])
	}

	filtered, n, err := st.ListAIUsage(ctx, store.AIUsageListFilter{Day: "2026-08-31", Email: "b@example.com"})
	if err != nil || n != 1 || len(filtered) != 1 || filtered[0].UserID != "user_b" {
		t.Fatalf("email: %d %+v %v", n, filtered, err)
	}

	empty, z, err := st.ListAIUsage(ctx, store.AIUsageListFilter{Day: "2026-08-30"})
	if err != nil || z != 0 || len(empty) != 0 {
		t.Fatalf("other day: %d %+v %v", z, empty, err)
	}

	trend, err := st.ListAIUsageTrend(ctx, "2026-08-31", 3)
	if err != nil || len(trend) != 3 {
		t.Fatalf("trend: %+v %v", trend, err)
	}
	if trend[0].UsageDay != "2026-08-29" || trend[0].RequestCount != 0 {
		t.Fatalf("trend pad: %+v", trend[0])
	}
	if trend[2].UsageDay != "2026-08-31" || trend[2].RequestCount != 4 || trend[2].UserCount != 2 {
		t.Fatalf("trend end: %+v", trend[2])
	}
}
