# Tiny static binary for the Teleport Application Access dogfood.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ingress-demo .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget \
  && adduser -D -H -u 65532 nonroot
COPY --from=build /out/ingress-demo /usr/local/bin/ingress-demo
USER nonroot
EXPOSE 8080
ENV PORT=8080
HEALTHCHECK --interval=2s --timeout=2s --retries=15 CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
CMD ["ingress-demo"]
