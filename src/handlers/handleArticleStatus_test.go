package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArticleStatusRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/article/00000000-0000-0000-0000-000000000000/status", strings.NewReader(`{"status":"watched"}`))
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleArticleStatus)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestArticleStatusValidation(t *testing.T) {
	// The valid-status cases hit the DB, so they are covered by the live e2e
	// test instead of unit tests (which have no database).
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "invalid node id", path: "/article/not-a-uuid/status", body: `{"status":"watched"}`},
		{name: "invalid json", path: "/article/00000000-0000-0000-0000-000000000000/status", body: `{`},
		{name: "unknown status", path: "/article/00000000-0000-0000-0000-000000000000/status", body: `{"status":"maybe"}`},
		{name: "missing status", path: "/article/00000000-0000-0000-0000-000000000000/status", body: `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthedRequest(t, tt.body)
			req.URL.Path = tt.path
			rec := httptest.NewRecorder()

			WithAuth(http.HandlerFunc(HandleArticleStatus)).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}
