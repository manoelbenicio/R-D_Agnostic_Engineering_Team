package metrics

import "github.com/prometheus/client_golang/prometheus"

var credentialDurationBuckets = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}

type CredentialMetrics struct {
	credentialRestore             *prometheus.CounterVec
	credEnvInjection              *prometheus.CounterVec
	credentialPrepare             *prometheus.HistogramVec
	accountStatus                 *prometheus.GaugeVec
	accountTokensUsed             *prometheus.GaugeVec
	accountWindowSecondsRemaining *prometheus.GaugeVec
	accountsAvailable             *prometheus.GaugeVec
	allAccountsExhausted          *prometheus.GaugeVec
	rotation                      *prometheus.CounterVec
	rotationDuration              *prometheus.HistogramVec
	exhaustionDetected            *prometheus.CounterVec
}

func NewCredentialMetrics(registerers ...prometheus.Registerer) *CredentialMetrics {
	m := &CredentialMetrics{
		credentialRestore: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "credential_restore_total",
			Help: "Total credential restore attempts by vendor and result.",
		}, []string{"vendor", "result"}),
		credEnvInjection: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "cred_env_injection_total",
			Help: "Total credential environment injection attempts by vendor and result.",
		}, []string{"vendor", "result"}),
		credentialPrepare: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "credential_prepare_seconds",
			Help:    "Credential preparation duration in seconds by vendor.",
			Buckets: credentialDurationBuckets,
		}, []string{"vendor"}),
		accountStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "account_status",
			Help: "Current account status marker by vendor, account id, and status.",
		}, []string{"vendor", "account_id", "status"}),
		accountTokensUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "account_tokens_used",
			Help: "Current account token usage by vendor and account id.",
		}, []string{"vendor", "account_id"}),
		accountWindowSecondsRemaining: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "account_window_seconds_remaining",
			Help: "Current account quota window seconds remaining by vendor and account id.",
		}, []string{"vendor", "account_id"}),
		accountsAvailable: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "accounts_available",
			Help: "Current available account count by vendor.",
		}, []string{"vendor"}),
		allAccountsExhausted: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "all_accounts_exhausted",
			Help: "Whether all accounts are currently exhausted by vendor.",
		}, []string{"vendor"}),
		rotation: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "rotation_total",
			Help: "Total account rotation attempts by vendor, reason, and result.",
		}, []string{"vendor", "reason", "result"}),
		rotationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "rotation_duration_seconds",
			Help:    "Account rotation duration in seconds by vendor.",
			Buckets: credentialDurationBuckets,
		}, []string{"vendor"}),
		exhaustionDetected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "exhaustion_detected_total",
			Help: "Total account exhaustion detections by vendor and signal.",
		}, []string{"vendor", "signal"}),
	}
	for _, registerer := range registerers {
		if registerer != nil {
			registerer.MustRegister(m.Collectors()...)
		}
	}
	return m
}

func (m *CredentialMetrics) Collectors() []prometheus.Collector {
	if m == nil {
		return nil
	}
	return []prometheus.Collector{
		m.credentialRestore,
		m.credEnvInjection,
		m.credentialPrepare,
		m.accountStatus,
		m.accountTokensUsed,
		m.accountWindowSecondsRemaining,
		m.accountsAvailable,
		m.allAccountsExhausted,
		m.rotation,
		m.rotationDuration,
		m.exhaustionDetected,
	}
}

func (m *CredentialMetrics) ObserveRestore(vendor, result string) {
	if m == nil {
		return
	}
	m.credentialRestore.WithLabelValues(vendor, result).Inc()
}

func (m *CredentialMetrics) ObserveEnvInjection(vendor, result string) {
	if m == nil {
		return
	}
	m.credEnvInjection.WithLabelValues(vendor, result).Inc()
}

func (m *CredentialMetrics) ObservePrepare(vendor string, seconds float64) {
	if m == nil || seconds < 0 {
		return
	}
	m.credentialPrepare.WithLabelValues(vendor).Observe(seconds)
}

func (m *CredentialMetrics) SetAccountStatus(vendor, accountID, status string, value float64) {
	if m == nil {
		return
	}
	m.accountStatus.WithLabelValues(vendor, accountID, status).Set(value)
}

func (m *CredentialMetrics) SetAccountTokensUsed(vendor, accountID string, tokens float64) {
	if m == nil {
		return
	}
	m.accountTokensUsed.WithLabelValues(vendor, accountID).Set(tokens)
}

func (m *CredentialMetrics) SetAccountWindowSecondsRemaining(vendor, accountID string, seconds float64) {
	if m == nil {
		return
	}
	m.accountWindowSecondsRemaining.WithLabelValues(vendor, accountID).Set(seconds)
}

func (m *CredentialMetrics) SetAccountsAvailable(vendor string, count float64) {
	if m == nil {
		return
	}
	m.accountsAvailable.WithLabelValues(vendor).Set(count)
}

func (m *CredentialMetrics) SetAllAccountsExhausted(vendor string, exhausted bool) {
	if m == nil {
		return
	}
	value := 0.0
	if exhausted {
		value = 1
	}
	m.allAccountsExhausted.WithLabelValues(vendor).Set(value)
}

func (m *CredentialMetrics) ObserveRotation(vendor, reason, result string, seconds float64) {
	if m == nil {
		return
	}
	m.rotation.WithLabelValues(vendor, reason, result).Inc()
	if seconds >= 0 {
		m.rotationDuration.WithLabelValues(vendor).Observe(seconds)
	}
}

func (m *CredentialMetrics) ObserveExhaustionDetected(vendor, signal string) {
	if m == nil {
		return
	}
	m.exhaustionDetected.WithLabelValues(vendor, signal).Inc()
}
