package e2e

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ContractVersion is the schema-versioned correlation contract emitted by every
// span. Consumers MUST reject spans whose ContractVersion they do not support.
const ContractVersion = "agent-brain.e2e.v1"

// HopKind identifies one of the eight end-to-end hops. Hops 1..7 are emitted by
// their owning lane; HopTrace (8) is synthesized by the W5 assembler.
type HopKind string

const (
	HopIngress   HopKind = "ingress"   // 1 — control API (W6)
	HopQueue     HopKind = "queue"     // 2 — DB queue (W7)
	HopAdmission HopKind = "admission" // 3 — daemon admission/lifecycle (W1)
	HopCLI       HopKind = "cli"       // 4 — CLI process (W3)
	HopRoute     HopKind = "route"     // 5 — OmniRoute/provider (W2)
	HopPersist   HopKind = "persist"   // 6 — terminal persistence (W7)
	HopDelivery  HopKind = "delivery"  // 7 — WS/UI delivery (W6)
	HopTrace     HopKind = "trace"     // 8 — trace assembly (W5)
)

// EmittingHops returns the seven hops that owning lanes emit, in order.
func EmittingHops() []HopKind {
	return []HopKind{HopIngress, HopQueue, HopAdmission, HopCLI, HopRoute, HopPersist, HopDelivery}
}

// OrderedHops returns all eight hops in end-to-end order.
func OrderedHops() []HopKind {
	return append(EmittingHops(), HopTrace)
}

func isEmittingHop(h HopKind) bool {
	for _, k := range EmittingHops() {
		if k == h {
			return true
		}
	}
	return false
}

// IDField is the name of a correlation identifier.
type IDField string

const (
	IDRequest    IDField = "request_id"
	IDQueueMsg   IDField = "queue_msg_id"
	IDTask       IDField = "task_id"
	IDSession    IDField = "session_id"
	IDLaunch     IDField = "launch_id"
	IDProc       IDField = "proc_id"
	IDOmniReq    IDField = "omni_request_id"
	IDResult     IDField = "result_id"
	IDDelivery   IDField = "delivery_id"
)

