package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicSubscriptionDoesNotRequireLogin(t *testing.T) {
	app := newTestApp(t)
	sub := createTestSubscription(t, app.store, "demo", []ProxyLinkInput{{Name: "named-node", URL: sampleVLESS}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/sub/"+sub.Token+".yaml", nil)
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "proxies:") || !strings.Contains(rec.Body.String(), "named-node") {
		t.Fatalf("unexpected subscription body:\n%s", rec.Body.String())
	}
}

func TestSubscriptionAPIRequiresLoginAndCreatesSubscription(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"name":"work","links":[{"name":"work-node","url":"` + sampleVLESS + `"}],"rule_group_ids":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", rec.Code)
	}

	cookie := loginTestUser(t, app)
	body = bytes.NewBufferString(`{"name":"work","links":[{"name":"work-node","url":"` + sampleVLESS + `"}],"rule_group_ids":[]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/subscriptions", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("authenticated status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got SubscriptionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "work" || got.Token == "" || !strings.Contains(got.URL, "/sub/") || got.Links[0].Name != "work-node" {
		t.Fatalf("bad response: %#v", got)
	}
}

func TestAppUsesConfiguredAdminCredentials(t *testing.T) {
	store, err := OpenStore("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	app := NewApp(store, AppConfig{AdminUser: "owner", AdminPassword: "secret-value"})

	form := bytes.NewBufferString("username=admin&password=" + defaultAdminPassword)
	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("default credentials unexpectedly accepted with status %d", rec.Code)
	}

	form = bytes.NewBufferString("username=owner&password=secret-value")
	req = httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("configured credentials status = %d", rec.Code)
	}
}
