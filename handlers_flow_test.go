package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	txttmpl "text/template"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	tmpl, err := txttmpl.New("example.yaml").Option("missingkey=error").Parse("#cloud-config\nhostname: {{.Hostname}}\n")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}

	return &Server{
		Store: NewStore(),
		Templates: map[string]CloudInitTemplate{
			"example": {
				Name:     "example",
				Filename: "example.yaml",
				Raw:      "#cloud-config\n",
				Compiled: tmpl,
			},
		},
		BoxTypes: DefaultBoxTypes(),
	}
}

func postForm(t *testing.T, srv *Server, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	switch path {
	case "/provision":
		srv.HandleProvision(rr, req)
	case "/consume":
		srv.HandleConsume(rr, req)
	case "/force-replace":
		srv.HandleForceReplace(rr, req)
	default:
		t.Fatalf("unsupported path: %s", path)
	}
	return rr
}

func baseProvisionForm() url.Values {
	return url.Values{
		"template":  {"example"},
		"box_type":  {"nuc-dual-nic"},
		"hostname":  {"edge-001"},
		"static_ip": {"192.168.50.10"},
		"cidr":      {"24"},
		"gateway":   {"192.168.50.1"},
		"dns":       {"1.1.1.1,8.8.8.8"},
	}
}

func TestHandleProvisionRejectsInvalidGateway(t *testing.T) {
	srv := newTestServer(t)
	form := baseProvisionForm()
	form.Set("gateway", "not-an-ip")

	rr := postForm(t, srv, "/provision", form)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d body=%q", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if got := srv.Store.GetCurrent(); got != nil {
		t.Fatalf("expected no active config, got %+v", got)
	}
}

func TestHandleProvisionRejectsInvalidDNS(t *testing.T) {
	srv := newTestServer(t)
	form := baseProvisionForm()
	form.Set("dns", "1.1.1.1,not-an-ip")

	rr := postForm(t, srv, "/provision", form)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d body=%q", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if got := srv.Store.GetCurrent(); got != nil {
		t.Fatalf("expected no active config, got %+v", got)
	}
}

func TestHandleProvisionSuccessAndConflict(t *testing.T) {
	srv := newTestServer(t)
	form := baseProvisionForm()

	rr := postForm(t, srv, "/provision", form)
	if rr.Code != http.StatusOK {
		t.Fatalf("first status=%d want %d body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}
	if got := srv.Store.GetCurrent(); got == nil || got.Hostname != "edge-001" {
		t.Fatalf("expected active config for edge-001, got %+v", got)
	}

	form2 := baseProvisionForm()
	form2.Set("hostname", "edge-002")
	rr2 := postForm(t, srv, "/provision", form2)
	if rr2.Code != http.StatusConflict {
		t.Fatalf("second status=%d want %d body=%q", rr2.Code, http.StatusConflict, rr2.Body.String())
	}
}

func TestConsumeAndForceReplaceHandlers(t *testing.T) {
	srv := newTestServer(t)
	form := baseProvisionForm()
	_ = postForm(t, srv, "/provision", form)

	consumeResp := postForm(t, srv, "/consume", url.Values{})
	if consumeResp.Code != http.StatusSeeOther {
		t.Fatalf("consume status=%d want %d", consumeResp.Code, http.StatusSeeOther)
	}
	if srv.Store.GetCurrent() != nil {
		t.Fatalf("expected no current config after consume")
	}

	form2 := baseProvisionForm()
	form2.Set("hostname", "edge-003")
	_ = postForm(t, srv, "/provision", form2)

	forceResp := postForm(t, srv, "/force-replace", url.Values{})
	if forceResp.Code != http.StatusSeeOther {
		t.Fatalf("force-replace status=%d want %d", forceResp.Code, http.StatusSeeOther)
	}
	if srv.Store.GetCurrent() != nil {
		t.Fatalf("expected no current config after force replace")
	}
}