// Correlation carries the nine metadata-only identifiers that stitch the eight
// hops together. Every field is optional at the type level but constrained
// per-hop by RequiredIDs; every non-empty value must be a safe identifier.
type Correlation struct {
	RequestID     string `json:"request_id,omitempty"`
	QueueMsgID    string `json:"queue_msg_id,omitempty"`
	TaskID        string `json:"task_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	LaunchID      string `json:"launch_id,omitempty"`
	ProcID        string `json:"proc_id,omitempty"`
	OmniRequestID string `json:"omni_request_id,omitempty"`
	ResultID      string `json:"result_id,omitempty"`
	DeliveryID    string `json:"delivery_id,omitempty"`
}

// Get returns the value of a named identifier field.
func (c Correlation) Get(f IDField) string {
	switch f {
	case IDRequest:
		return c.RequestID
	case IDQueueMsg:
		return c.QueueMsgID
	case IDTask:
		return c.TaskID
	case IDSession:
		return c.SessionID
	case IDLaunch:
		return c.LaunchID
	case IDProc:
		return c.ProcID
	case IDOmniReq:
		return c.OmniRequestID
	case IDResult:
		return c.ResultID
	case IDDelivery:
		return c.DeliveryID
	default:
		return ""
	}
}

// RequiredIDs returns the identifier fields a given hop MUST carry. These are
// the documented join keys (docs/observability/e2e-metadata-span.md §2).
func RequiredIDs(h HopKind) []IDField {
	switch h {
	case HopIngress:
		return []IDField{IDRequest, IDTask}
	case HopQueue:
		return []IDField{IDQueueMsg, IDTask}
	case HopAdmission:
		return []IDField{IDTask, IDSession, IDLaunch}
	case HopCLI:
		return []IDField{IDLaunch, IDProc}
	case HopRoute:
		return []IDField{IDRequest, IDOmniReq}
	case HopPersist:
		return []IDField{IDTask, IDResult}
	case HopDelivery:
		return []IDField{IDSession, IDDelivery}
	default:
		return nil
	}
}

// JoinRelationship documents how two adjacent identifiers are joined. This is
// the machine-readable form of the correlation contract for reviewers.
type JoinRelationship struct {
	Hop  HopKind   `json:"hop"`
	Keys []IDField `json:"keys"`
}

// JoinContract returns the full join contract for all emitting hops.
func JoinContract() []JoinRelationship {
	rels := make([]JoinRelationship, 0, len(EmittingHops()))
	for _, h := range EmittingHops() {
		rels = append(rels, JoinRelationship{Hop: h, Keys: RequiredIDs(h)})
	}
	return rels
}

// Carrier header/metadata keys for cross-hop propagation. Values are always
// safe identifiers; carriers never transport secrets or content.
const (
	HeaderContractVersion = "X-AB-E2E-Contract"
	HeaderRequestID       = "X-AB-Request-Id"
	HeaderQueueMsgID      = "X-AB-Queue-Msg-Id"
	HeaderTaskID          = "X-AB-Task-Id"
	HeaderSessionID       = "X-AB-Session-Id"
	HeaderLaunchID        = "X-AB-Launch-Id"
	HeaderProcID          = "X-AB-Proc-Id"
	HeaderOmniRequestID   = "X-AB-Omni-Request-Id"
	HeaderResultID        = "X-AB-Result-Id"
	HeaderDeliveryID      = "X-AB-Delivery-Id"
)

// ToCarrier renders the correlation as a propagation carrier (e.g. HTTP headers
// or message metadata). Empty identifiers are omitted. The contract version is
// always included.
func (c Correlation) ToCarrier() map[string]string {
	out := map[string]string{HeaderContractVersion: ContractVersion}
	put := func(k, v string) {
		if v != "" {
			out[k] = v
		}
	}
	put(HeaderRequestID, c.RequestID)
	put(HeaderQueueMsgID, c.QueueMsgID)
	put(HeaderTaskID, c.TaskID)
	put(HeaderSessionID, c.SessionID)
	put(HeaderLaunchID, c.LaunchID)
	put(HeaderProcID, c.ProcID)
	put(HeaderOmniRequestID, c.OmniRequestID)
	put(HeaderResultID, c.ResultID)
	put(HeaderDeliveryID, c.DeliveryID)
	return out
}

// CorrelationFromCarrier reconstructs a Correlation from a propagation carrier.
// Unknown keys are ignored. The result is not validated here; callers pass it
// through Span validation before use.
func CorrelationFromCarrier(carrier map[string]string) Correlation {
	return Correlation{
		RequestID:     carrier[HeaderRequestID],
		QueueMsgID:    carrier[HeaderQueueMsgID],
		TaskID:        carrier[HeaderTaskID],
		SessionID:     carrier[HeaderSessionID],
		LaunchID:      carrier[HeaderLaunchID],
		ProcID:        carrier[HeaderProcID],
		OmniRequestID: carrier[HeaderOmniRequestID],
		ResultID:      carrier[HeaderResultID],
		DeliveryID:    carrier[HeaderDeliveryID],
	}
}

// Validate enforces that every present identifier is a safe correlation token.
// Presence of the per-hop required identifiers is checked by Span.Validate.
func (c Correlation) Validate() error {
	fields := []struct {
		f IDField
		v string
	}{
		{IDRequest, c.RequestID}, {IDQueueMsg, c.QueueMsgID}, {IDTask, c.TaskID},
		{IDSession, c.SessionID}, {IDLaunch, c.LaunchID}, {IDProc, c.ProcID},
		{IDOmniReq, c.OmniRequestID}, {IDResult, c.ResultID}, {IDDelivery, c.DeliveryID},
	}
	for _, fv := range fields {
		if fv.v == "" {
			continue
		}
		if !safeID(fv.v, maxIDLen) {
			return fmt.Errorf("correlation %s is not a safe identifier", fv.f)
		}
	}
	return nil
}

const (
	maxIDLen        = 128
	maxCodeLen      = 96
	maxLabelValLen  = 128
	maxLabels       = 32
	maxCounters     = 32
	maxArgvTokens   = 64
)

// Span is a single metadata-only observability record for one hop. It contains
// no free-form content field by construction.
type Span struct {
	ContractVersion string            `json:"contract_version"`
	Hop             HopKind           `json:"hop"`
	Correlation     Correlation       `json:"correlation"`
	StartedAt       time.Time         `json:"started_at"`
	EndedAt         time.Time         `json:"ended_at"`
	Outcome         string            `json:"outcome"`
	ReasonCode      string            `json:"reason_code,omitempty"`
	HTTPStatus      int               `json:"http_status,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Counters        map[string]int64  `json:"counters,omitempty"`
	ArgvShape       []string          `json:"argv_shape,omitempty"`
	SecretsPresent  bool              `json:"secrets_present"`
}

