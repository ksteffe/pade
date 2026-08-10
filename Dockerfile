# Multi-stage build for the pade-broker runtime image only (not the pade CLI).
# No secrets, policy, or bindings are baked into the image.

FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/pade-broker ./cmd/pade-broker

# static-debian12 includes CA certificates for outbound HTTPS (JWKS, Keeper).
# nonroot user (uid 65532). No shell/curl — platforms probe application /healthz.
# Docker HEALTHCHECK omitted: distroless has no HTTP client; Cloud Run/K8s probe /healthz.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/pade-broker /pade-broker
USER nonroot:nonroot
ENTRYPOINT ["/pade-broker"]
