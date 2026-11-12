package handler

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"saas-starter/api/internal/store"
)

type userKey struct{}

func (h *Handler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := h.svc.UserFromSession(r.Context(), sessionToken(r))
		if err != nil {
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, &u)))
	})
}

func userFrom(r *http.Request) *store.User {
	u, _ := r.Context().Value(userKey{}).(*store.User)
	return u
}

func sessionToken(r *http.Request) string {
	const scheme = "Session"
	v := r.Header.Get("Authorization")
	if len(v) <= len(scheme)+1 || !strings.EqualFold(v[:len(scheme)], scheme) || v[len(scheme)] != ' ' {
		return ""
	}
	return strings.TrimSpace(v[len(scheme)+1:])
}

func clientIP(r *http.Request) string {
	key := os.Getenv("INTERNAL_KEY")
	if key != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Key")), []byte(key)) == 1 {
		if ip := limitIP(r.Header.Get("X-Client-IP")); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := limitIP(host); ip != "" {
		return ip
	}
	return host
}

func limitIP(s string) string {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String()
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http", "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "ms", time.Since(start).Milliseconds())
	})
}
