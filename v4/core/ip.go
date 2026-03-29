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

		ips := append(strings.Split(xff, ","), directIP)
		for i := len(ips) - 1; i >= 0; i-- {
			candidate := strings.Trim(strings.TrimSpace(ips[i]), "[]")
			ip := net.ParseIP(candidate)
			if ip == nil {
				return directIP
			}
			if !isTrustedIP(ip) {
				return ip.String()
			}
		}

		return strings.TrimSpace(ips[0])
	}
}

func isTrustedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate()
}
