package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	store, err := OpenStore("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	return NewApp(store, DefaultAppConfig())
}

func createTestSubscription(t *testing.T, store *Store, name string, links []ProxyLinkInput, groupIDs []uint) *Subscription {
	t.Helper()
	sub, err := store.CreateSubscription(SubscriptionInput{Name: name, Links: links, RuleGroupIDs: groupIDs})
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
	return sub
}

func loginTestUser(t *testing.T, app *App) *http.Cookie {
	t.Helper()
	cfg := app.config
	form := url.Values{"username": {cfg.AdminUser}, "password": {cfg.AdminPassword}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.routes().ServeHTTP(rec, req)
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookie {
			return cookie
		}
	}
	t.Fatalf("login did not set session cookie, status=%d body=%s", rec.Code, rec.Body.String())
	return nil
}
