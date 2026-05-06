package main

import (
	"strings"
	"testing"
)

func TestTemplatesSupportSavedLanguagePreference(t *testing.T) {
	for _, html := range []string{loginHTML, adminHTML} {
		if !strings.Contains(html, "localStorage.getItem('ui_lang')") {
			t.Fatalf("template missing saved language lookup")
		}
		if !strings.Contains(html, "navigator.language") {
			t.Fatalf("template missing browser language default")
		}
		if !strings.Contains(html, "setLang") {
			t.Fatalf("template missing language switch handler")
		}
	}
}
