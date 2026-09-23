# syntax=docker/dockerfile:1
# Multi-arch runtime image for tailcat-quic. The application is built from the
# selected Git tag/commit; no release binary is downloaded during the build.
FROM --platform=$BUILDPLATFORM golang:1.27.1-bookworm AS build

ARG TARGETOS=linux
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN set -eux; \
    goarm=""; \
    if [ "$TARGETARCH" = "arm" ]; then goarm=7; fi; \
    tags="$(cat build-tags.txt)"; \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" GOARM="$goarm" GOWORK=off \
      go build -mod=readonly -trimpath -tags="$tags" \
      -ldflags="-s -w -X main.version=$VERSION" \
      -o /out/tailcat ./cmd/tailcat; \
    test "$(/out/tailcat version)" = "$VERSION"

FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
LABEL org.opencontainers.image.title="tailcat-quic" \
      org.opencontainers.image.description="Independent QUIC/HTTP3-only Tailcat with BBRv3" \
      org.opencontainers.image.source="https://github.com/LiuTangLei/tailcat-quic" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.licenses="BSD-3-Clause"
COPY --from=build /out/tailcat /usr/local/bin/tailcat
ENTRYPOINT ["/usr/local/bin/tailcat"]
