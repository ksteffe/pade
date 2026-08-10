package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ksteffe/pade/internal/binding"
	envprovider "github.com/ksteffe/pade/internal/binding/env"
	keeperprovider "github.com/ksteffe/pade/internal/binding/keeper"
	keepersmprovider "github.com/ksteffe/pade/internal/binding/keepersm"
	onepasswordprovider "github.com/ksteffe/pade/internal/binding/onepassword"
	vaultprovider "github.com/ksteffe/pade/internal/binding/vault"
	"github.com/ksteffe/pade/internal/broker"
)

func main() {
	var (
		listen = flag.String("listen", "",
			"listen address. Empty: use PORT env as 0.0.0.0:PORT, else "+broker.DefaultListenAddr+". Non-loopback plaintext requires -tls-termination=proxy or -tls-cert/-tls-key")
		policy   = flag.String("policy", "", "path to broker-policy.yaml (required)")
		bindings = flag.String("bindings", "", "path to server-side bindings.yaml (required)")
		certFile = flag.String("tls-cert", "", "TLS certificate file (broker-managed TLS; use with -tls-key)")
		keyFile  = flag.String("tls-key", "", "TLS private key file (broker-managed TLS; use with -tls-cert)")
		tlsTerm  = flag.String("tls-termination", "",
			`TLS ownership model: empty (default safe auto) or "proxy". `+
				`proxy asserts that a trusted upstream (Cloud Run, ingress, load balancer, etc.) terminates TLS `+
				`and that this plaintext listener is reachable only inside that deployment boundary. `+
				`PADE does not verify the proxy. Incompatible with -tls-cert/-tls-key.`)
	)
	flag.Parse()
	if *policy == "" || *bindings == "" {
		fmt.Fprintln(os.Stderr, "usage: pade-broker -policy FILE -bindings FILE [-listen ADDR] [-tls-cert FILE -tls-key FILE | -tls-termination=proxy]")
		os.Exit(2)
	}

	addr, err := broker.ResolveListenAddr(*listen, os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	listenCfg := broker.ListenConfig{
		Addr:           addr,
		CertFile:       *certFile,
		KeyFile:        *keyFile,
		TLSTermination: *tlsTerm,
	}
	mode, err := listenCfg.Validate()
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	pol, err := broker.LoadPolicy(*policy)
	if err != nil {
		log.Fatalf("policy: %v", err)
	}
	bindCfg, err := binding.Load(*bindings)
	if err != nil {
		log.Fatalf("bindings: %v", err)
	}

	jwksURL := pol.OIDC.JWKSURL
	if jwksURL == "" {
		jwksURL = "https://api.cursor.com/keys"
	}
	srv := &broker.Server{
		Policy: pol,
		Verifier: &broker.Verifier{
			Issuer:   pol.OIDC.Issuer,
			Audience: pol.OIDC.Audience,
			JWKSURL:  jwksURL,
		},
		Registry: binding.NewRegistry(
			envprovider.New(),
			vaultprovider.New(),
			onepasswordprovider.New(),
			keeperprovider.New(),
			keepersmprovider.New(),
		),
		Bindings: bindCfg,
		Logger:   log.New(os.Stderr, "pade-broker: ", log.LstdFlags),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("pade-broker listening on %s transport=%s (issuer=%s audience=%s)", addr, mode, pol.OIDC.Issuer, pol.OIDC.Audience)
	if err := broker.ListenAndServe(ctx, listenCfg, srv.Handler()); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
