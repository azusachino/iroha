package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

// forwardedClientHeader carries the original client IP set by Cloudflare. It
// is trusted only when the direct peer is a configured proxy (ADR-0008 §5);
// from any other peer it is ignored, so a caller cannot pick its own bucket.
const forwardedClientHeader = "Cf-Connecting-Ip"

// clientIP returns the address rate limits and logs key on.
func (s *Server) clientIP(r *http.Request) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	}
	addr, err := netip.ParseAddr(peer)
	if err != nil || !s.trustedProxy(addr) {
		return peer
	}
	forwarded, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get(forwardedClientHeader)))
	if err != nil {
		return peer
	}
	return forwarded.String()
}

func (s *Server) trustedProxy(addr netip.Addr) bool {
	for _, prefix := range s.trustedProxies {
		if prefix.Contains(addr.Unmap()) {
			return true
		}
	}
	return false
}

// limitByClient builds a per-client rate limiter over window.
func (s *Server) limitByClient(requests int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitBy(requests, window, func(r *http.Request) (string, error) {
		return s.clientIP(r), nil
	}, httprate.WithLimitHandler(rateLimitResponse))
}

func parseTrustedProxies(values []string) ([]netip.Prefix, []string) {
	var prefixes []netip.Prefix
	var invalid []string
	for _, value := range values {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			invalid = append(invalid, value)
			continue
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, invalid
}
