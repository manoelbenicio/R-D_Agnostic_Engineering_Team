package e2e

import (
	"testing"
)

func fullCorrelation() Correlation {
	return Correlation{
		RequestID:     "req-001",
		QueueMsgID:    "qmsg-001",
		TaskID:        "task-001",
		SessionID:     "sess-001",
		LaunchID:      "launch-001",
		ProcID:        "proc-001",
		OmniRequestID: "omni-001",
		ResultID:      "result-001",
		DeliveryID:    "delivery-001",
	}
}

func TestContractVersionAndHops(t *testing.T) {
	if ContractVersion != "agent-brain.e2e.v1" {
		t.Fatalf("unexpected contract version %q", ContractVersion)
	}
	if got := len(EmittingHops()); got != 7 {
		t.Fatalf("expected 7 emitting hops, got %d", got)
	}
	if got := len(OrderedHops()); got != 8 {
		t.Fatalf("expected 8 ordered hops, got %d", got)
	}
	if OrderedHops()[7] != HopTrace {
		t.Fatalf("expected HopTrace last, got %q", OrderedHops()[7])
	}
}

func TestRequiredIDsMatchContract(t *testing.T) {
	want := map[HopKind][]IDField{
		HopIngress:   {IDRequest, IDTask},
		HopQueue:     {IDQueueMsg, IDTask},
		HopAdmission: {IDTask, IDSession, IDLaunch},
		HopCLI:       {IDLaunch, IDProc},
		HopRoute:     {IDRequest, IDOmniReq},
		HopPersist:   {IDTask, IDResult},
		HopDelivery:  {IDSession, IDDelivery},
	}
	for hop, exp := range want {
		got := RequiredIDs(hop)
		if len(got) != len(exp) {
			t.Fatalf("hop %q required ids len=%d want=%d", hop, len(got), len(exp))
		}
		for i := range exp {
			if got[i] != exp[i] {
				t.Fatalf("hop %q id[%d]=%q want %q", hop, i, got[i], exp[i])
			}
		}
	}
}

func TestCarrierRoundTrip(t *testing.T) {
	corr := fullCorrelation()
	carrier := corr.ToCarrier()
	if carrier[HeaderContractVersion] != ContractVersion {
		t.Fatalf("carrier missing contract version")
	}
	back, err := CorrelationFromCarrier(carrier)
	if err != nil {
		t.Fatalf("carrier rejected: %v", err)
	}
	if back != corr {
		t.Fatalf("carrier round-trip mismatch:\n got=%+v\nwant=%+v", back, corr)
	}
}

func TestCarrierVersionAndIdentifiersFailClosed(t *testing.T) {
	valid := fullCorrelation().ToCarrier()
	tests := []struct {
		name   string
		mutate func(map[string]string)
	}{
		{name: "missing version", mutate: func(carrier map[string]string) {
			delete(carrier, HeaderContractVersion)
		}},
		{name: "unsupported version", mutate: func(carrier map[string]string) {
			carrier[HeaderContractVersion] = "agent-brain.e2e.v999"
		}},
		{name: "unsafe identifier", mutate: func(carrier map[string]string) {
			carrier[HeaderRequestID] = "not safe content"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			carrier := make(map[string]string, len(valid))
			for key, value := range valid {
				carrier[key] = value
			}
			test.mutate(carrier)
			if _, err := CorrelationFromCarrier(carrier); err == nil {
				t.Fatal("unsafe carrier accepted")
			}
		})
	}
}

func TestCarrierOmitsEmpty(t *testing.T) {
	corr := Correlation{RequestID: "req-1", TaskID: "task-1"}
	carrier := corr.ToCarrier()
	if _, ok := carrier[HeaderSessionID]; ok {
		t.Fatalf("empty session id must not appear in carrier")
	}
}

func TestValidSpanPasses(t *testing.T) {
	s := NewSpan(HopIngress, Correlation{RequestID: "req-1", TaskID: "task-1"}).
		WithLabel("method", "POST").
		WithLabel("route_template", "/v1/tasks").
		WithLabel("principal_class", "service").
		WithCounter("latency_ms", 12).
		WithHTTPStatus(202).
		WithOutcome("accepted", "admitted").
		Finish()
	if err := s.Validate(); err != nil {
		t.Fatalf("expected valid span, got %v", err)
	}
}

func TestSecretsPresentInvariant(t *testing.T) {
	s := NewSpan(HopIngress, Correlation{RequestID: "req-1", TaskID: "task-1"}).
		WithOutcome("accepted", "").Finish()
	s.SecretsPresent = true
	if err := s.Validate(); err == nil {
		t.Fatalf("expected secrets_present invariant to fail closed")
	}
}

func TestMissingRequiredIDFails(t *testing.T) {
	// admission requires task/session/launch; omit launch.
	s := NewSpan(HopAdmission, Correlation{TaskID: "task-1", SessionID: "sess-1"}).
		WithOutcome("admitted", "").Finish()
	if err := s.Validate(); err == nil {
		t.Fatalf("expected missing launch_id to fail")
	}
}

func TestUnapprovedLabelKeyFails(t *testing.T) {
	s := NewSpan(HopIngress, Correlation{RequestID: "req-1", TaskID: "task-1"}).
		WithLabel("prompt", "safe").WithOutcome("accepted", "").Finish()
	if err := s.Validate(); err == nil {
		t.Fatalf("expected unapproved label key to fail")
	}
}

