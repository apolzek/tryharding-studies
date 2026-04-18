# Imagem do telemetrygen (contrib) + script que dispara carga contra os 5 collectors.
FROM ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.150.0 AS tgen

FROM alpine:3.20
RUN apk add --no-cache bash coreutils
COPY --from=tgen /telemetrygen /usr/local/bin/telemetrygen
COPY generate.sh /usr/local/bin/generate.sh
RUN chmod +x /usr/local/bin/generate.sh
ENTRYPOINT ["/usr/local/bin/generate.sh"]
