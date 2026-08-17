package handlers

import (
	"backend/src/auth"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// init ensures JWT_SECRET is set before any test calls into auth.GenerateToken.
func init() {
	os.Setenv("JWT_SECRET", "test-secret")
}

// newAuthedRequest builds a POST /stream request with a valid bearer token.
func newAuthedRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	token, err := auth.GenerateToken(uuid.New(), "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestCreateStreamRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(`{"topic":"Go"}`))
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleCreateStream)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCreateStreamValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: `{"topic":`},
		{name: "empty topic", body: `{"topic":"   "}`},
		{name: "oversized topic", body: `{"topic":"` + strings.Repeat("a", 256) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthedRequest(t, tt.body)
			rec := httptest.NewRecorder()

			WithAuth(http.HandlerFunc(HandleCreateStream)).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}
