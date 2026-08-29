package db

import "testing"

func TestSplitMySQLDSN(t *testing.T) {
	userHost, dbName, query, ok := splitMySQLDSN(
		`root:datahub@tcp(121.43.231.91:63306)/niuma_cloud?parseTime=true&charset=utf8mb4&loc=UTC`,
	)
	if !ok {
		t.Fatal("expected ok")
	}
	if userHost != `root:datahub@tcp(121.43.231.91:63306)` {
		t.Fatalf("userHost=%q", userHost)
	}
	if dbName != "niuma_cloud" {
		t.Fatalf("dbName=%q", dbName)
	}
	if query != "?parseTime=true&charset=utf8mb4&loc=UTC" {
		t.Fatalf("query=%q", query)
	}
}
