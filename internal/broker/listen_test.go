package broker_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/broker"
)

func TestListenConfigValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cfg     broker.ListenConfig
		want    broker.TransportMode
		wantErr string
	}{
		{
			name: "loopback http ipv4",
			cfg:  broker.ListenConfig{Addr: "127.0.0.1:8787"},
			want: broker.TransportLocal,
		},
		{
			name: "loopback http localhost",
			cfg:  broker.ListenConfig{Addr: "localhost:8787"},
			want: broker.TransportLocal,
		},
		{
			name: "loopback http ipv6",
			cfg:  broker.ListenConfig{Addr: "[::1]:8787"},
			want: broker.TransportLocal,
		},
		{
			name: "loopback with broker tls",
			cfg:  broker.ListenConfig{Addr: "127.0.0.1:8787", CertFile: "cert.pem", KeyFile: "key.pem"},
			want: broker.TransportTLS,
		},
		{
			name: "non-loopback with broker tls",
			cfg:  broker.ListenConfig{Addr: "0.0.0.0:8080", CertFile: "cert.pem", KeyFile: "key.pem"},
			want: broker.TransportTLS,
		},
		{
			name: "non-loopback proxy termination",
			cfg:  broker.ListenConfig{Addr: "0.0.0.0:8080", TLSTermination: broker.TLSTerminationProxy},
			want: broker.TransportTLSProxy,
		},
		{
			name: "empty host all interfaces with proxy",
			cfg:  broker.ListenConfig{Addr: ":8080", TLSTermination: "proxy"},
			want: broker.TransportTLSProxy,
		},
		{
			name:    "non-loopback http rejected by default",
			cfg:     broker.ListenConfig{Addr: "0.0.0.0:8080"},
			wantErr: "tls-termination=proxy",
		},
		{
			name:    "hostname treated as non-loopback without tls",
			cfg:     broker.ListenConfig{Addr: "broker.example.com:8787"},
			wantErr: "tls-termination=proxy",
		},
		{
			name:    "missing tls key",
			cfg:     broker.ListenConfig{Addr: "0.0.0.0:8080", CertFile: "cert.pem"},
			wantErr: "-tls-key is required",
		},
		{
			name:    "missing tls cert",
			cfg:     broker.ListenConfig{Addr: "0.0.0.0:8080", KeyFile: "key.pem"},
			wantErr: "-tls-cert is required",
		},
		{
			name: "proxy plus cert/key rejected",
			cfg: broker.ListenConfig{
				Addr:           "0.0.0.0:8080",
				CertFile:       "cert.pem",
				KeyFile:        "key.pem",
				TLSTermination: broker.TLSTerminationProxy,
			},
			wantErr: "incompatible",
		},
		{
			name:    "unknown tls-termination",
			cfg:     broker.ListenConfig{Addr: "127.0.0.1:8787", TLSTermination: "cloud-run"},
			wantErr: "invalid -tls-termination",
		},
		{
			name:    "malformed listen address",
			cfg:     broker.ListenConfig{Addr: "not-an-address"},
			wantErr: "invalid listen address",
		},
		{
			name:    "empty listen address",
			cfg:     broker.ListenConfig{Addr: ""},
			wantErr: "listen address is required",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mode, err := tc.cfg.Validate()
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got mode=%s", tc.wantErr, mode)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode != tc.want {
				t.Fatalf("mode=%q want=%q", mode, tc.want)
			}
		})
	}
}

func TestListenRequiresTLSForNonLoopback(t *testing.T) {
	t.Parallel()
	_, err := broker.ListenConfig{Addr: "0.0.0.0:0"}.Validate()
	if err == nil || !strings.Contains(err.Error(), "tls-termination=proxy") {
		t.Fatalf("err=%v", err)
	}
}

func TestListenProxyAllowsNonLoopbackPlaintext(t *testing.T) {
	t.Parallel()
	mode, err := broker.ListenConfig{
		Addr:           "0.0.0.0:8080",
		TLSTermination: broker.TLSTerminationProxy,
	}.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if mode != broker.TransportTLSProxy {
		t.Fatalf("mode=%q", mode)
	}
}

func TestHealthzNoAuth(t *testing.T) {
	t.Parallel()
	srv := &broker.Server{}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	resp, err := http.Get(hs.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if string(body) != "ok" {
		t.Fatalf("body=%q", body)
	}
}

// TestProxyTransportDoesNotBypassOIDC proves /v1/resolve auth is independent of
// transport mode: choosing tls-proxy deployment does not wrap or alter the mux.
func TestProxyTransportDoesNotBypassOIDC(t *testing.T) {
	t.Parallel()
	mode, err := broker.ListenConfig{
		Addr:           "0.0.0.0:8080",
		TLSTermination: broker.TLSTerminationProxy,
	}.Validate()
	if err != nil || mode != broker.TransportTLSProxy {
		t.Fatalf("mode=%s err=%v", mode, err)
	}

	srv := &broker.Server{
		Policy: &broker.PolicyFile{
			OIDC: broker.OIDCConfig{Issuer: testIssuer, Audience: testAudience},
			Policies: []broker.PolicyRule{{
				Subject:      testSubject,
				Capabilities: []string{"github.user.read"},
			}},
		},
		Verifier: &broker.Verifier{Issuer: testIssuer, Audience: testAudience, JWKSURL: "http://127.0.0.1:1/keys"},
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	resp := postResolve(t, hs.URL, "", "github.user.read")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401 (proxy mode must not bypass bearer auth)", resp.StatusCode)
	}
}
