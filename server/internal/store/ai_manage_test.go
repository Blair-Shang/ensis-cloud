package store_test

import (
	"path/filepath"
	"testing"

	"github.com/Blair-Shang/ensis-cloud/server/internal/db"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

func TestAIEndpointAndModelCRUD(t *testing.T) {
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	ctx := t.Context()

	ep := &store.AIUpstreamEndpoint{
		EndpointID:          "aiend_test",
		ProviderDisplayName: "NiuMa",
		UpstreamBaseURL:     "https://api.example.com/v1",
		APISecretCipher:     "cipher",
		ServiceEnabled:      true,
		DefaultModelCode:    "fast",
		DailyRequestLimit:   10,
		CreatedByStaffID:    "staff_a",
		UpdatedByStaffID:    "staff_a",
	}
	if err := st.InsertAIEndpoint(ctx, ep); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetActiveAIEndpoint(ctx)
	if err != nil || got.EndpointID != "aiend_test" || !got.HasAPISecret {
		t.Fatalf("active: %+v %v", got, err)
	}

	m := &store.AIModelCatalog{
		CatalogID:        "aimdl_test",
		EndpointID:       ep.EndpointID,
		ModelCode:        "fast",
		ModelLabel:       "Fast",
		RecordStatus:     "active",
		CreatedByStaffID: "staff_a",
		UpdatedByStaffID: "staff_a",
	}
	if err := st.InsertAIModel(ctx, m); err != nil {
		t.Fatal(err)
	}
	list, err := st.ListAIModels(ctx, ep.EndpointID, "active")
	if err != nil || len(list) != 1 {
		t.Fatalf("models: %v %v", list, err)
	}
	if err := st.DeleteAIEndpoint(ctx, ep.EndpointID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetAIModel(ctx, m.CatalogID); err != store.ErrNotFound {
		t.Fatalf("cascade: %v", err)
	}
}
