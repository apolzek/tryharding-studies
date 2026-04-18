#!/usr/bin/env bash
# Dispara carga sintética (traces + logs + métricas) contra os 5 collectors
# usando telemetrygen (contrib v0.150.0). Cada collector recebe uma mistura
# diferente de serviços/taxas para que o cockpit mostre curvas distintas.
set -euo pipefail

COLLECTORS=(
  "otel-collector-1:4317"
  "otel-collector-2:4317"
  "otel-collector-3:4317"
  "otel-collector-4:4317"
  "otel-collector-5:4317"
)

SERVICES=("checkout" "cart" "catalog" "payments" "shipping" "auth" "frontend" "recommender")

run_traces() {
  local endpoint="$1" service="$2" rate="$3" duration="$4"
  telemetrygen traces \
    --otlp-endpoint "$endpoint" \
    --otlp-insecure \
    --service "$service" \
    --rate "$rate" \
    --duration "$duration" \
    --child-spans 3 \
    --status-code 0 \
    --otlp-attributes "deployment.environment=\"poc-041\"" \
    --telemetry-attributes "http.request.method=\"GET\"" \
    --telemetry-attributes "http.response.status_code=\"200\"" &
}

run_errors() {
  # telemetrygen mapeia o int 2 para STATUS_CODE_UNSET; só a string "Error"
  # marca o span como STATUS_CODE_ERROR de fato.
  local endpoint="$1" service="$2" rate="$3" duration="$4"
  telemetrygen traces \
    --otlp-endpoint "$endpoint" \
    --otlp-insecure \
    --service "$service" \
    --rate "$rate" \
    --duration "$duration" \
    --child-spans 2 \
    --status-code Error \
    --otlp-attributes "deployment.environment=\"poc-041\"" \
    --telemetry-attributes "http.request.method=\"POST\"" \
    --telemetry-attributes "http.response.status_code=\"500\"" &
}

run_logs() {
  # telemetrygen valida que severity-text bate com severity-number:
  # 9-12 = Info, 13-16 = Warn, 17-20 = Error. Passamos os dois juntos.
  local endpoint="$1" service="$2" rate="$3" duration="$4" severity="$5" text="$6"
  telemetrygen logs \
    --otlp-endpoint "$endpoint" \
    --otlp-insecure \
    --service "$service" \
    --rate "$rate" \
    --duration "$duration" \
    --severity-number "$severity" \
    --severity-text "$text" \
    --otlp-attributes "deployment.environment=\"poc-041\"" &
}

run_metrics() {
  local endpoint="$1" service="$2" rate="$3" duration="$4"
  telemetrygen metrics \
    --otlp-endpoint "$endpoint" \
    --otlp-insecure \
    --service "$service" \
    --rate "$rate" \
    --duration "$duration" \
    --otlp-attributes "deployment.environment=\"poc-041\"" &
}

echo "[loadgen] aguardando collectors ficarem prontos..."
sleep 15

while true; do
  for i in "${!COLLECTORS[@]}"; do
    endpoint="${COLLECTORS[$i]}"
    # Taxa escalonada por collector para variedade visual no cockpit.
    trace_rate=$(( 5 + i * 3 ))
    error_rate=$(( 1 + i ))
    log_info_rate=$(( 10 + i * 4 ))
    log_warn_rate=$(( 2 + i ))
    log_error_rate=$(( 1 + i / 2 ))
    metric_rate=$(( 4 + i * 2 ))

    svc_a="${SERVICES[$(( i % ${#SERVICES[@]} ))]}"
    svc_b="${SERVICES[$(( (i + 3) % ${#SERVICES[@]} ))]}"

    run_traces  "$endpoint" "$svc_a" "$trace_rate"     "55s"
    run_traces  "$endpoint" "$svc_b" "$(( trace_rate / 2 + 1 ))" "55s"
    run_errors  "$endpoint" "$svc_a" "$error_rate"     "55s"

    run_logs    "$endpoint" "$svc_a" "$log_info_rate"  "55s" 9  Info
    run_logs    "$endpoint" "$svc_b" "$log_warn_rate"  "55s" 13 Warn
    run_logs    "$endpoint" "$svc_a" "$log_error_rate" "55s" 17 Error

    run_metrics "$endpoint" "$svc_a" "$metric_rate"    "55s"
  done

  # Espera a janela terminar antes de iniciar a próxima para manter ~1 ciclo/min.
  wait
done
