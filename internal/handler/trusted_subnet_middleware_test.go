package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedSubnetMiddleware_EmptySubnet(t *testing.T) {
	tm, err := NewTrustedSubnetMiddleware("")
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	handler := tm.Check(next)

	req := httptest.NewRequest("POST", "/update", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	w := httptest.NewRecorder()

	handler(w, req)

	if !nextCalled {
		t.Error("Next handler should be called when subnet is empty")
	}
}

func TestTrustedSubnetMiddleware_IPInSubnet(t *testing.T) {
	tm, err := NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	handler := tm.Check(next)

	req := httptest.NewRequest("POST", "/update", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	w := httptest.NewRecorder()

	handler(w, req)

	if !nextCalled {
		t.Error("Next handler should be called when IP is in subnet")
	}
}

func TestTrustedSubnetMiddleware_IPNotInSubnet(t *testing.T) {
	tm, err := NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	handler := tm.Check(next)

	req := httptest.NewRequest("POST", "/update", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	w := httptest.NewRecorder()

	handler(w, req)

	if nextCalled {
		t.Error("Next handler should not be called when IP is not in subnet")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestTrustedSubnetMiddleware_MissingHeader(t *testing.T) {
	tm, err := NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	handler := tm.Check(next)

	req := httptest.NewRequest("POST", "/update", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if nextCalled {
		t.Error("Next handler should not be called when X-Real-IP header is missing")
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestTrustedSubnetMiddleware_InvalidIP(t *testing.T) {
	tm, err := NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	nextCalled := false
	next := func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}

	handler := tm.Check(next)

	req := httptest.NewRequest("POST", "/update", nil)
	req.Header.Set("X-Real-IP", "invalid-ip")
	w := httptest.NewRecorder()

	handler(w, req)

	if nextCalled {
		t.Error("Next handler should not be called when IP is invalid")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNewTrustedSubnetMiddleware_InvalidCIDR(t *testing.T) {
	_, err := NewTrustedSubnetMiddleware("invalid-cidr")
	if err == nil {
		t.Error("Expected error for invalid CIDR")
	}
}
