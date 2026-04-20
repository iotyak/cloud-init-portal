package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareChainAddsRequestID(t *testing.T) {
	srv := &Server{
		StatusLimiter: newFixedWindowLimiter(100),
		WriteLimiter:  newFixedWindowLimiter(100),
	}
	h := middlewareChain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), srv)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusNoContent)
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestMiddlewareChainRateLimitsStatusEndpoint(t *testing.T) {
	srv := &Server{
		StatusLimiter: newFixedWindowLimiter(1),
		WriteLimiter:  newFixedWindowLimiter(100),
	}
	h := middlewareChain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), srv)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.RemoteAddr = "127.0.0.1:23456"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if i == 0 && rr.Code != http.StatusOK {
			t.Fatalf("first status=%d want %d", rr.Code, http.StatusOK)
		}
		if i == 1 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("second status=%d want %d", rr.Code, http.StatusTooManyRequests)
		}
	}
}

func TestMiddlewareChainRateLimitsWriteEndpoints(t *testing.T) {
	srv := &Server{
		StatusLimiter: newFixedWindowLimiter(100),
		WriteLimiter:  newFixedWindowLimiter(1),
	}
	h := middlewareChain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), srv)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/provision", nil)
		req.RemoteAddr = "127.0.0.1:34567"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if i == 0 && rr.Code != http.StatusOK {
			t.Fatalf("first status=%d want %d", rr.Code, http.StatusOK)
		}
		if i == 1 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("second status=%d want %d", rr.Code, http.StatusTooManyRequests)
		}
	}
}
