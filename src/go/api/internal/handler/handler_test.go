package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/werd-platform/werd/src/go/api/internal/middleware"
	"github.com/werd-platform/werd/src/go/api/internal/service"
)

// TestWriteJSON verifies the JSON response helper sets headers correctly.
func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["key"] != "value" {
		t.Fatalf("unexpected body: %v", body)
	}
}

// TestWriteError verifies error responses include both message and detail.
func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusInternalServerError, "something failed", fmt.Errorf("database: connection refused"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var body errorResponse
	json.NewDecoder(w.Body).Decode(&body)
	if body.Message != "something failed" {
		t.Fatalf("expected message 'something failed', got '%s'", body.Message)
	}
	if body.Detail != "database: connection refused" {
		t.Fatalf("expected detail 'database: connection refused', got '%s'", body.Detail)
	}
}

// TestWriteErrorNilErr verifies error responses work when err is nil.
func TestWriteErrorNilErr(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad input", nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var body errorResponse
	json.NewDecoder(w.Body).Decode(&body)
	if body.Message != "bad input" {
		t.Fatalf("expected message 'bad input', got '%s'", body.Message)
	}
	if body.Detail != "" {
		t.Fatalf("expected empty detail, got '%s'", body.Detail)
	}
}

// TestLoginRequest_MissingFields verifies login validation.
func TestLoginRequest_MissingFields(t *testing.T) {
	auth := &Auth{svc: nil} // svc not called for validation failures

	tests := []struct {
		name string
		body string
		code int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"missing password", `{"email":"a@b.com"}`, http.StatusBadRequest},
		{"missing email", `{"password":"pass"}`, http.StatusBadRequest},
		{"invalid json", `{bad`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			auth.Login(w, req)

			if w.Code != tt.code {
				t.Fatalf("expected %d, got %d: %s", tt.code, w.Code, w.Body.String())
			}
		})
	}
}

// TestErrAuthUserNotFound_Exists verifies the sentinel error is properly defined
// so handlers can detect stale-token scenarios and return 401 instead of 500.
func TestErrAuthUserNotFound_Exists(t *testing.T) {
	if service.ErrAuthUserNotFound == nil {
		t.Fatal("ErrAuthUserNotFound should be defined")
	}
	if service.ErrAuthUserNotFound.Error() != "user not found" {
		t.Fatalf("unexpected error text: %s", service.ErrAuthUserNotFound.Error())
	}
}

// TestErrorResponseFormat verifies error responses include the detail field
// which helps debug 500 errors without looking at server logs.
func TestErrorResponseFormat(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, 500, "operation failed", fmt.Errorf("FK violation: user_id does not exist"))

	var body errorResponse
	json.NewDecoder(w.Body).Decode(&body)

	if body.Message != "operation failed" {
		t.Fatalf("wrong message: %s", body.Message)
	}
	if body.Detail == "" {
		t.Fatal("detail should include the underlying error")
	}
	if body.Detail != "FK violation: user_id does not exist" {
		t.Fatalf("wrong detail: %s", body.Detail)
	}
}

// TestMiddlewareUserIDKey verifies the context key is accessible from handler tests.
func TestMiddlewareUserIDKey(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, "test-user-id")
	val := middleware.UserIDFromContext(ctx)
	if val != "test-user-id" {
		t.Fatalf("expected test-user-id, got %s", val)
	}
}
