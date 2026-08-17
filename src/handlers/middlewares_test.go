package handlers

import (
	"backend/src/auth"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
)

// init ensures JWT_SECRET is set before any test calls into auth.GenerateToken.
func init() {
	os.Setenv("JWT_SECRET", "test-secret")
}

// testHandler echoes the authenticated user's email, or "no auth".
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := UserFromContext(r.Context())
		if !ok {
			http.Error(w, "no auth", http.StatusForbidden)
			return
		}
		http.Error(w, claims.Email, http.StatusOK)
	})
}

func TestWithAuth(t *testing.T) {
	validToken, err := auth.GenerateToken(uuid.New(), "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	tests := []struct {
		name     string
		auth     string // value of the Authorization header
		wantCode int
		wantBody string // empty means don't assert body
	}{
		{name: "missing header", wantCode: http.StatusUnauthorized},
		{name: "invalid token", auth: "Bearer not-a-real-token", wantCode: http.StatusUnauthorized},
		{name: "valid token", auth: "Bearer " + validToken, wantCode: http.StatusOK, wantBody: "user@example.com\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()

			WithAuth(testHandler()).ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
