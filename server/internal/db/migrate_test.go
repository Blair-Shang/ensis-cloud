package db

import (
	"strings"
	"testing"
)

func TestSplitSQL(t *testing.T) {
	src := `
-- comment
CREATE TABLE a (id INT);

CREATE TABLE b (id INT);
`
	got := splitSQL(src)
	if len(got) != 2 {
		t.Fatalf("got %d stmts: %#v", len(got), got)
	}
	if !strings.Contains(got[0], "CREATE TABLE a") {
		t.Fatalf("stmt0=%q", got[0])
	}
	if !strings.Contains(got[1], "CREATE TABLE b") {
		t.Fatalf("stmt1=%q", got[1])
	}
}
