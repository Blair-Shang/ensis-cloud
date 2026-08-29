package store

import "testing"

func TestParseAndCompareSemVer(t *testing.T) {
	a, err := ParseSemVer("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseSemVer("v1.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if CompareSemVer(a, b) >= 0 {
		t.Fatalf("expected 1.0.0 < 1.0.1")
	}
	if _, err := ParseSemVer("1.0.0-beta"); err == nil {
		t.Fatal("pre-release should fail")
	}
	if _, err := ParseSemVer("01.0.0"); err == nil {
		t.Fatal("leading zero should fail")
	}
}
