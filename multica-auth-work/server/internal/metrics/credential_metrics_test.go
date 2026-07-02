package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewCredentialMetricsRegistersNoError(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	m := NewCredentialMetrics(registry)

	if m == nil {
		t.Fatal("NewCredentialMetrics returned nil")
	}
	if _, err := registry.Gather(); err != nil {
		t.Fatalf("gather registered credential metrics: %v", err)
	}
}

func TestCredentialMetricsEmissionValues(t *testing.T) {
	m := NewCredentialMetrics()

	m.ObserveRestore("codex", "ok")
	m.ObserveEnvInjection("codex", "error")
	m.ObservePrepare("codex", 1.5)
	m.SetAccountStatus("codex", "acct-1", "available", 1)
	m.SetAccountTokensUsed("codex", "acct-1", 1234)
	m.SetAccountWindowSecondsRemaining("codex", "acct-1", 300)
	m.SetAccountsAvailable("codex", 2)
	m.SetAllAccountsExhausted("codex", true)
	m.ObserveRotation("codex", "quota", "ok", 0.75)
	m.ObserveExhaustionDetected("codex", "ledger")

	assertMetricValue(t, "credential_restore_total", testutil.ToFloat64(m.credentialRestore.WithLabelValues("codex", "ok")), 1)
	assertMetricValue(t, "cred_env_injection_total", testutil.ToFloat64(m.credEnvInjection.WithLabelValues("codex", "error")), 1)
	assertHistogram(t, "credential_prepare_seconds", m.credentialPrepare, 1, 1.5)
	assertMetricValue(t, "account_status", testutil.ToFloat64(m.accountStatus.WithLabelValues("codex", "acct-1", "available")), 1)
	assertMetricValue(t, "account_tokens_used", testutil.ToFloat64(m.accountTokensUsed.WithLabelValues("codex", "acct-1")), 1234)
	assertMetricValue(t, "account_window_seconds_remaining", testutil.ToFloat64(m.accountWindowSecondsRemaining.WithLabelValues("codex", "acct-1")), 300)
	assertMetricValue(t, "accounts_available", testutil.ToFloat64(m.accountsAvailable.WithLabelValues("codex")), 2)
	assertMetricValue(t, "all_accounts_exhausted", testutil.ToFloat64(m.allAccountsExhausted.WithLabelValues("codex")), 1)
	assertMetricValue(t, "rotation_total", testutil.ToFloat64(m.rotation.WithLabelValues("codex", "quota", "ok")), 1)
	assertHistogram(t, "rotation_duration_seconds", m.rotationDuration, 1, 0.75)
	assertMetricValue(t, "exhaustion_detected_total", testutil.ToFloat64(m.exhaustionDetected.WithLabelValues("codex", "ledger")), 1)
}

func assertMetricValue(t *testing.T, name string, got, want float64) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func assertHistogram(t *testing.T, name string, collector prometheus.Collector, wantCount uint64, wantSum float64) {
	t.Helper()
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather %s: %v", name, err)
	}
	if len(families) != 1 {
		t.Fatalf("%s families = %d, want 1", name, len(families))
	}
	family := families[0]
	if family.GetName() != name {
		t.Fatalf("family name = %s, want %s", family.GetName(), name)
	}
	if len(family.GetMetric()) != 1 {
		t.Fatalf("%s metric count = %d, want 1", name, len(family.GetMetric()))
	}
	histogram := family.GetMetric()[0].GetHistogram()
	if histogram == nil {
		t.Fatalf("%s did not gather as histogram", name)
	}
	if histogram.GetSampleCount() != wantCount {
		t.Fatalf("%s sample count = %d, want %d", name, histogram.GetSampleCount(), wantCount)
	}
	if histogram.GetSampleSum() != wantSum {
		t.Fatalf("%s sample sum = %v, want %v", name, histogram.GetSampleSum(), wantSum)
	}
}
