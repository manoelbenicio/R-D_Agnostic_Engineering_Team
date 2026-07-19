package daemon

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-shellwords"

	"github.com/multica-ai/multica/server/internal/l2runtime"
)

const (
	runtimeRouterOwnerRustL2      = "rust_l2"
	runtimeRouterOwnerSourceL2    = "l2_start_session"
	rotationNoopReasonL2RouterOwn = "l2_router_owner"
)

type l2RuntimeClient interface {
	Health(context.Context) (*l2runtime.HealthResponse, error)
	Ready(context.Context) (*l2runtime.ReadyResponse, error)
	ApplyPolicy(context.Context, l2runtime.Policy) (*l2runtime.ApplyPolicyResponse, error)
	RegisterAccounts(context.Context, l2runtime.AccountRegistration) (*l2runtime.RegisterAccountsResponse, error)
	StartSession(context.Context, l2runtime.StartSessionRequest) (*l2runtime.StartSessionResponse, error)
	StopSession(context.Context, l2runtime.StopSessionRequest) error
	ApplyKillSwitch(context.Context, l2runtime.KillSwitch) (*l2runtime.KillSwitchResponse, error)
}

type runtimeRouterOwnerRecord struct {
	SessionID                   string
	RuntimeSessionID            string
	RuntimeRouterOwner          string
	RuntimeRouterOwnerSource    string
	RuntimeRouterOwnerStartedAt time.Time
	EventStreamURL              string
	RuntimeEndpoint             string
	RuntimeLogRef               string
}

type l2Sidecar struct {
	daemon *Daemon

	mu            sync.Mutex
	cmd           *exec.Cmd
	started       bool
	stopRequested bool
}

func (d *Daemon) initL2RuntimeClient() {
	if !d.cfg.L2Runtime.Enabled {
		return
	}
	client, err := l2runtime.NewClient(d.cfg.L2Runtime.BaseURL, d.cfg.L2Runtime.BearerToken, d.cfg.L2Runtime.Timeout)
	if err != nil {
		d.l2InitErr = err
		return
	}
	d.l2Client = client
	d.l2Sidecar = &l2Sidecar{daemon: d}
}

func (d *Daemon) startL2Runtime(ctx context.Context) error {
	if !d.cfg.L2Runtime.Enabled {
		return nil
	}
	if d.l2InitErr != nil {
		return fmt.Errorf("l2 runtime client unavailable: %w", d.l2InitErr)
	}
	if d.l2Client == nil {
		return fmt.Errorf("l2 runtime enabled but client is not configured")
	}
	if !d.cfg.Prodex.Enabled {
		return fmt.Errorf("l2 runtime enabled but prodex launch is disabled")
	}
	if d.l2Sidecar == nil {
		d.l2Sidecar = &l2Sidecar{daemon: d}
	}
	if err := d.reconcileProdexProfiles(ctx); err != nil {
		return err
	}
	if err := d.l2Sidecar.Start(ctx); err != nil {
		return err
	}
	if err := d.pushL2DesiredState(ctx); err != nil {
		_ = d.l2Sidecar.Stop(context.Background())
		return err
	}
	go d.l2HealthLoop(ctx)
	return nil
}

func (d *Daemon) stopL2Runtime() {
	if d.l2Sidecar == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.l2Sidecar.Stop(ctx); err != nil && d.logger != nil {
		d.logger.Warn("l2 sidecar stop failed", "error", err)
	}
}

func (d *Daemon) l2HealthLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthCtx, cancel := context.WithTimeout(ctx, d.cfg.L2Runtime.Timeout)
			err := d.l2Sidecar.Health(healthCtx)
			cancel()
			if err != nil && d.logger != nil {
				d.logger.Warn("l2 sidecar health failed", "error", err)
			}
		}
	}
}

func (s *l2Sidecar) Start(ctx context.Context) error {
	if s == nil || s.daemon == nil {
		return fmt.Errorf("l2 sidecar is not configured")
	}
	d := s.daemon
	args, err := l2SidecarArgs()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("l2 runtime enabled but MULTICA_L2_SIDECAR_ARGS is required")
	}

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	s.stopRequested = false
	s.mu.Unlock()

	go s.runLoop(ctx, args)

	readyCtx, cancel := context.WithTimeout(ctx, d.cfg.L2Runtime.Timeout)
	defer cancel()
	return s.Health(readyCtx)
}

