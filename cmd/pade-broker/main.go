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
		listen   = flag.String("listen", "127.0.0.1:8787", "listen address (non-loopback requires TLS)")
		policy   = flag.String("policy", "", "path to broker-policy.yaml (required)")
		bindings = flag.String("bindings", "", "path to server-side bindings.yaml (required)")
		certFile = flag.String("tls-cert", "", "TLS certificate file")
		keyFile  = flag.String("tls-key", "", "TLS private key file")
	)
	flag.Parse()
	if *policy == "" || *bindings == "" {
		fmt.Fprintln(os.Stderr, "usage: pade-broker -policy FILE -bindings FILE [-listen ADDR] [-tls-cert FILE -tls-key FILE]")
		os.Exit(2)
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

	log.Printf("pade-broker listening on %s (issuer=%s audience=%s)", *listen, pol.OIDC.Issuer, pol.OIDC.Audience)
	if err := broker.ListenAndServe(ctx, *listen, srv.Handler(), *certFile, *keyFile); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
