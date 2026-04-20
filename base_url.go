package main

import (
	"net/http"
	"net/url"
	"strings"
)

func requestBaseURL(r *http.Request, publicBaseURL string, trustProxyHeaders bool) string {
	if cleaned := normalizePublicBaseURL(publicBaseURL); cleaned != "" {
		return cleaned
	}

	scheme := "http"
	host := strings.TrimSpace(r.Host)

	if trustProxyHeaders {
		if xfProto := firstHeaderValue(r.Header.Get("X-Forwarded-Proto")); xfProto != "" {
			scheme = xfProto
		}
		if xfHost := firstHeaderValue(r.Header.Get("X-Forwarded-Host")); xfHost != "" {
			host = xfHost
		}
	}

	if host == "" {
		host = "127.0.0.1:8080"
	}

	return scheme + "://" + host
}

func normalizePublicBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.TrimRight(u.String(), "/")
}

func firstHeaderValue(v string) string {
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ",")
	return strings.TrimSpace(parts[0])
}