func (s *l2Sidecar) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.stopRequested = true
	cmd := s.cmd
	s.mu.Unlock()

	if s.daemon != nil {
		s.daemon.stopL2Sessions(ctx)
	}
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)

	done := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return ctx.Err()
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		return fmt.Errorf("l2 sidecar did not stop gracefully")
	}
}

func (s *l2Sidecar) Health(ctx context.Context) error {
	if s == nil || s.daemon == nil || s.daemon.l2Client == nil {
		return fmt.Errorf("l2 sidecar health failed: client is not configured")
	}
	if _, err := s.daemon.l2Client.Health(ctx); err != nil {
		return err
	}
	if _, err := s.daemon.l2Client.Ready(ctx); err != nil {
		return err
	}
	return nil
}

func (s *l2Sidecar) runLoop(ctx context.Context, args []string) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		cmd := exec.CommandContext(ctx, s.daemon.cfg.L2Runtime.SidecarPath, args...)
		cmd.Env = prodexSidecarEnv(s.daemon.cfg)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		s.mu.Lock()
		if s.stopRequested {
			s.mu.Unlock()
			return
		}
		s.cmd = cmd
		s.mu.Unlock()

		err := cmd.Start()
		if err != nil {
			if s.daemon.logger != nil {
				s.daemon.logger.Warn("l2 sidecar start failed", "error", err)
			}
		} else {
			if s.daemon.logger != nil {
				s.daemon.logger.Info("l2 sidecar started", "path", s.daemon.cfg.L2Runtime.SidecarPath, "argument", l2SidecarSubcommand(args))
			}
			err = cmd.Wait()
		}

		s.mu.Lock()
		stopping := s.stopRequested
		if s.cmd == cmd {
			s.cmd = nil
		}
		s.mu.Unlock()
		if stopping || ctx.Err() != nil {
			return
		}
		if s.daemon.logger != nil {
			s.daemon.logger.Warn("l2 sidecar exited; restarting after backoff", "error", err, "backoff", backoff)
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func l2SidecarArgs(legacyProdexPath ...string) ([]string, error) {
	raw := strings.TrimSpace(os.Getenv("MULTICA_L2_SIDECAR_ARGS"))
	if raw == "" {
		if len(legacyProdexPath) > 0 {
			return nil, nil
		}
		return []string{"127.0.0.1:43117"}, nil
	}
	args, err := shellwords.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse MULTICA_L2_SIDECAR_ARGS: %w", err)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("MULTICA_L2_SIDECAR_ARGS is empty")
	}
	if len(legacyProdexPath) > 0 {
		return normalizeLegacyL2SidecarArgs(legacyProdexPath[0], args)
	}
	if strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("MULTICA_L2_SIDECAR_ARGS contains an empty first argument")
	}
	if strings.ContainsRune(args[0], os.PathSeparator) {
		return nil, fmt.Errorf("MULTICA_L2_SIDECAR_ARGS must contain adapter arguments, not an executable path")
	}
	return args, nil
}

func normalizeLegacyL2SidecarArgs(prodexPath string, args []string) ([]string, error) {
	first := strings.TrimSpace(args[0])
	if first == "" {
		return nil, fmt.Errorf("MULTICA_L2_SIDECAR_ARGS contains an empty first argument")
	}
	if isProdexExecutableToken(prodexPath, first) {
		args = args[1:]
		if len(args) == 0 {
			return nil, fmt.Errorf("MULTICA_L2_SIDECAR_ARGS must include a prodex subcommand")
		}
		first = strings.TrimSpace(args[0])
	}
	if strings.ContainsRune(first, os.PathSeparator) {
		return nil, fmt.Errorf("MULTICA_L2_SIDECAR_ARGS must launch configured prodex %q, not sidecar executable %q", prodexPath, first)
	}
	return args, nil
}

func isProdexExecutableToken(prodexPath, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" || filepath.Base(token) != "prodex" {
		return false
	}
	if prodexPath == "" || token == "prodex" {
		return true
	}
	if filepath.IsAbs(token) || strings.ContainsRune(token, os.PathSeparator) {
		absToken, tokenErr := filepath.Abs(token)
		absProdex, prodexErr := filepath.Abs(prodexPath)
		if tokenErr == nil && prodexErr == nil && absToken == absProdex {
			return true
		}
	}
	return token == prodexPath
}

