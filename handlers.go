package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	sessionCookie = "mihomo_sub_session"
)

type App struct {
	store    *Store
	config   AppConfig
	sessions map[string]time.Time
}

func NewApp(store *Store, config AppConfig) *App {
	return &App{store: store, config: config, sessions: map[string]time.Time{}}
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/logout", a.handleLogout)
	mux.HandleFunc("/sub/", a.handlePublicSubscription)
	mux.HandleFunc("/api/subscriptions", a.requireAuth(a.handleSubscriptions))
	mux.HandleFunc("/api/subscriptions/", a.requireAuth(a.handleSubscriptionByID))
	mux.HandleFunc("/api/rule-groups", a.requireAuth(a.handleRuleGroups))
	mux.HandleFunc("/api/rule-groups/", a.requireAuth(a.handleRuleGroupByID))
	mux.HandleFunc("/api/export", a.requireAuth(a.handleExport))
	mux.HandleFunc("/api/import", a.requireAuth(a.handleImport))
	return mux
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !a.isAuthed(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminHTML))
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(loginHTML))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	if r.FormValue("username") != a.config.AdminUser || r.FormValue("password") != a.config.AdminPassword {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	token := newToken()
	a.sessions[token] = time.Now().Add(24 * time.Hour)
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		delete(a.sessions, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (a *App) handlePublicSubscription(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/sub/"), ".yaml")
	if token == "" {
		http.NotFound(w, r)
		return
	}
	sub, err := a.store.GetSubscriptionByToken(token)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	config, err := BuildMihomoConfig(toSubscriptionConfig(sub))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	_, _ = w.Write([]byte(config))
}

func (a *App) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		subs, err := a.store.ListSubscriptions()
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		out := make([]SubscriptionResponse, 0, len(subs))
		for i := range subs {
			out = append(out, subscriptionResponse(&subs[i], r))
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var input SubscriptionInput
		if !decodeJSON(w, r, &input) {
			return
		}
		sub, err := a.store.CreateSubscription(input)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, subscriptionResponse(sub, r))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/subscriptions/")
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPut:
		var input SubscriptionInput
		if !decodeJSON(w, r, &input) {
			return
		}
		sub, err := a.store.UpdateSubscription(id, input)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		sub, _ = a.store.GetSubscriptionByID(sub.ID)
		writeJSON(w, http.StatusOK, subscriptionResponse(sub, r))
	case http.MethodDelete:
		if err := a.store.DeleteSubscription(id); err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleRuleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		groups, err := a.store.ListRuleGroups()
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		out := make([]RuleGroupResponse, 0, len(groups))
		for _, group := range groups {
			out = append(out, RuleGroupResponse{ID: group.ID, Name: group.Name, Rules: group.Rules})
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		var input RuleGroupInput
		if !decodeJSON(w, r, &input) {
			return
		}
		group, err := a.store.CreateRuleGroup(input)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, RuleGroupResponse{ID: group.ID, Name: group.Name, Rules: group.Rules})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleRuleGroupByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "/api/rule-groups/")
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPut:
		var input RuleGroupInput
		if !decodeJSON(w, r, &input) {
			return
		}
		group, err := a.store.UpdateRuleGroup(id, input)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, RuleGroupResponse{ID: group.ID, Name: group.Name, Rules: group.Rules})
	case http.MethodDelete:
		if err := a.store.DeleteRuleGroup(id); err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := a.store.Export()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="mihomo-sub-manager-export.json"`)
	writeJSON(w, http.StatusOK, data)
}

func (a *App) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var data ExportData
	if !decodeJSON(w, r, &data) {
		return
	}
	if err := a.store.Import(data); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.isAuthed(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *App) isAuthed(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	expiry, ok := a.sessions[cookie.Value]
	if !ok || time.Now().After(expiry) {
		return false
	}
	return true
}

func toSubscriptionConfig(sub *Subscription) SubscriptionConfig {
	cfg := SubscriptionConfig{Name: sub.Name}
	for _, link := range sub.Links {
		cfg.Links = append(cfg.Links, ProxyLinkConfig{Name: link.Name, URL: link.Raw})
	}
	for _, group := range sub.RuleGroups {
		cfg.RuleGroups = append(cfg.RuleGroups, RuleGroupConfig{Name: group.Name, Rules: group.Rules})
	}
	return cfg
}

func subscriptionResponse(sub *Subscription, r *http.Request) SubscriptionResponse {
	resp := SubscriptionResponse{ID: sub.ID, Name: sub.Name, Token: sub.Token, URL: absoluteSubURL(r, sub.Token)}
	for _, link := range sub.Links {
		resp.Links = append(resp.Links, ProxyLinkInput{Name: link.Name, URL: link.Raw})
	}
	for _, group := range sub.RuleGroups {
		resp.RuleGroupIDs = append(resp.RuleGroupIDs, group.ID)
	}
	return resp
}

func absoluteSubURL(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/sub/" + token + ".yaml"
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error, status int) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func pathID(w http.ResponseWriter, r *http.Request, prefix string) (uint, bool) {
	raw := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id64 == 0 {
		http.NotFound(w, r)
		return 0, false
	}
	return uint(id64), true
}