func TestLabelValueLeakRejected(t *testing.T) {
	cases := map[string]string{
		"email":      "user@example.com",
		"bearer":     "Bearer abc123",
		"url":        "https://host/x",
		"jwt":        "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig",
		"whitespace": "hello world",
		"apikey":     "sk-livesecretvalue",
		"conn":       "postgres://u:p@h:5432/db",
	}
	for name, val := range cases {
		s := NewSpan(HopRoute, Correlation{RequestID: "req-1", OmniRequestID: "omni-1"}).
			WithLabel("route_model", val).WithOutcome("ok", "").Finish()
		if err := s.Validate(); err == nil {
			t.Fatalf("case %q: expected leak value %q to be rejected", name, val)
		}
	}
}

func TestRouteModelAndPseudonymLabels(t *testing.T) {
	valid := NewSpan(HopRoute, Correlation{RequestID: "req-1", OmniRequestID: "omni-1"}).
		WithLabel("route_model", "agy/claude-opus-4-6-thinking").
		WithLabel("account_pseudonym", "acct_0123456789abcdef").
		WithLabel("connection_pseudonym", "conn_fedcba9876543210").
		WithOutcome("ok", "").
		Finish()
	if err := valid.Validate(); err != nil {
		t.Fatalf("safe route metadata rejected: %v", err)
	}
	principal := NewSpan(HopIngress, Correlation{RequestID: "req-1", TaskID: "task-1"}).
		WithLabel("principal_pseudonym", "principal_0123456789abcdef").
		WithOutcome("accepted", "").
		Finish()
	if err := principal.Validate(); err != nil {
		t.Fatalf("safe principal pseudonym rejected: %v", err)
	}

	invalid := []struct {
		key   string
		value string
	}{
		{key: "route_model", value: "https://provider.invalid/model"},
		{key: "route_model", value: "/provider/model"},
		{key: "route_model", value: "provider//model"},
		{key: "route_model", value: "provider/model with content"},
		{key: "account_pseudonym", value: "raw-account-name"},
		{key: "account_pseudonym", value: "acct_not-a-hex-digest"},
		{key: "connection_pseudonym", value: "conn_0123"},
		{key: "connection_pseudonym", value: "conn_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0"},
		{key: "principal_pseudonym", value: "user@example.com"},
	}
	for _, test := range invalid {
		span := NewSpan(HopRoute, Correlation{RequestID: "req-1", OmniRequestID: "omni-1"}).
			WithLabel(test.key, test.value).
			WithOutcome("ok", "").
			Finish()
		if err := span.Validate(); err == nil {
			t.Fatalf("unsafe %s accepted", test.key)
		}
	}
}

func TestCountersAreClosedPerHop(t *testing.T) {
	valid := NewSpan(HopRoute, Correlation{RequestID: "req-1", OmniRequestID: "omni-1"}).
		WithCounter("retry_count", 1).
		WithCounter("total_tokens", 42).
		WithOutcome("ok", "").
		Finish()
	if err := valid.Validate(); err != nil {
		t.Fatalf("approved route counters rejected: %v", err)
	}

	for _, key := range []string{"arbitrary_count", "queue_depth", "trace_count"} {
		span := NewSpan(HopRoute, Correlation{RequestID: "req-1", OmniRequestID: "omni-1"}).
			WithCounter(key, 1).
			WithOutcome("ok", "").
			Finish()
		if err := span.Validate(); err == nil {
			t.Fatalf("route accepted unapproved counter %q", key)
		}
	}
}

func TestArgvShapeStructural(t *testing.T) {
	ok := NewSpan(HopCLI, Correlation{LaunchID: "launch-1", ProcID: "proc-1"}).
		WithArgvShape([]string{"subcommand", "flag", "flag=<redacted>", "path=<redacted>"}).
		WithOutcome("exited", "code_0").Finish()
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid argv shape, got %v", err)
	}
	bad := NewSpan(HopCLI, Correlation{LaunchID: "launch-1", ProcID: "proc-1"}).
		WithArgvShape([]string{"--model=claude-3-opus"}).
		WithOutcome("exited", "code_0").Finish()
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected raw argv value to be rejected")
	}
}

func TestRecorderFailsClosed(t *testing.T) {
	sink := NewMemorySink()
	rec := NewRecorder(sink)

	good := NewSpan(HopQueue, Correlation{QueueMsgID: "q-1", TaskID: "task-1"}).
		WithCounter("wait_ms", 5).WithOutcome("dequeued", "").Finish()
	if err := rec.Emit(good); err != nil {
		t.Fatalf("expected good span to record, got %v", err)
	}
	bad := NewSpan(HopQueue, Correlation{QueueMsgID: "q-1", TaskID: "task-1"}).
		WithLabel("route_model", "user@example.com").WithOutcome("dequeued", "").Finish()
	if err := rec.Emit(bad); err == nil {
		t.Fatalf("expected leaking span to be refused")
	}
	if sink.Len() != 1 {
		t.Fatalf("expected exactly 1 recorded span, got %d", sink.Len())
	}
}

func TestNilRecorderSinkSafe(t *testing.T) {
	rec := NewRecorder(nil)
	s := NewSpan(HopPersist, Correlation{TaskID: "task-1", ResultID: "res-1"}).
		WithOutcome("persisted", "").Finish()
	if err := rec.Emit(s); err != nil {
		t.Fatalf("nil sink should be safe, got %v", err)
	}
}

func TestDescriptorShape(t *testing.T) {
	d := Descriptor()
	if d.ContractVersion != ContractVersion {
		t.Fatalf("descriptor version mismatch")
	}
	if len(d.Identifiers) != 9 {
		t.Fatalf("expected 9 identifiers, got %d", len(d.Identifiers))
	}
	if len(d.Joins) != 7 {
		t.Fatalf("expected 7 join relationships, got %d", len(d.Joins))
	}
	if len(d.Carriers) != 9 {
		t.Fatalf("expected 9 carriers, got %d", len(d.Carriers))
	}
}
