package broker_test

import (
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/broker"
)

func TestPolicyRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	_, err := broker.ParsePolicy([]byte(`
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.local
policies:
  - subject: "user:42"
    requireRepoUrl: true
    repositories: ["github.com/ksteffe/pade"]
    capabilities: ["github.user.read"]
`))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "requireRepoUrl") {
		t.Fatalf("error should name unknown field: %v", err)
	}
}

func TestPolicyRejectsMissingRequireRepoURLs(t *testing.T) {
	t.Parallel()
	_, err := broker.ParsePolicy([]byte(`
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.local
policies:
  - subject: "user:42"
    repositories: ["github.com/ksteffe/pade"]
    capabilities: ["github.user.read"]
`))
	if err == nil {
		t.Fatal("expected requireRepoURLs required error")
	}
	if !strings.Contains(err.Error(), "requireRepoURLs") {
		t.Fatalf("error should mention requireRepoURLs: %v", err)
	}
}

func TestPolicyAcceptsExplicitFalseRequireRepoURLs(t *testing.T) {
	t.Parallel()
	p, err := broker.ParsePolicy([]byte(`
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.local
policies:
  - subject: "user:42"
    requireRepoURLs: false
    capabilities: ["github.user.read"]
`))
	if err != nil {
		t.Fatal(err)
	}
	d := p.Authorize(broker.Claims{Subject: "user:42"}, "github.user.read")
	if !d.Allowed {
		t.Fatalf("%+v", d)
	}
}
