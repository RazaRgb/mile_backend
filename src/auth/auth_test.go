package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// init ensures JWT_SECRET is set before any test calls into jwtSecret().
func init() {
	os.Setenv("JWT_SECRET", "test-secret")
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Fatal("hash must not be plaintext")
	}
	if !VerifyPassword(hash, "correct-horse-battery-staple") {
		t.Fatal("correct password should verify")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("wrong password must not verify")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	userID := uuid.New()
	token, err := GenerateToken(userID, "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("subject = %q, want %q", claims.Subject, userID.String())
	}
	if claims.Email != "user@example.com" {
		t.Errorf("email = %q, want %q", claims.Email, "user@example.com")
	}
	if claims.UserID() != userID {
		t.Errorf("UserID() = %v, want %v", claims.UserID(), userID)
	}
}

func TestParseTokenRejectsTamperedToken(t *testing.T) {
	token, err := GenerateToken(uuid.New(), "a@b.com")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if _, err := ParseToken(token + "tampered"); err == nil {
		t.Fatal("tampered token should be rejected")
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	claims := Claims{
		Email: "a@b.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // already expired
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret())
	if err != nil {
		t.Fatalf("sign expired token failed: %v", err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Fatal("expired token should be rejected")
	}
}

func TestParseTokenRejectsWrongSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{Email: "a@b.com"})
	bad, err := token.SignedString(jwtSecret())
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if _, err := ParseToken(bad); err == nil {
		t.Fatal("HS512 token should be rejected by HS256-only parser")
	}
}
