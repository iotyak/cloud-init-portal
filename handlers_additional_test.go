package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleIndexMethodAndPathGuards(t *testing.T) {
	srv := newTestServer(t)

	notFoundReq := httptest.NewRequest(http.MethodGet, "/missing", nil)
	notFoundRR := httptest.NewRecorder()
	srv.HandleIndex(notFoundRR, notFoundReq)
	if notFoundRR.Code != http.StatusNotFound {
		t.Fatalf("status=%d want %d", notFoundRR.Code, http.StatusNotFound)
	}

	methodReq := httptest.NewRequest(http.MethodPost, "/", nil)
	methodRR := httptest.NewRecorder()
	srv.HandleIndex(methodRR, methodReq)
	if methodRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d", methodRR.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleStatusMethodGuard(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/status", nil)
	rr := httptest.NewRecorder()
	srv.HandleStatus(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserDataAndMetaDataLifecycle(t *testing.T) {
	srv := newTestServer(t)
	_ = postForm(t, srv, "/provision", baseProvisionForm())

	udReq := httptest.NewRequest(http.MethodGet, "/user-data", nil)
	udRR := httptest.NewRecorder()
	srv.HandleUserData(udRR, udReq)
	if udRR.Code != http.StatusOK {
		t.Fatalf("user-data status=%d want %d", udRR.Code, http.StatusOK)
	}
	if ct := udRR.Header().Get("Content-Type"); !strings.Contains(ct, "text/yaml") {
		t.Fatalf("unexpected user-data content-type: %q", ct)
	}
	if body := udRR.Body.String(); !strings.Contains(body, "#cloud-config") {
		t.Fatalf("unexpected user-data body: %q", body)
	}

	mdReq := httptest.NewRequest(http.MethodGet, "/meta-data", nil)
	mdRR := httptest.NewRecorder()
	srv.HandleMetaData(mdRR, mdReq)
	if mdRR.Code != http.StatusOK {
		t.Fatalf("meta-data status=%d want %d", mdRR.Code, http.StatusOK)
	}
	if ct := mdRR.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("unexpected meta-data content-type: %q", ct)
	}
	if body := mdRR.Body.String(); !strings.Contains(body, "instance-id") {
		t.Fatalf("unexpected meta-data body: %q", body)
	}

	status := srv.Store.CurrentStatus()
	if status.Status != StatusConsumed {
		t.Fatalf("status=%q want %q", status.Status, StatusConsumed)
	}

	nextUserReq := httptest.NewRequest(http.MethodGet, "/user-data", nil)
	nextUserRR := httptest.NewRecorder()
	srv.HandleUserData(nextUserRR, nextUserReq)
	if nextUserRR.Code != http.StatusNotFound {
		t.Fatalf("status=%d want %d", nextUserRR.Code, http.StatusNotFound)
	}
}

func TestUserDataAndMetaDataMethodGuards(t *testing.T) {
	srv := newTestServer(t)

	udReq := httptest.NewRequest(http.MethodPost, "/user-data", nil)
	udRR := httptest.NewRecorder()
	srv.HandleUserData(udRR, udReq)
	if udRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d", udRR.Code, http.StatusMethodNotAllowed)
	}

	mdReq := httptest.NewRequest(http.MethodPost, "/meta-data", nil)
	mdRR := httptest.NewRecorder()
	srv.HandleMetaData(mdRR, mdReq)
	if mdRR.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d", mdRR.Code, http.StatusMethodNotAllowed)
	}
}

func TestRequestBaseURLFallbackWhenHostMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = ""
	if got, want := requestBaseURL(req), "http://127.0.0.1:8080"; got != want {
		t.Fatalf("requestBaseURL()=%q want %q", got, want)
	}
}

func TestLoggingMiddlewarePassThrough(t *testing.T) {
	h := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d want %d", rr.Code, http.StatusCreated)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("body=%q want ok", rr.Body.String())
	}
}