// NewSpan starts a span for a hop with the given correlation. The returned span
// is not valid until an Outcome is set and Finish is called.
func NewSpan(hop HopKind, corr Correlation) *Span {
	return &Span{
		ContractVersion: ContractVersion,
		Hop:             hop,
		Correlation:     corr,
		StartedAt:       time.Now().UTC(),
		SecretsPresent:  false,
		Labels:          map[string]string{},
		Counters:        map[string]int64{},
	}
}

// WithLabel adds a bounded classification label (e.g. route_model, protocol).
func (s *Span) WithLabel(key, value string) *Span {
	if s.Labels == nil {
		s.Labels = map[string]string{}
	}
	s.Labels[key] = value
	return s
}

// WithCounter adds a numeric counter (e.g. latency_ms, byte_count).
func (s *Span) WithCounter(key string, value int64) *Span {
	if s.Counters == nil {
		s.Counters = map[string]int64{}
	}
	s.Counters[key] = value
	return s
}

// WithOutcome sets the terminal outcome and optional reason code.
func (s *Span) WithOutcome(outcome, reason string) *Span {
	s.Outcome = outcome
	s.ReasonCode = reason
	return s
}

// WithHTTPStatus sets an HTTP status classification.
func (s *Span) WithHTTPStatus(code int) *Span {
	s.HTTPStatus = code
	return s
}

// WithArgvShape sets the structural argv shape (shape tokens only, never
// values). See allowedArgvShapeTokens.
func (s *Span) WithArgvShape(shape []string) *Span {
	s.ArgvShape = shape
	return s
}

// Finish stamps the end time.
func (s *Span) Finish() *Span {
	s.EndedAt = time.Now().UTC()
	return s
}

// DurationMs returns the span duration in milliseconds (0 if not finished).
func (s *Span) DurationMs() int64 {
	if s.EndedAt.IsZero() || s.EndedAt.Before(s.StartedAt) {
		return 0
	}
	return s.EndedAt.Sub(s.StartedAt).Milliseconds()
}

// labelKind classifies how a label's VALUE is validated.
type labelKind int

const (
	kindCode labelKind = iota // strict safe identifier charset
	kindPath                  // route-template charset: adds '/', '{', '}'
)

// allowedLabelKeys is the closed set of safe classification label keys mapped to
// the value kind each accepts. Any other key is rejected as potentially
// content-bearing.
var allowedLabelKeys = map[string]labelKind{
	"route_model": kindCode, "protocol": kindCode, "cli_kind": kindCode, "router_owner": kindCode,
	"status_class": kindCode, "selection_reason": kindCode, "affinity_reason": kindCode,
	"quota_state": kindCode, "circuit_state": kindCode, "capacity_tier": kindCode, "cohort": kindCode,
	"admission_decision": kindCode, "readiness_result": kindCode, "fail_closed_class": kindCode,
	"terminal_status": kindCode, "exit_code_class": kindCode, "principal_class": kindCode,
	"method": kindCode, "route_template": kindPath, "route": kindPath, "token_class": kindCode,
	"backpressure_state": kindCode, "gap_reason": kindCode, "orphan_reason": kindCode,
}

