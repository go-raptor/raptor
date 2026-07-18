package core

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

type IPExtractor func(*http.Request) string

// IPTrustFunc reports whether an IP belongs to a trusted proxy.
type IPTrustFunc func(net.IP) bool

func defaultTrustedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate()
}

// TrustedProxies builds an IPTrustFunc from CIDR entries (bare IPs are
// accepted too). An empty list yields the default trust set: loopback,
// link-local, and private ranges.
func TrustedProxies(cidrs []string) (IPTrustFunc, error) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		entry := strings.TrimSpace(cidr)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		_, ipNet, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: %w", cidr, err)
		}
		nets = append(nets, ipNet)
	}
	if len(nets) == 0 {
		return defaultTrustedIP, nil
	}
	return func(ip net.IP) bool {
		for _, n := range nets {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}, nil
}

func trustFunc(trusted []IPTrustFunc) IPTrustFunc {
	if len(trusted) > 0 && trusted[0] != nil {
		return trusted[0]
	}
	return defaultTrustedIP
}

func ExtractIPDirect() IPExtractor {
	return func(req *http.Request) string {
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		return host
	}
}

func ExtractIPFromRealIPHeader(trusted ...IPTrustFunc) IPExtractor {
	isTrusted := trustFunc(trusted)
	return func(req *http.Request) string {
		directIP, _, _ := net.SplitHostPort(req.RemoteAddr)

		realIP := req.Header.Get(HeaderXRealIP)
		if realIP == "" {
			return directIP
		}

		if ip := net.ParseIP(directIP); ip != nil && isTrusted(ip) {
			realIP = strings.Trim(realIP, "[]")
			if net.ParseIP(realIP) != nil {
				return realIP
			}
		}

		return directIP
	}
}

func ExtractIPFromXFFHeader(trusted ...IPTrustFunc) IPExtractor {
	isTrusted := trustFunc(trusted)
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
		var ip net.IP
		for {
			ip = net.ParseIP(strings.Trim(strings.TrimSpace(candidate), "[]"))
			if ip == nil {
				return directIP
			}
			if !isTrusted(ip) {
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

		// Every hop was trusted; ip holds the leftmost XFF entry, already
		// parsed above — return it validated and normalized.
		return ip.String()
	}
}
