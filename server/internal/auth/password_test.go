package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("ChangeMeAdmin1!")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "ChangeMeAdmin1!") {
		t.Fatal("password should match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password should not match")
	}
}

func TestOTP(t *testing.T) {
	code, err := GenerateOTP()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("want 6 digits, got %q", code)
	}
	if HashOTP(code) == code {
		t.Fatal("otp hash should differ from plaintext")
	}
}
