package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStreamTreeRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stream/00000000-0000-0000-0000-000000000000/tree", nil)
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleGetStreamTree)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetStreamTreeInvalidID(t *testing.T) {
	req := newAuthedRequest(t, "")
	req.URL.Path = "/stream/not-a-uuid/tree"
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleGetStreamTree)).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
