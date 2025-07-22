package middlewares

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/clearthree/url-shortener/internal/app/config"
)

// IPNet is the storage for CIDR specified in config
var IPNet *net.IPNet

func resolveIP(r *http.Request) (net.IP, error) {
	if !config.Settings.UseHeaderForSourceAddress {
		addr := r.RemoteAddr
		ipStr, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", ip)
		}
		return ip, nil
	} else {
		ipStr := r.Header.Get("X-Real-IP")
		ip := net.ParseIP(ipStr)
		if ip == nil {
			ips := r.Header.Get("X-Forwarded-For")
			ipStrs := strings.Split(ips, ",")
			ipStr = ipStrs[0]
			ip = net.ParseIP(ipStr)
		}
		if ip == nil {
			return nil, fmt.Errorf("failed parse ip from http header")
		}
		return ip, nil
	}
}

// CheckSubnet is a middleware that checks if the request's source addr matches the trusted subnet.
func CheckSubnet(next http.Handler) http.Handler {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		if config.Settings.TrustedSubnet == "" {
			http.Error(writer, "no trusted subnet specified", http.StatusForbidden)
			return
		} else if IPNet == nil {
			_, IPNet, _ = net.ParseCIDR(config.Settings.TrustedSubnet)
		}

		address, err := resolveIP(request)
		if err != nil || address == nil {
			http.Error(writer, "Unexpected error during IP parsing", http.StatusForbidden)
			return
		}

		if !IPNet.Contains(address) {
			http.Error(writer, "IP address not in trusted subnet", http.StatusForbidden)
			return
		}

		next.ServeHTTP(writer, request)
	}
	return http.HandlerFunc(fn)
}