// allowedArgvShapeTokens is the closed vocabulary for structural argv redaction.
// A shape describes the SHAPE of an argument, never its value.
var allowedArgvShapeTokens = map[string]struct{}{
	"subcommand":     {},
	"flag":           {},
	"flag=<redacted>": {},
	"arg=<redacted>":  {},
	"path=<redacted>": {},
	"value=<redacted>": {},
}

// Validate performs full structural validation of a span, including the
// metadata-only / secrets_present invariant and per-hop required identifiers.
func (s *Span) Validate() error {
	if s.ContractVersion != ContractVersion {
		return fmt.Errorf("unsupported e2e contract version %q", s.ContractVersion)
	}
	if !isEmittingHop(s.Hop) {
		return fmt.Errorf("hop %q is not an emitting hop", s.Hop)
	}
	if s.SecretsPresent {
		return fmt.Errorf("secrets_present invariant violated for hop %q", s.Hop)
	}
	if err := s.Correlation.Validate(); err != nil {
		return err
	}
	for _, req := range RequiredIDs(s.Hop) {
		if s.Correlation.Get(req) == "" {
			return fmt.Errorf("hop %q missing required correlation %s", s.Hop, req)
		}
	}
	if s.StartedAt.IsZero() {
		return fmt.Errorf("hop %q missing start time", s.Hop)
	}
	if !s.EndedAt.IsZero() && s.EndedAt.Before(s.StartedAt) {
		return fmt.Errorf("hop %q end precedes start", s.Hop)
	}
	if !safeCode(s.Outcome, maxCodeLen) {
		return fmt.Errorf("hop %q outcome must be a bounded safe code", s.Hop)
	}
	if s.ReasonCode != "" && !safeCode(s.ReasonCode, maxCodeLen) {
		return fmt.Errorf("hop %q reason code must be a bounded safe code", s.Hop)
	}
	if s.HTTPStatus < 0 || s.HTTPStatus > 599 {
		return fmt.Errorf("hop %q has invalid HTTP status", s.Hop)
	}
	if len(s.Labels) > maxLabels {
		return fmt.Errorf("hop %q exceeds label budget", s.Hop)
	}
	for k, v := range s.Labels {
		kind, ok := allowedLabelKeys[k]
		if !ok {
			return fmt.Errorf("hop %q uses unapproved label key %q", s.Hop, k)
		}
		if leak, reason := detectInlineSecret(v, kind); leak {
			return fmt.Errorf("hop %q label %q rejected: %s", s.Hop, k, reason)
		}
	}
	if len(s.Counters) > maxCounters {
		return fmt.Errorf("hop %q exceeds counter budget", s.Hop)
	}
	for k, v := range s.Counters {
		if !safeCode(k, maxCodeLen) {
			return fmt.Errorf("hop %q counter key %q is not a safe code", s.Hop, k)
		}
		if v < 0 {
			return fmt.Errorf("hop %q counter %q must not be negative", s.Hop, k)
		}
	}
	if len(s.ArgvShape) > maxArgvTokens {
		return fmt.Errorf("hop %q exceeds argv shape budget", s.Hop)
	}
	for _, tok := range s.ArgvShape {
		if _, ok := allowedArgvShapeTokens[tok]; !ok {
			return fmt.Errorf("hop %q argv shape token %q is not a redacted shape", s.Hop, tok)
		}
	}
	return nil
}

// safeID reports whether value is a bounded correlation identifier. The charset
// deliberately excludes '/', '+', '=', '@', and whitespace so that URL-,
// base64-, email-, and connection-string-shaped values are structurally
// rejected.
func safeID(value string, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == ':':
		default:
			return false
		}
	}
	return true
}

func safeCode(value string, max int) bool {
	return safeID(value, max)
}

