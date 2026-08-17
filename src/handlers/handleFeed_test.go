package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeedRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleFeed)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestFeedCountValidation(t *testing.T) {
	tests := []struct {
		name  string
		count string
	}{
		{name: "zero", count: "0"},
		{name: "negative", count: "-1"},
		{name: "too large", count: "51"},
		{name: "not a number", count: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newAuthedRequest(t, "") // valid token, body unused
			req.URL.RawQuery = "count=" + tt.count
			rec := httptest.NewRecorder()

			WithAuth(http.HandlerFunc(HandleFeed)).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}
