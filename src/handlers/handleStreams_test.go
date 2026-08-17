package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListStreamsRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/streams", nil)
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleListStreams)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestDeleteStreamRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/stream/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleDeleteStream)).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestDeleteStreamInvalidID(t *testing.T) {
	req := newAuthedRequest(t, "")
	req.Method = http.MethodDelete
	req.URL.Path = "/stream/not-a-uuid"
	rec := httptest.NewRecorder()

	WithAuth(http.HandlerFunc(HandleDeleteStream)).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
