package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type RuleGroupConfig struct {
	Name  string `json:"name"`
	Rules string `json:"rules"`
}

type SubscriptionConfig struct {
	Name       string            `json:"name"`
	Links      []ProxyLinkConfig `json:"links"`
	RuleGroups []RuleGroupConfig `json:"rule_groups"`
}

type ProxyLinkConfig struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var defaultRules = []string{
	"DOMAIN-SUFFIX,local,DIRECT",
	"IP-CIDR,10.0.0.0/8,DIRECT,no-resolve",
	"IP-CIDR,172.16.0.0/12,DIRECT,no-resolve",
	"IP-CIDR,192.168.0.0/16,DIRECT,no-resolve",
	"IP-CIDR,127.0.0.0/8,DIRECT,no-resolve",
	"GEOIP,LAN,DIRECT",
	"GEOIP,CN,DIRECT",
}

func BuildMihomoConfig(sub SubscriptionConfig) (string, error) {
	proxies := make([]map[string]any, 0, len(sub.Links))
	names := make([]string, 0, len(sub.Links))
	for i, link := range sub.Links {
		if strings.TrimSpace(link.URL) == "" {
			continue
		}
		proxy, err := ParseProxyLink(link.URL, i)
		if err != nil {
			continue
		}
		if strings.TrimSpace(link.Name) != "" {
			proxy["name"] = strings.TrimSpace(link.Name)
		}
		proxies = append(proxies, proxy)
		names = append(names, proxy["name"].(string))
	}
	if len(proxies) == 0 {
		return "", errors.New("no valid proxy links")
	}

	proxyChoices := append([]string{"DIRECT"}, names...)
	groups := []map[string]any{
		{"name": "PROXY", "type": "select", "proxies": proxyChoices[1:]},
		{"name": "FINAL", "type": "select", "proxies": []string{"PROXY", "DIRECT"}},
	}

	rules := append([]string{}, defaultRules...)
	for _, group := range sub.RuleGroups {
		for _, line := range strings.Split(group.Rules, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			rules = append(rules, line)
		}
	}
	rules = append(rules, "MATCH,FINAL")

	doc := map[string]any{
		"mixed-port":                7890,
		"allow-lan":                 false,
		"mode":                      "rule",
		"log-level":                 "info",
		"ipv6":                      true,
		"unified-delay":             true,
		"tcp-concurrent":            true,
		"global-client-fingerprint": "chrome",
		"dns": map[string]any{
			"enable":             true,
			"listen":             "0.0.0.0:1053",
			"enhanced-mode":      "fake-ip",
			"fake-ip-range":      "198.18.0.1/16",
			"default-nameserver": []string{"223.5.5.5", "119.29.29.29"},
			"nameserver":         []string{"https://dns.alidns.com/dns-query", "https://doh.pub/dns-query"},
			"fallback":           []string{"https://1.1.1.1/dns-query", "https://8.8.8.8/dns-query"},
			"fake-ip-filter":     []string{"*.lan", "*.local"},
		},
		"proxies":      proxies,
		"proxy-groups": groups,
		"rules":        rules,
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func ParseProxyLink(raw string, index int) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(u.Scheme) {
	case "vless":
		return parseVLESS(u, index)
	case "trojan":
		return parseTrojan(u, index)
	case "ss":
		return parseSS(raw, u, index)
	case "vmess":
		return parseVMess(raw, index)
	case "hysteria", "hysteria2", "hy2":
		return parseHysteria(u, index)
	case "ssr":
		return parseSSR(raw, index)
	default:
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
}

func parseVLESS(u *url.URL, index int) (map[string]any, error) {
	host, port, err := splitHostPort(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	proxy := baseProxy("vless", displayName(u, index), host, port)
	proxy["uuid"] = u.User.Username()
	proxy["network"] = valueOr(q.Get("type"), "tcp")
	proxy["udp"] = true
	proxy["tls"] = q.Get("security") == "tls" || q.Get("security") == "reality"
	if q.Get("sni") != "" {
		proxy["servername"] = q.Get("sni")
	}
	if q.Get("fp") != "" {
		proxy["client-fingerprint"] = q.Get("fp")
	}
	if q.Get("flow") != "" {
		proxy["flow"] = q.Get("flow")
	}
	if q.Get("security") == "reality" {
		reality := map[string]any{}
		if q.Get("pbk") != "" {
			reality["public-key"] = q.Get("pbk")
		}
		if q.Get("sid") != "" {
			reality["short-id"] = q.Get("sid")
		}
		if q.Get("spx") != "" {
			reality["spider-x"] = q.Get("spx")
		}
		proxy["reality-opts"] = reality
	}
	applyTransport(proxy, q)
	return proxy, nil
}

func parseTrojan(u *url.URL, index int) (map[string]any, error) {
	host, port, err := splitHostPort(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	proxy := baseProxy("trojan", displayName(u, index), host, port)
	proxy["password"] = u.User.Username()
	proxy["udp"] = true
	proxy["sni"] = q.Get("sni")
	proxy["skip-cert-verify"] = q.Get("allowInsecure") == "1" || q.Get("skip-cert-verify") == "true"
	applyTransport(proxy, q)
	return proxy, nil
}

func parseSS(raw string, u *url.URL, index int) (map[string]any, error) {
	name := displayName(u, index)
	host, port, err := splitHostPort(u)
	if err != nil {
		return nil, err
	}
	user := u.User.String()
	if !strings.Contains(user, ":") {
		if decoded, decErr := decodeBase64(user); decErr == nil {
			user = decoded
		}
	}
	method, password, ok := strings.Cut(user, ":")
	if !ok {
		return nil, fmt.Errorf("invalid ss user info in %q", raw)
	}
	proxy := baseProxy("ss", name, host, port)
	proxy["cipher"] = method
	proxy["password"] = password
	proxy["udp"] = true
	return proxy, nil
}

func parseVMess(raw string, index int) (map[string]any, error) {
	payload := strings.TrimPrefix(raw, "vmess://")
	decoded, err := decodeBase64(payload)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(decoded), &v); err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(fmt.Sprint(v["port"]))
	proxy := baseProxy("vmess", valueOr(fmt.Sprint(v["ps"]), fmt.Sprintf("proxy-%d", index+1)), fmt.Sprint(v["add"]), port)
	proxy["uuid"] = fmt.Sprint(v["id"])
	proxy["alterId"] = numberOrZero(v["aid"])
	proxy["cipher"] = valueOr(fmt.Sprint(v["scy"]), "auto")
	proxy["network"] = valueOr(fmt.Sprint(v["net"]), "tcp")
	proxy["tls"] = fmt.Sprint(v["tls"]) == "tls"
	if fmt.Sprint(v["sni"]) != "" {
		proxy["servername"] = fmt.Sprint(v["sni"])
	}
	return proxy, nil
}

func parseHysteria(u *url.URL, index int) (map[string]any, error) {
	host, port, err := splitHostPort(u)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	proxyType := "hysteria2"
	if u.Scheme == "hysteria" {
		proxyType = "hysteria"
	}
	proxy := baseProxy(proxyType, displayName(u, index), host, port)
	proxy["password"] = u.User.Username()
	if q.Get("auth") != "" {
		proxy["auth-str"] = q.Get("auth")
	}
	if q.Get("sni") != "" {
		proxy["sni"] = q.Get("sni")
	}
	proxy["skip-cert-verify"] = q.Get("insecure") == "1"
	return proxy, nil
}

func parseSSR(raw string, index int) (map[string]any, error) {
	payload := strings.TrimPrefix(raw, "ssr://")
	decoded, err := decodeBase64(payload)
	if err != nil {
		return nil, err
	}
	mainPart, paramPart, _ := strings.Cut(decoded, "/?")
	parts := strings.Split(mainPart, ":")
	if len(parts) < 6 {
		return nil, errors.New("invalid ssr link")
	}
	port, _ := strconv.Atoi(parts[1])
	password, _ := decodeBase64(parts[5])
	name := fmt.Sprintf("proxy-%d", index+1)
	params, _ := url.ParseQuery(paramPart)
	if remarks := params.Get("remarks"); remarks != "" {
		if decodedName, err := decodeBase64(remarks); err == nil {
			name = decodedName
		}
	}
	proxy := baseProxy("ssr", name, parts[0], port)
	proxy["cipher"] = parts[3]
	proxy["password"] = password
	proxy["protocol"] = parts[2]
	proxy["obfs"] = parts[4]
	return proxy, nil
}

func baseProxy(proxyType, name, server string, port int) map[string]any {
	return map[string]any{"name": name, "type": proxyType, "server": server, "port": port}
}

func splitHostPort(u *url.URL) (string, int, error) {
	host := u.Hostname()
	portString := u.Port()
	if host == "" || portString == "" {
		return "", 0, fmt.Errorf("missing host or port")
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		return "", 0, err
	}
	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	}
	return host, port, nil
}

func displayName(u *url.URL, index int) string {
	if u.Fragment != "" {
		return u.Fragment
	}
	return fmt.Sprintf("proxy-%d", index+1)
}

func valueOr(value, fallback string) string {
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func numberOrZero(value any) int {
	n, _ := strconv.Atoi(fmt.Sprint(value))
	return n
}

func decodeBase64(value string) (string, error) {
	value = strings.TrimSpace(value)
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return string(decoded), nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(value); err == nil {
		return string(decoded), nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return string(decoded), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	return string(decoded), err
}

func applyTransport(proxy map[string]any, q url.Values) {
	network := valueOr(q.Get("type"), valueOr(fmt.Sprint(proxy["network"]), "tcp"))
	proxy["network"] = network
	switch network {
	case "ws":
		opts := map[string]any{}
		if q.Get("path") != "" {
			opts["path"] = q.Get("path")
		}
		if q.Get("host") != "" {
			opts["headers"] = map[string]any{"Host": q.Get("host")}
		}
		proxy["ws-opts"] = opts
	case "grpc":
		if q.Get("serviceName") != "" {
			proxy["grpc-opts"] = map[string]any{"grpc-service-name": q.Get("serviceName")}
		}
	}
}