func l2SidecarSubcommand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func (d *Daemon) pushL2DesiredState(ctx context.Context) error {
	if !d.cfg.L2Runtime.Enabled || d.l2Client == nil {
		return nil
	}
	tenantID := d.cfg.L2Runtime.TenantID
	policy := l2runtime.Policy{
		ControlEnvelope: l2runtime.ControlEnvelope{
			RequestID: fmt.Sprintf("l2_policy_%d", time.Now().UnixNano()),
			TenantID:  tenantID,
		},
		PolicyID:         d.cfg.L2Runtime.PolicyID,
		Revision:         time.Now().Unix(),
		AllowedProviders: []string{"codex"},
		AllowedProfiles:  d.l2ApprovedProfileIDs(),
		SmartContext: map[string]any{
			"mode":           "shadow",
			"canary_percent": 0,
		},
		KillSwitches: []l2runtime.KillSwitch{
			{
				ControlEnvelope: l2runtime.ControlEnvelope{
					RequestID: fmt.Sprintf("l2_kill_default_%d", time.Now().UnixNano()),
					TenantID:  tenantID,
				},
				Feature:     "auto_redeem",
				State:       "disabled",
				Reason:      "default_plan_03_01_guardrail",
				EffectiveAt: "next_request",
			},
		},
	}
	if policy.PolicyID == "" {
		policy.PolicyID = "default"
	}
	if _, err := d.l2Client.ApplyPolicy(ctx, policy); err != nil {
		return fmt.Errorf("l2 apply policy failed closed: %w", err)
	}
	profiles := d.l2ApprovedAccountProfiles()
	if len(profiles) == 0 {
		if d.logger != nil {
			d.logger.Info("l2 account registration skipped: no approved profiles discovered")
		}
		return nil
	}
	_, err := d.l2Client.RegisterAccounts(ctx, l2runtime.AccountRegistration{
		ControlEnvelope: l2runtime.ControlEnvelope{
			RequestID: fmt.Sprintf("l2_accounts_%d", time.Now().UnixNano()),
			TenantID:  tenantID,
		},
		Profiles: profiles,
	})
	if err != nil {
		return fmt.Errorf("l2 register accounts failed closed: %w", err)
	}
	return nil
}

func (d *Daemon) applyL2KillSwitch(ctx context.Context, req l2runtime.KillSwitch) error {
	if d.l2Client == nil {
		return fmt.Errorf("l2 kill switch failed closed: client is not configured")
	}
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("l2_kill_%d", time.Now().UnixNano())
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	_, err := d.l2Client.ApplyKillSwitch(ctx, req)
	if err != nil {
		return fmt.Errorf("l2 kill switch failed closed: %w", err)
	}
	return nil
}

func (d *Daemon) stopL2Sessions(ctx context.Context) {
	if d.l2Client == nil {
		return
	}
	d.l2SessionsMu.RLock()
	records := make([]runtimeRouterOwnerRecord, 0, len(d.l2Sessions))
	for _, rec := range d.l2Sessions {
		records = append(records, rec)
	}
	d.l2SessionsMu.RUnlock()
	for _, rec := range records {
		_ = d.l2Client.StopSession(ctx, l2runtime.StopSessionRequest{
			ControlEnvelope: l2runtime.ControlEnvelope{
				RequestID: fmt.Sprintf("l2_stop_%s_%d", shortID(rec.SessionID), time.Now().UnixNano()),
				TenantID:  "default",
			},
			SessionID:        rec.SessionID,
			RuntimeSessionID: rec.RuntimeSessionID,
			Reason:           "daemon_shutdown",
		})
	}
}

func (d *Daemon) l2ApprovedProfileIDs() []string {
	profiles := d.l2ApprovedAccountProfiles()
	ids := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		ids = append(ids, profile.ProfileID)
	}
	return ids
}

func (d *Daemon) l2ApprovedAccountProfiles() []l2runtime.AccountProfile {
	if profiles := d.reconciledProdexProfiles(); len(profiles) > 0 {
		return profiles
	}
	names := csvEnv("MULTICA_L2_APPROVED_PROFILES")
	if len(names) == 0 {
		names = prodexProfileDirs()
	}
	profiles := make([]l2runtime.AccountProfile, 0, len(names))
	prodexHome := strings.TrimSpace(os.Getenv("PRODEX_HOME"))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		home := name
		if prodexHome != "" && !filepath.IsAbs(home) {
			home = filepath.Join(prodexHome, "profiles", name)
		}
		profiles = append(profiles, l2runtime.AccountProfile{
			ProfileID:     name,
			Provider:      "codex",
			ProfileHome:   home,
			AuthMode:      "oauth_profile",
			Status:        "approved",
			CapabilityRef: "codex.oauth_profile.v1",
		})
	}
	return profiles
}

func csvEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func prodexProfileDirs() []string {
	prodexHome := strings.TrimSpace(os.Getenv("PRODEX_HOME"))
	if prodexHome == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(prodexHome, "profiles"))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func (d *Daemon) startL2SessionForTask(ctx context.Context, task *Task, provider, model, workDir string, taskLog *slog.Logger) (*runtimeRouterOwnerRecord, error) {
	return d.startL2SessionForTaskWithCredentialHome(ctx, task, provider, model, workDir, "", taskLog)
}

func (d *Daemon) startL2SessionForTaskWithCredentialHome(ctx context.Context, task *Task, provider, model, workDir, credentialHome string, taskLog *slog.Logger) (*runtimeRouterOwnerRecord, error) {
	if !d.cfg.L2Runtime.Enabled {
		return nil, nil
	}
	if task == nil {
		return nil, fmt.Errorf("l2 start session failed closed: task is nil")
	}
	if d.cfg.Prodex.Required && task.WorkspaceID != d.cfg.L2Runtime.TenantID {
		return nil, fmt.Errorf("l2 start session failed closed: task tenant does not match configured credential inventory")
	}
	if d.l2InitErr != nil {
		return nil, fmt.Errorf("l2 runtime client unavailable: %w", d.l2InitErr)
	}
	if d.l2Client == nil {
		return nil, fmt.Errorf("l2 runtime enabled but client is not configured")
	}
	if _, err := d.l2Client.Ready(ctx); err != nil {
		return nil, fmt.Errorf("l2 readiness failed closed: %w", err)
	}

	profilePool, err := d.l2ProfilePoolForTask(*task, provider, credentialHome)
	if err != nil {
		return nil, fmt.Errorf("l2 profile selection failed closed: %w", err)
	}
	req := l2runtime.StartSessionRequest{
		ControlEnvelope: l2runtime.ControlEnvelope{
			RequestID: fmt.Sprintf("l2_start_%s_%d", shortID(task.ID), time.Now().UnixNano()),
			TenantID:  task.WorkspaceID,
		},
		WorkspaceID:       task.WorkspaceID,
		TaskID:            task.ID,
		SessionID:         task.ID,
		PolicyID:          d.l2PolicyID(task),
		RequestedProvider: provider,
		RequestedModel:    model,
		WorkingDirectory:  workDir,
		ProfilePool:       profilePool,
		Continuation:      l2ContinuationForTask(*task),
	}
	resp, err := d.l2Client.StartSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("l2 start session failed closed: %w", err)
	}

	rec := runtimeRouterOwnerRecord{
		SessionID:                   task.ID,
		RuntimeSessionID:            strings.TrimSpace(resp.RuntimeSessionID),
		RuntimeRouterOwner:          taskRuntimeRouterOwner(Task{RuntimeRouterOwner: resp.RouterOwner}),
		RuntimeRouterOwnerSource:    runtimeRouterOwnerSourceL2,
		RuntimeRouterOwnerStartedAt: time.Now(),
		EventStreamURL:              strings.TrimSpace(resp.EventStreamURL),
		RuntimeEndpoint:             strings.TrimSpace(resp.RuntimeEndpoint),
		RuntimeLogRef:               strings.TrimSpace(resp.RuntimeLogRef),
	}
	if err := d.persistRuntimeRouterOwner(rec); err != nil {
		return nil, fmt.Errorf("persist l2 runtime router owner failed closed: %w", err)
	}
	task.RuntimeRouterOwner = rec.RuntimeRouterOwner
	if taskLog != nil {
		taskLog.Info("l2 runtime session started",
			"session_id", shortID(rec.SessionID),
			"runtime_session_id", shortID(rec.RuntimeSessionID),
			"runtime_router_owner", rec.RuntimeRouterOwner,
			"runtime_router_owner_source", rec.RuntimeRouterOwnerSource,
		)
	}
	return &rec, nil
}

