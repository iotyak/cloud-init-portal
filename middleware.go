package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type fixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	entries map[string]windowEntry
}

type windowEntry struct {
	window int64
	count  int
}

func newFixedWindowLimiter(limit int) *fixedWindowLimiter {
	if limit <= 0 {
		limit = 1
	}
	return &fixedWindowLimiter{limit: limit, entries: make(map[string]windowEntry)}
}

func (l *fixedWindowLimiter) allow(key string) bool {
	if l == nil {
		return true
	}
	nowWindow := time.Now().Unix()

	l.mu.Lock()
	defer l.mu.Unlock()

	e := l.entries[key]
	if e.window != nowWindow {
		e.window = nowWindow
		e.count = 0
	}
	e.count++
	l.entries[key] = e
	return e.count <= l.limit
}

var requestCounter uint64

func middlewareChain(next http.Handler, srv *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := nextRequestID()
		w.Header().Set("X-Request-ID", reqID)

		if isRateLimitedEndpoint(r) {
			key := clientIP(r) + "|" + r.URL.Path
			allowed := true
			if r.URL.Path == "/status" {
				allowed = srv.StatusLimiter.allow(key)
			} else {
				allowed = srv.WriteLimiter.allow(key)
			}
			if !allowed {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				log.Printf("req_id=%s method=%s path=%s remote=%s status=429", reqID, r.Method, r.URL.Path, r.RemoteAddr)
				return
			}
		}

		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		log.Printf(
			"req_id=%s method=%s path=%s remote=%s status=%d duration_ms=%d",
			reqID,
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			rw.status,
			duration.Milliseconds(),
		)
	})
}

func nextRequestID() string {
	n := atomic.AddUint64(&requestCounter, 1)
	return fmt.Sprintf("req-%d-%d", time.Now().Unix(), n)
}

func isRateLimitedEndpoint(r *http.Request) bool {
	if r.Method == http.MethodGet && r.URL.Path == "/status" {
		return true
	}
	if r.Method == http.MethodPost {
		switch r.URL.Path {
		case "/provision", "/consume", "/force-replace":
			return true
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if ip := net.ParseIP(r.RemoteAddr); ip != nil {
		return ip.String()
	}
	return strconv.Quote(r.RemoteAddr)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
