# Playwright E2E tests

## Install

```bash
npm install
npx playwright install
```

## Run

Make sure the three frontends are running:

- React:   http://localhost:5173
- Vue:     http://localhost:5174
- Vanilla: http://localhost:5175

And the observability stack (Prometheus, Loki, Tempo, Grafana) is up.

```bash
npx playwright test                 # all projects + specs
npx playwright test --project=react
npm run test:smoke
npm run test:obs
```

## Env overrides

- `REACT_URL`, `VUE_URL`, `VANILLA_URL`
- `PROM_URL`, `LOKI_URL`, `TEMPO_URL`, `GRAFANA_URL`