func (d *Daemon) persistRuntimeRouterOwner(rec runtimeRouterOwnerRecord) error {
	rec.SessionID = strings.TrimSpace(rec.SessionID)
	rec.RuntimeSessionID = strings.TrimSpace(rec.RuntimeSessionID)
	rec.RuntimeRouterOwner = strings.ToLower(strings.TrimSpace(rec.RuntimeRouterOwner))
	rec.RuntimeRouterOwnerSource = strings.TrimSpace(rec.RuntimeRouterOwnerSource)
	if rec.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if rec.RuntimeSessionID == "" {
		return fmt.Errorf("runtime_session_id is required")
	}
	if rec.RuntimeRouterOwner != runtimeRouterOwnerRustL2 {
		return fmt.Errorf("runtime_router_owner = %q, want %q", rec.RuntimeRouterOwner, runtimeRouterOwnerRustL2)
	}
	if rec.RuntimeRouterOwnerSource == "" {
		return fmt.Errorf("runtime_router_owner_source is required")
	}
	if rec.RuntimeRouterOwnerStartedAt.IsZero() {
		rec.RuntimeRouterOwnerStartedAt = time.Now()
	}

	d.l2SessionsMu.Lock()
	defer d.l2SessionsMu.Unlock()
	if d.l2Sessions == nil {
		d.l2Sessions = make(map[string]runtimeRouterOwnerRecord)
	}
	d.l2Sessions[rec.SessionID] = rec
	return nil
}

func (d *Daemon) ingestL2RuntimeEvent(_ context.Context, event l2runtime.RuntimeEvent, taskLog *slog.Logger) error {
	// TODO(runtime-event-validation-spec): wire the ledger sink once
	// docs/contracts/runtime-event-validation-spec.md is final. Events remain
	// observability-only here and must never drive legacy Go rotation.
	if taskLog == nil {
		taskLog = d.logger
	}
	if taskLog != nil {
		taskLog.Info("l2 runtime event ingested",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"tenant_id", event.TenantID,
			"session_id", event.SessionID,
			"runtime_request_id", event.RuntimeRequestID,
			"severity", event.Severity,
		)
	}
	return nil
}

func (d *Daemon) runtimeRouterOwnerForTask(task Task) string {
	if owner := taskRuntimeRouterOwner(task); owner != "" {
		return owner
	}
	if task.ID == "" {
		return ""
	}
	d.l2SessionsMu.RLock()
	defer d.l2SessionsMu.RUnlock()
	return strings.ToLower(strings.TrimSpace(d.l2Sessions[task.ID].RuntimeRouterOwner))
}

func (d *Daemon) legacyGoRotationNoopReason(task Task) string {
	switch d.runtimeRouterOwnerForTask(task) {
	case runtimeRouterOwnerRustL2:
		return rotationNoopReasonL2RouterOwn
	case "omniroute":
		return "omniroute_router_owns"
	}
	return ""
}

func (d *Daemon) l2PolicyID(task *Task) string {
	if d.cfg.L2Runtime.PolicyID != "" {
		return d.cfg.L2Runtime.PolicyID
	}
	if task != nil && task.WorkspaceID != "" {
		return "workspace-" + task.WorkspaceID
	}
	return "default"
}

func (d *Daemon) l2ProfilePoolForTask(task Task, provider, credentialHome string) ([]string, error) {
	if strings.EqualFold(strings.TrimSpace(provider), "codex") && strings.TrimSpace(credentialHome) != "" {
		if profile, ok := d.prodexProfileForCredentialHome(credentialHome); ok {
			return []string{profile}, nil
		}
		if d.cfg.Prodex.Required {
			return nil, fmt.Errorf("assigned Codex credential home is not an approved Prodex profile")
		}
	}
	if task.RuntimeID != "" {
		return []string{task.RuntimeID}, nil
	}
	if task.AgentID != "" {
		return []string{task.AgentID}, nil
	}
	if provider != "" {
		return []string{provider}, nil
	}
	return nil, nil
}

func l2ContinuationForTask(task Task) map[string]string {
	continuation := make(map[string]string, 2)
	if task.PriorSessionID != "" {
		continuation["previous_response_id"] = task.PriorSessionID
	}
	if task.PriorWorkDir != "" {
		continuation["session_binding_hint"] = task.PriorWorkDir
	}
	if len(continuation) == 0 {
		return nil
	}
	return continuation
}
