package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ksteffe/pade/internal/binding"
)

// Server is the minimal PADE capability broker HTTP API.
type Server struct {
	Policy   *PolicyFile
	Verifier *Verifier
	Registry *binding.Registry
	Bindings *binding.Config
	Logger   *log.Logger
	Now      func() time.Time
}

type resolveRequest struct {
	Capability string `json:"capability"`
}

type resolveResponse struct {
	Env map[string]string `json:"env"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Handler returns the HTTP handler for the broker.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/resolve", s.handleResolve)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	var req resolveRequest
	jsonDec := json.NewDecoder(bytes.NewReader(raw))
	jsonDec.DisallowUnknownFields()
	if err := jsonDec.Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	capability := strings.TrimSpace(req.Capability)
	authzHeader := r.Header.Get("Authorization")
	token, ok := bearerToken(authzHeader)
	if !ok {
		s.logf("decision=deny capability=%q reason=missing_bearer", capability)
		s.writeError(w, http.StatusUnauthorized, "missing_bearer")
		return
	}

	claims, err := s.Verifier.Verify(r.Context(), token)
	if err != nil {
		s.logf("decision=deny capability=%q reason=verify_failed", capability)
		s.writeError(w, http.StatusUnauthorized, "token_invalid")
		return
	}

	dec := s.Policy.Authorize(claims, capability)
	if !dec.Allowed {
		s.logf("decision=deny subject=%q capability=%q reason=%s repos=%v", claims.Subject, capability, dec.Reason, claims.RepoURLs)
		s.writeError(w, http.StatusForbidden, "not_authorized")
		return
	}

	if s.Registry == nil || s.Bindings == nil {
		s.logf("decision=deny subject=%q capability=%q reason=bindings_unavailable", claims.Subject, capability)
		s.writeError(w, http.StatusInternalServerError, "bindings_unavailable")
		return
	}

	results, err := binding.ResolveMaterials(r.Context(), s.Registry, s.Bindings, []string{capability})
	if err != nil {
		s.logf("decision=deny subject=%q capability=%q reason=resolve_failed", claims.Subject, capability)
		s.writeError(w, http.StatusBadGateway, "resolve_failed")
		return
	}
	defer binding.ClearMaterials(results)
	if len(results) != 1 || results[0].Material == nil {
		s.writeError(w, http.StatusBadGateway, "resolve_failed")
		return
	}

	env := map[string]string{}
	for k, v := range results[0].Material.Env {
		env[k] = v
	}
	s.logf("decision=allow subject=%q capability=%q cloud_agent=%q repos=%v", claims.Subject, capability, claims.CloudAgentID, claims.RepoURLs)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(resolveResponse{Env: env})
}

func bearerToken(h string) (string, bool) {
	const p = "Bearer "
	if !strings.HasPrefix(h, p) {
		return "", false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, p))
	return tok, tok != ""
}

func (s *Server) writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}

// ListenAndServe starts the broker HTTP server after validating transport policy.
// Non-loopback plaintext is allowed only when TLSTermination is "proxy".
func ListenAndServe(ctx context.Context, cfg ListenConfig, h http.Handler) error {
	mode, err := cfg.Validate()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              strings.TrimSpace(cfg.Addr),
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	errCh := make(chan error, 1)
	go func() {
		if mode == TransportTLS {
			errCh <- srv.ListenAndServeTLS(strings.TrimSpace(cfg.CertFile), strings.TrimSpace(cfg.KeyFile))
			return
		}
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
