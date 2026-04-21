---
title: OpenFeature — SDK padrão para feature flags (flagd como provider)
tags: [cncf, incubating, openfeature, feature-flags, flagd]
status: stable
---

## OpenFeature (CNCF Incubating)

**O que é:** API **padrão** para feature flags. A app importa o SDK OpenFeature (JS, Go, Python, Java, .NET, Ruby, PHP, Rust...) e o provider (flagd, LaunchDarkly, ConfigCat, Split, GrowthBook, Flagsmith) é pluggable. Mudar de vendor = trocar provider, não reescrever.

**Quando usar (SRE day-to-day):**

- Evitar vendor lock-in de FF — rolar LaunchDarkly hoje, Flagsmith amanhã, sem mexer em app.
- Self-hosted via **flagd** (provider oficial, k8s-friendly, arquivo/HTTP/S3 como source).
- Kill-switch e canário sem redeploy.
- Targeting por user/org/geo/cohort.

### Cenário real

*"Quero liberar `new-checkout` só para `beta@company.com` e `qa@company.com`. E um `discount-percent` com 50% dos users pegando 10%, 10% pegando 25%, 40% ficando sem desconto."*

### Reproducing

```bash
cd content/044/openfeature
docker compose up -d
sleep 3
```

Avalia flags via flagd gRPC/HTTP (porta 8013):

```bash
# boolean flag
curl -s -X POST http://localhost:8013/flagd.evaluation.v1.Service/ResolveBoolean \
  -H 'Content-Type: application/json' \
  -d '{"flagKey":"new-checkout","context":{"email":"beta@company.com"}}'
# → {"value":true,"reason":"TARGETING_MATCH","variant":"on"}

curl -s -X POST http://localhost:8013/flagd.evaluation.v1.Service/ResolveBoolean \
  -H 'Content-Type: application/json' \
  -d '{"flagKey":"new-checkout","context":{"email":"random@user.com"}}'
# → {"value":false,"reason":"DEFAULT","variant":"off"}

# int flag com fractional rollout
for u in alice bob carol dave eve frank grace henry ian jake; do
  v=$(curl -s -X POST http://localhost:8013/flagd.evaluation.v1.Service/ResolveInt \
    -H 'Content-Type: application/json' \
    -d "{\"flagKey\":\"discount-percent\",\"context\":{\"userId\":\"$u\"}}")
  echo "$u: $v"
done
```

### Em app (exemplo Go)

```go
import (
  "github.com/open-feature/go-sdk/openfeature"
  flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
)

openfeature.SetProvider(flagd.NewProvider())
client := openfeature.NewClient("checkout")

ctx := openfeature.NewEvaluationContext("user-123", map[string]interface{}{"email": "u@x.com"})
if on, _ := client.BooleanValue(context.Background(), "new-checkout", false, ctx); on {
    renderNewCheckout()
}
```

### Cleanup

```bash
docker compose down -v
```

### Tips de SRE

- **Kill-switch**: `state: DISABLED` no flag → `defaultVariant` sempre. Desliga feature ruim em 1 commit.
- **Source**: flagd carrega de arquivo (POC), HTTP, S3, GCS, ou sync-server — em prod use ConfigMap via volume + `inotify` reload.
- **Observabilidade**: flagd expõe Prometheus metrics em 8014. Alerte em `flagd_impressions_total` por flag.
- **Gradual rollout**: `fractionalEvaluation` baseado em hash do userId é **determinístico** — mesmo user, mesmo resultado. Essencial para A/B test.
- **OpenFeature Operator** (in-cluster): `FeatureFlagSource` + injeção sidecar flagd nos pods. Zero config no app.

### References

- https://openfeature.dev/docs/
- https://flagd.dev/
