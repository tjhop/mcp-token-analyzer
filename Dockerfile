ARG TARGETPLATFORM

FROM alpine:latest AS certs
ARG TARGETPLATFORM
RUN apk update && apk add ca-certificates

FROM cgr.dev/chainguard/busybox:latest
ARG TARGETPLATFORM
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

COPY $TARGETPLATFORM/mcp-token-analyzer /usr/bin/mcp-token-analyzer

USER nobody
ENTRYPOINT ["/usr/bin/mcp-token-analyzer"]
