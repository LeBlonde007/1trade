# syntax=docker/dockerfile:1.6
# credit-ledger image. Mirrors deploy/docker/Dockerfile.go-service but with this service's module
# as the build context. Distroless + non-root + static binary. See docs/plans/DEPLOYMENT.md §2.
FROM golang:1.25-alpine AS build
WORKDIR /src
ARG VERSION=dev
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /out/app ./cmd/credit-ledger

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /usr/local/bin/app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
USER nonroot:nonroot
EXPOSE 8002
ENTRYPOINT ["/usr/local/bin/app"]
