// Integration tests that don't need a live cluster.
//
//  - helm_test.go: runs `helm template` on the tenant-stack chart with a
//    realistic values set, asserts every expected resource kind/name is
//    rendered. This is the fastest way to catch template regressions.
//
// Running against a real kind cluster (optional):
//   INTEGRATION_KIND=1 go test ./...
package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// chartPath resolves to the repo-relative tenant-stack chart.
func chartPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(wd, "..", "..", "charts", "tenant-stack")
	if _, err := os.Stat(filepath.Join(p, "Chart.yaml")); err != nil {
		t.Fatalf("chart path not found: %v", err)
	}
	return p
}

func helmTemplate(t *testing.T, chart string, sets ...string) string {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not installed; skipping template test")
	}
	args := []string{"template", "tenant", chart, "--namespace", "tenant-t-test"}
	for _, s := range sets {
		args = append(args, "--set", s)
	}
	cmd := exec.Command("helm", args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("helm template: %v\n%s", err, errb.String())
	}
	return out.String()
}

func TestChartRendersAllComponents(t *testing.T) {
	out := helmTemplate(t, chartPath(t),
		"tenant.id=t-test",
		"tenant.grafanaAdminPassword=dev-password-123", // pragma: allowlist secret
		"tenant.ingestJWT.secret=0123456789abcdef0123456789abcdef", // pragma: allowlist secret
		"tenant.ingestJWT.expectedTid=t-test",
		"clickhouse.password=chpw", // pragma: allowlist secret
	)
	// Every top-level component must be present.
	required := []string{
		"kind: ResourceQuota",
		"kind: LimitRange",
		"kind: NetworkPolicy",
		"name: clickhouse",
		"name: victoriametrics",
		"name: jaeger",
		"name: otel-collector",
		"name: grafana",
		"kind: Ingress",
	}
	for _, r := range required {
		if !strings.Contains(out, r) {
			t.Errorf("chart is missing %q", r)
		}
	}
	// Sanity: tenant id propagated.
	if !strings.Contains(out, "t-test") {
		t.Error("tenant.id did not propagate into rendered resources")
	}
}

func TestChartRejectsMissingRequired(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not installed")
	}
	// No overrides — should fail the guard template.
	cmd := exec.Command("helm", "template", "tenant", chartPath(t))
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err == nil {
		t.Fatal("expected helm template to fail without required values")
	}
	if !strings.Contains(errb.String(), "required") && !strings.Contains(errb.String(), "tenant.id") {
		t.Logf("stderr: %s", errb.String())
	}
}

func TestIngestHostsUseTenantID(t *testing.T) {
	out := helmTemplate(t, chartPath(t),
		"tenant.id=t-xyz",
		"tenant.ingestDomain=example.dev",
		"tenant.grafanaAdminPassword=dev-password-123", // pragma: allowlist secret
		"tenant.ingestJWT.secret=0123456789abcdef0123456789abcdef", // pragma: allowlist secret
		"tenant.ingestJWT.expectedTid=t-xyz",
		"clickhouse.password=chpw", // pragma: allowlist secret
	)
	for _, h := range []string{"t-xyz-ingest.example.dev", "t-xyz-grafana.example.dev"} {
		if !strings.Contains(out, h) {
			t.Errorf("expected host %q in rendered ingress", h)
		}
	}
}
