package broker

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// TransportMode is how the broker exposes HTTP relative to TLS.
type TransportMode string

const (
	// TransportLocal is plaintext on a loopback bind address.
	TransportLocal TransportMode = "local"
	// TransportTLS is broker-managed TLS (cert/key provided).
	TransportTLS TransportMode = "tls"
	// TransportTLSProxy is plaintext behind a trusted upstream that terminates TLS.
	// This is an operator deployment assertion — PADE does not verify the proxy.
	TransportTLSProxy TransportMode = "tls-proxy"
)

// TLSTerminationProxy is the only non-empty -tls-termination value.
const TLSTerminationProxy = "proxy"

// DefaultListenAddr is used when -listen is empty and PORT is unset.
const DefaultListenAddr = "127.0.0.1:8787"

// ListenConfig is the broker listener / TLS transport configuration.
type ListenConfig struct {
	Addr           string
	CertFile       string
	KeyFile        string
	TLSTermination string // "" (default) or "proxy"
}

// ResolveListenAddr chooses the bind address for container-friendly deploys.
// Explicit flagListen wins. Otherwise PORT (digits only) yields 0.0.0.0:PORT.
// If both are empty, DefaultListenAddr is used. PORT alone does not opt into
// trusted upstream TLS termination.
func ResolveListenAddr(flagListen, portEnv string) (string, error) {
	if addr := strings.TrimSpace(flagListen); addr != "" {
		return addr, nil
	}
	port := strings.TrimSpace(portEnv)
	if port == "" {
		return DefaultListenAddr, nil
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid PORT %q (want integer 1-65535)", portEnv)
	}
	return fmt.Sprintf("0.0.0.0:%d", n), nil
}

// Validate checks listen/TLS combinations before binding a socket.
// Non-loopback plaintext is allowed only with explicit TLSTermination=proxy.
func (c ListenConfig) Validate() (TransportMode, error) {
	addr := strings.TrimSpace(c.Addr)
	if addr == "" {
		return "", fmt.Errorf("listen address is required")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q: %w", addr, err)
	}

	term := strings.TrimSpace(strings.ToLower(c.TLSTermination))
	switch term {
	case "", TLSTerminationProxy:
	default:
		return "", fmt.Errorf("invalid -tls-termination %q (want empty or %q)", c.TLSTermination, TLSTerminationProxy)
	}

	cert := strings.TrimSpace(c.CertFile)
	key := strings.TrimSpace(c.KeyFile)
	hasCert := cert != ""
	hasKey := key != ""
	if hasCert != hasKey {
		if hasCert {
			return "", fmt.Errorf("-tls-key is required when -tls-cert is set")
		}
		return "", fmt.Errorf("-tls-cert is required when -tls-key is set")
	}
	hasTLSFiles := hasCert && hasKey

	if term == TLSTerminationProxy && hasTLSFiles {
		return "", fmt.Errorf("-tls-termination=%s is incompatible with -tls-cert/-tls-key (choose proxy termination or broker-managed TLS)", TLSTerminationProxy)
	}

	loopback := isLoopbackHost(host)

	if hasTLSFiles {
		return TransportTLS, nil
	}
	if term == TLSTerminationProxy {
		return TransportTLSProxy, nil
	}
	if loopback {
		return TransportLocal, nil
	}
	return "", fmt.Errorf("non-loopback listen address %q requires -tls-cert/-tls-key or -tls-termination=%s", addr, TLSTerminationProxy)
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
