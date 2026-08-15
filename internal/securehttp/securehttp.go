// Package securehttp validates authority endpoints and builds HTTP clients
// that refuse remote plaintext and HTTPS→HTTP downgrades.
package securehttp

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ValidateURL returns an error when rawURL is not an allowed authority endpoint.
// https is always allowed. http is allowed only for loopback hosts.
func ValidateURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	return validateParsed(u)
}

func validateParsed(u *url.URL) error {
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("URL must include scheme and host")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(u.Hostname()) {
			return nil
		}
		return fmt.Errorf("insecure http endpoint %q rejected; use https for remote hosts (http is allowed only for localhost/127.0.0.1/[::1])", u.Redacted())
	default:
		return fmt.Errorf("unsupported URL scheme %q (want https, or http for loopback)", u.Scheme)
	}
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Client returns an HTTP client with timeout and redirect protection.
// Redirects to disallowed URLs are rejected. HTTPS→HTTP downgrades are
// always rejected (including loopback) so credentials cannot follow a downgrade.
func Client(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if len(via) > 0 {
				prev := via[len(via)-1]
				if strings.EqualFold(prev.URL.Scheme, "https") && strings.EqualFold(req.URL.Scheme, "http") {
					return fmt.Errorf("redirect rejected: HTTPS to HTTP downgrade")
				}
			}
			if err := validateParsed(req.URL); err != nil {
				return fmt.Errorf("redirect rejected: %w", err)
			}
			return nil
		},
	}
}

// DefaultClient is a shared client for authority fetches.
func DefaultClient() *http.Client {
	return Client(30 * time.Second)
}
