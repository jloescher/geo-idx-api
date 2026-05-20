# Quantyra IDX API — Go multi-target image (Coolify / Dokploy)
# Targets: api | worker | scheduler

FROM golang:1.25-alpine AS build
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /var/cache/geoidx/images && chown -R nobody:nobody /var/cache/geoidx
COPY --from=build /out/api /usr/local/bin/api
COPY migrations /migrations
USER nobody
EXPOSE 8000
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8000/healthz || exit 1
CMD ["/usr/local/bin/api"]

FROM alpine:3.21 AS worker
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /var/cache/geoidx/images
COPY --from=build /out/worker /usr/local/bin/worker
USER nobody
CMD ["/usr/local/bin/worker"]

FROM alpine:3.21 AS scheduler
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/scheduler /usr/local/bin/scheduler
USER nobody
CMD ["/usr/local/bin/scheduler"]
