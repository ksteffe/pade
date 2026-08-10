// Command ingress-demo is a tiny HTTP server for the Milestone 8 Teleport ingress dogfood.
package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		host, _ := os.Hostname()
		now := time.Now().UTC().Format(time.RFC3339)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>PADE ingress demo</title>
<style>
  body { font-family: ui-sans-serif, system-ui, sans-serif; margin: 2rem; line-height: 1.5; max-width: 40rem; }
  h1 { font-size: 1.75rem; margin-bottom: 0.25rem; }
  .note { color: #333; }
  code { background: #f4f4f4; padding: 0.1rem 0.35rem; }
</style>
</head>
<body>
  <h1>PADE ingress demo</h1>
  <p class="note">This page is the private HTTP app. Through Teleport Application Access it should only appear after login; on <code>localhost:8080</code> it is intentionally open.</p>
  <p>Hostname: <code>%s</code></p>
  <p>UTC time: <code>%s</code></p>
  <p>Request host: <code>%s</code></p>
</body>
</html>
`, html.EscapeString(host), html.EscapeString(now), html.EscapeString(r.Host))
	})

	addr := ":" + port
	log.Printf("pade ingress-demo listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
