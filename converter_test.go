package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

const sampleVLESS = "vless://11111111-2222-3333-4444-555555555555@proxy.example.com:443?type=tcp&encryption=none&security=reality&pbk=example-public-key&fp=chrome&sni=example.com&sid=&spx=%2F#example-vless"

func TestParseVLESSRealityForMihomo(t *testing.T) {
	proxy, err := ParseProxyLink(sampleVLESS, 0)
	if err != nil {
		t.Fatalf("ParseProxyLink returned error: %v", err)
	}

	if proxy["type"] != "vless" {
		t.Fatalf("type = %v, want vless", proxy["type"])
	}
	if proxy["name"] != "example-vless" {
		t.Fatalf("name = %v", proxy["name"])
	}
	if proxy["server"] != "proxy.example.com" || proxy["port"] != 443 {
		t.Fatalf("server/port = %v/%v", proxy["server"], proxy["port"])
	}
	if proxy["uuid"] != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("uuid = %v", proxy["uuid"])
	}
	if proxy["network"] != "tcp" || proxy["tls"] != true || proxy["reality-opts"] == nil {
		t.Fatalf("missing Mihomo Reality fields: %#v", proxy)
	}
	reality := proxy["reality-opts"].(map[string]any)
	if reality["public-key"] != "example-public-key" {
		t.Fatalf("public-key = %v", reality["public-key"])
	}
	if proxy["client-fingerprint"] != "chrome" || proxy["servername"] != "example.com" {
		t.Fatalf("fingerprint/servername not mapped: %#v", proxy)
	}
}

func TestBuildMihomoConfigIncludesDefaultAndSelectedRules(t *testing.T) {
	config, err := BuildMihomoConfig(SubscriptionConfig{
		Name:  "demo",
		Links: []ProxyLinkConfig{{Name: "custom-us", URL: sampleVLESS}},
		RuleGroups: []RuleGroupConfig{
			{Name: "ai", Rules: "DOMAIN-SUFFIX,openai.com,PROXY\nDOMAIN-SUFFIX,anthropic.com,PROXY"},
		},
	})
	if err != nil {
		t.Fatalf("BuildMihomoConfig returned error: %v", err)
	}

	for _, want := range []string{
		"proxies:",
		"name: custom-us",
		"reality-opts:",
		"DOMAIN-SUFFIX,openai.com,PROXY",
		"GEOIP,LAN,DIRECT",
		"MATCH,FINAL",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("config missing %q:\n%s", want, config)
		}
	}
}

func TestParseSupportedProxySchemes(t *testing.T) {
	vmessPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"v":"2","ps":"vmess-node","add":"vmess.example.com","port":"443","id":"11111111-1111-1111-1111-111111111111","aid":"0","scy":"auto","net":"tcp","tls":"tls","sni":"vmess.example.com"}`))
	ssUser := base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:ss-pass"))
	ssrMain := base64.RawURLEncoding.EncodeToString([]byte("ssr.example.com:8388:origin:plain:aes-128-gcm:" + base64.RawURLEncoding.EncodeToString([]byte("ssr-pass")) + "/?remarks=" + base64.RawURLEncoding.EncodeToString([]byte("ssr-node"))))
	cases := []struct {
		raw       string
		wantType  string
		wantName  string
		wantField string
	}{
		{"vmess://" + vmessPayload, "vmess", "vmess-node", "uuid"},
		{"trojan://trojan-pass@trojan.example.com:443?sni=trojan.example.com#trojan-node", "trojan", "trojan-node", "password"},
		{"ss://" + ssUser + "@ss.example.com:8388#ss-node", "ss", "ss-node", "cipher"},
		{"ssr://" + ssrMain, "ssr", "ssr-node", "protocol"},
		{"hysteria://hysteria-pass@hy.example.com:443?sni=hy.example.com#hy-node", "hysteria", "hy-node", "password"},
		{"hysteria2://hy2-pass@hy2.example.com:443?sni=hy2.example.com#hy2-node", "hysteria2", "hy2-node", "password"},
		{"hy2://hy2-pass@hy2.example.com:443?sni=hy2.example.com#hy2-short-node", "hysteria2", "hy2-short-node", "password"},
	}

	for _, tc := range cases {
		t.Run(tc.wantType, func(t *testing.T) {
			proxy, err := ParseProxyLink(tc.raw, 0)
			if err != nil {
				t.Fatalf("ParseProxyLink returned error: %v", err)
			}
			if proxy["type"] != tc.wantType || proxy["name"] != tc.wantName || proxy[tc.wantField] == nil {
				t.Fatalf("bad parsed proxy: %#v", proxy)
			}
		})
	}
}
