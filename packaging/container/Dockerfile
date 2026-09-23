FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETARCH
ARG VERSION
LABEL org.opencontainers.image.title="tailcat-quic" \
      org.opencontainers.image.description="Independent QUIC/HTTP3-only Tailcat with BBRv3" \
      org.opencontainers.image.source="https://github.com/LiuTangLei/tailcat-quic" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.licenses="BSD-3-Clause"
COPY $TARGETARCH/tailcat /usr/local/bin/tailcat
ENTRYPOINT ["/usr/local/bin/tailcat"]