// suspiciousSubstrings are secret/content markers that must never appear even
// inside an otherwise charset-valid value.
var suspiciousSubstrings = []string{
	"://", "@", "bearer ", "eyj", "sk-", "sk_", "ghp_", "gho_", "github_pat_",
	"akia", "asia", "xoxb-", "xoxp-", "-----begin", "authorization",
	"password", "passwd", "secret", "token=", "apikey", "api_key",
	"cookie", "set-cookie", "\n", "\r", "\t",
}

// detectInlineSecret performs a STRUCTURAL leak check on a single value: charset
// enforcement first (rejects most secret encodings by shape), then explicit
// suspicious-marker and length checks. It fails closed. The kind selects the
// charset: kindCode is a strict safe identifier; kindPath additionally allows
// '/', '{', '}' for route templates while still rejecting URLs ("://") and
// emails ("@").
func detectInlineSecret(value string, kind labelKind) (bool, string) {
	if value == "" {
		return false, ""
	}
	if len(value) > maxLabelValLen {
		return true, "value exceeds metadata length budget"
	}
	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' {
			return true, "value contains control/whitespace (possible free-form content)"
		}
		if r < 0x20 || r == 0x7f {
			return true, "value contains control character"
		}
	}
	if strings.ContainsAny(value, " ") {
		return true, "value contains whitespace (possible free-form content)"
	}
	lower := strings.ToLower(value)
	for _, marker := range suspiciousSubstrings {
		if strings.Contains(lower, marker) {
			return true, fmt.Sprintf("value contains prohibited marker %q", marker)
		}
	}
	// JWT-like structure: three dot-separated long segments.
	if segs := strings.Split(value, "."); len(segs) == 3 {
		long := 0
		for _, s := range segs {
			if len(s) >= 16 {
				long++
			}
		}
		if long >= 2 {
			return true, "value has JWT-like structure"
		}
	}
	if !safeLabelCharset(value, kind, maxLabelValLen) {
		return true, "value is outside the safe metadata charset"
	}
	return false, ""
}

// safeLabelCharset enforces the charset for a label value according to its kind.
func safeLabelCharset(value string, kind labelKind, max int) bool {
	if value == "" || len(value) > max {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == ':':
		case kind == kindPath && (r == '/' || r == '{' || r == '}'):
		default:
			return false
		}
	}
	return true
}

// ContractDescriptor is a serializable summary of the OBS-1 contract, suitable
// for evidence artifacts and independent review.
type ContractDescriptor struct {
	ContractVersion string             `json:"contract_version"`
	Hops            []HopKind          `json:"hops"`
	Identifiers     []IDField          `json:"identifiers"`
	Joins           []JoinRelationship `json:"joins"`
	Carriers        map[string]string  `json:"carriers"`
	SecretsInvariant string            `json:"secrets_invariant"`
}

// Descriptor returns the machine-readable OBS-1 contract descriptor.
func Descriptor() ContractDescriptor {
	ids := []IDField{IDRequest, IDQueueMsg, IDTask, IDSession, IDLaunch, IDProc, IDOmniReq, IDResult, IDDelivery}
	carriers := map[string]string{
		string(IDRequest):  HeaderRequestID,
		string(IDQueueMsg): HeaderQueueMsgID,
		string(IDTask):     HeaderTaskID,
		string(IDSession):  HeaderSessionID,
		string(IDLaunch):   HeaderLaunchID,
		string(IDProc):     HeaderProcID,
		string(IDOmniReq):  HeaderOmniRequestID,
		string(IDResult):   HeaderResultID,
		string(IDDelivery): HeaderDeliveryID,
	}
	return ContractDescriptor{
		ContractVersion:  ContractVersion,
		Hops:             OrderedHops(),
		Identifiers:      ids,
		Joins:            JoinContract(),
		Carriers:         carriers,
		SecretsInvariant: "secrets_present==false; metadata-only; argv shape-only",
	}
}

// sortedKeys is a small helper for deterministic iteration in reports.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
