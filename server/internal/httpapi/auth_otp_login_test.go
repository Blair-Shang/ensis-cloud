package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/config"
	"github.com/Blair-Shang/niuma-cloud/server/internal/db"
	"github.com/Blair-Shang/niuma-cloud/server/internal/mail"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

func testAuthServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	sqlDB, err := db.Open("sqlite", filepath.Join(t.TempDir(), "cloud.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	st := store.New(sqlDB, "sqlite")
	cfg := config.Config{
		JWTSecret:       "test-jwt-secret-16",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		OTPTTL:          5 * time.Minute,
		OTPDailyLimit:   20,
	}
	s := New(cfg, st, mail.New(cfg, zap.NewNop()), zap.NewNop())
	return s, st
}

func insertLoginOTP(t *testing.T, st *store.Store, email, code string) {
	t.Helper()
	now := time.Now().UTC()
	if err := st.InsertOTP(t.Context(), store.OTP{
		ID:        "otp_test_" + strings.ReplaceAll(email, "@", "_"),
		Email:     email,
		Purpose:   "login",
		CodeHash:  auth.HashOTP(code),
		ExpiresAt: now.Add(5 * time.Minute).Format(time.RFC3339Nano),
		CreatedAt: now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestLoginOTPStartSendsForUnknownEmail(t *testing.T) {
	s, st := testAuthServer(t)
	email := "new-user@example.com"
	req := httptest.NewRequest(http.MethodPost, "/otp/start", strings.NewReader(`{"email":"`+email+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handleLoginOTPStart(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	n, err := st.CountOTPSince(t.Context(), email, "login", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 otp stored for unknown email, got %d", n)
	}
}

func TestLoginOTPCompleteCreatesUser(t *testing.T) {
	s, st := testAuthServer(t)
	email := "otp-new@example.com"
	insertLoginOTP(t, st, email, "123456")

	req := httptest.NewRequest(http.MethodPost, "/otp/complete", strings.NewReader(`{"email":"`+email+`","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handleLoginOTPComplete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var body struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			Email       string `json:"email"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AccessToken == "" {
		t.Fatal("missing access token")
	}
	if body.User.Email != email {
		t.Fatalf("email %q", body.User.Email)
	}
	if body.User.DisplayName != "otp-new" {
		t.Fatalf("displayName %q", body.User.DisplayName)
	}

	u, err := st.GetUserByEmail(t.Context(), email)
	if err != nil {
		t.Fatal(err)
	}
	if u.Status != "active" || u.EmailVerifiedAt == nil {
		t.Fatalf("user not verified/active: %+v", u)
	}
	if auth.CheckPassword(u.PasswordHash, "123456") {
		t.Fatal("otp-provisioned user must not use the code as password")
	}

	login := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"`+email+`","password":"whatever1x"}`))
	login.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	s.handleLoginPassword(loginRec, login)
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("password login should fail, got %d %s", loginRec.Code, loginRec.Body.String())
	}
}

func TestLoginOTPCompleteExistingUser(t *testing.T) {
	s, st := testAuthServer(t)
	email := "otp-old@example.com"
	hash, err := auth.HashPassword("ChangeMeAdmin1!")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := st.CreateUser(t.Context(), store.User{
		ID:           "usr_existing",
		Email:        email,
		PasswordHash: hash,
		DisplayName:  "Old",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
	insertLoginOTP(t, st, email, "654321")

	req := httptest.NewRequest(http.MethodPost, "/otp/complete", strings.NewReader(`{"email":"`+email+`","code":"654321"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handleLoginOTPComplete(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	n, err := st.CountUsers(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("must not create a second user, got %d", n)
	}
	u, err := st.GetUserByEmail(t.Context(), email)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != "usr_existing" || u.DisplayName != "Old" {
		t.Fatalf("mutated existing user: %+v", u)
	}
}
