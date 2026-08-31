package cryptutil

import "testing"

func TestSealOpen(t *testing.T) {
	c, err := Seal("master-secret-at-least-16", "sk-test")
	if err != nil {
		t.Fatal(err)
	}
	if c == "" || c == "sk-test" {
		t.Fatalf("cipher: %q", c)
	}
	plain, err := Open("master-secret-at-least-16", c)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "sk-test" {
		t.Fatalf("plain: %q", plain)
	}
}
