package core

import (
	"net"
	"net/http"
	"strings"
)

type IPExtractor func(*http.Request) string

func ExtractIPDirect() IPExtractor {
	return func(req *http.Request) string {
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		return host
	}
}

func ExtractIPFromRealIPHeader() IPExtractor {
	return func(req *http.Request) string {
		directIP, _, _ := net.SplitHostPort(req.RemoteAddr)

		realIP := req.Header.Get(HeaderXRealIP)
		if realIP == "" {
			return directIP
		}

		if ip := net.ParseIP(directIP); ip != nil && isTrustedIP(ip) {
			realIP = strings.Trim(realIP, "[]")
			if net.ParseIP(realIP) != nil {
				return realIP
			}
		}

		return directIP
	}
}

func ExtractIPFromXFFHeader() IPExtractor {
	return func(req *http.Request) string {
		directIP, _, _ := net.SplitHostPort(req.RemoteAddr)

		xff := req.Header.Get(HeaderXForwardedFor)
		if xff == "" {
			return directIP
		}

		// Walk right-to-left without allocating: directIP first, then XFF
		// entries from rightmost to leftmost. Return the first non-trusted IP.
		candidate := directIP
		remaining := xff
		for {
			ip := net.ParseIP(strings.Trim(strings.TrimSpace(candidate), "[]"))
			if ip == nil {
				return directIP
			}
			if !isTrustedIP(ip) {
				return ip.String()
			}
			if remaining == "" {
				break
			}
			if i := strings.LastIndexByte(remaining, ','); i >= 0 {
				candidate = remaining[i+1:]
				remaining = remaining[:i]
			} else {
				candidate = remaining
				remaining = ""
			}
		}

		// All IPs trusted; fall back to the leftmost XFF entry.
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
}

func isTrustedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate()
}
