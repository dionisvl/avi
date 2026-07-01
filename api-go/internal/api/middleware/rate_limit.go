package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/dionisvl/avi/api-go/internal/api"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
)

const limiterTTL = 5 * time.Minute

// hardcoded trusted proxy CIDRs: loopback + Docker/private networks
var defaultTrustedCIDRs = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters = make(map[string]*limiterEntry)
	mu       sync.Mutex
)

func init() {
	go func() {
		ticker := time.NewTicker(limiterTTL)
		defer ticker.Stop()
		for range ticker.C {
			evictStale()
		}
	}()
}

func evictStale() {
	mu.Lock()
	defer mu.Unlock()
	cutoff := time.Now().Add(-limiterTTL)
	for ip, e := range limiters {
		if e.lastSeen.Before(cutoff) {
			delete(limiters, ip)
		}
	}
}

// parseCIDRs merges defaultTrustedCIDRs with any extra CIDRs from config.
func parseCIDRs(extra []string) []*net.IPNet {
	all := append(defaultTrustedCIDRs, extra...)
	nets := make([]*net.IPNet, 0, len(all))
	for _, cidr := range all {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			nets = append(nets, ipNet)
		}
	}
	return nets
}

func isTrustedProxy(remoteAddr string, trustedNets []*net.IPNet) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trustedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func getIP(r *http.Request, trustedNets []*net.IPNet) string {
	if isTrustedProxy(r.RemoteAddr, trustedNets) {
		if raw := strings.TrimSpace(r.Header.Get("X-Real-IP")); raw != "" {
			if ip := net.ParseIP(raw); ip != nil {
				return ip.String()
			}
		}
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			first := fwd
			if before, _, ok := strings.Cut(fwd, ","); ok {
				first = before
			}
			if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func RateLimit(rps float64, burst int, extraTrustedCIDRs ...string) func(http.Handler) http.Handler {
	trustedNets := parseCIDRs(extraTrustedCIDRs)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r, trustedNets)
			key := fmt.Sprintf("%s|%g|%d", ip, rps, burst)

			mu.Lock()
			e, ok := limiters[key]
			if !ok {
				e = &limiterEntry{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
				limiters[key] = e
			}
			e.lastSeen = time.Now()
			limiter := e.limiter
			mu.Unlock()

			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				api.WriteError(w, apierr.New(apierr.ErrRateLimited, "Rate limit exceeded. Please try again later."))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ResetLimiters clears all rate limiters (useful for testing)
func ResetLimiters() {
	mu.Lock()
	defer mu.Unlock()
	limiters = make(map[string]*limiterEntry)
}
