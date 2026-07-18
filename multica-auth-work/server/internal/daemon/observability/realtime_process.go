package observability

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrUnsupportedHostMetric = errors.New("unsupported host metric")
	errObservedProcessExited = errors.New("observed process exited")
)

const ProcessMeasurementSchemaVersion = "agent-brain.process-measurement.v1"
const syntheticAllowedPath = "/usr/bin:/bin"
const SyntheticExecutableIdentity = "synthetic-helper"

type UnsupportedHostMetricError struct {
	Metric string
	Reason string
}

func (e *UnsupportedHostMetricError) Error() string {
	return fmt.Sprintf("%v: %s: %s", ErrUnsupportedHostMetric, e.Metric, e.Reason)
}

func (e *UnsupportedHostMetricError) Unwrap() error { return ErrUnsupportedHostMetric }

type HostProcessSample struct {
	Elapsed         time.Duration `json:"elapsed_nanos"`
	CPUTime         time.Duration `json:"cpu_time_nanos"`
	RSSBytes        int64         `json:"rss_bytes"`
	PeakMemoryBytes int64         `json:"peak_memory_bytes"`
	OpenFDs         int           `json:"open_file_descriptors"`
	OpenSockets     int           `json:"open_sockets"`
}

type ProcessTermination string

const (
	ProcessCompleted ProcessTermination = "completed"
	ProcessFailed    ProcessTermination = "failed"
	ProcessCancelled ProcessTermination = "cancelled"
)

type ProcessMeasurement struct {
	SchemaVersion        string              `json:"schema_version"`
	Source               string              `json:"source"`
	ContentCapture       bool                `json:"content_capture"`
	Termination          ProcessTermination  `json:"termination"`
	ExitCode             int                 `json:"exit_code"`
	Duration             time.Duration       `json:"duration_nanos"`
	CPUTime              time.Duration       `json:"cpu_time_nanos"`
	LastRSSBytes         int64               `json:"last_rss_bytes"`
	PeakRSSBytes         int64               `json:"peak_rss_bytes"`
	PeakMemoryBytes      int64               `json:"peak_memory_bytes"`
	LastOpenFDs          int                 `json:"last_open_file_descriptors"`
	PeakOpenFDs          int                 `json:"peak_open_file_descriptors"`
	LastOpenSockets      int                 `json:"last_open_sockets"`
	PeakOpenSockets      int                 `json:"peak_open_sockets"`
	CancellationRelease  time.Duration       `json:"cancellation_release_nanos,omitempty"`
	Samples              []HostProcessSample `json:"samples"`
	SampleCadence        time.Duration       `json:"sample_cadence_nanos"`
	ContainmentValidated bool                `json:"containment_validated"`
	ExecutableName       string              `json:"trusted_executable_name"`
	ArgvCount            int                 `json:"validated_argv_count"`
	PathPolicy           string              `json:"path_policy"`
	ProcessTreeComplete  bool                `json:"process_tree_complete"`
	ProcessTreeResidual  string              `json:"process_tree_residual,omitempty"`
}

type ProcessMeasurementOptions struct {
	SampleInterval     time.Duration
	StartupTimeout     time.Duration
	SandboxRoot        string
	RequireProcessTree bool
}

type hostProcessSampler interface {
	Sample(pid int) (HostProcessSample, error)
	FinalPeakMemoryBytes(stateProcessState) (int64, error)
	Source() string
}

// stateProcessState is the content-free ProcessState surface used by the
// platform sampler. It avoids exposing a command or its environment.
type stateProcessState interface {
	UserTime() time.Duration
	SystemTime() time.Duration
	ExitCode() int
	Success() bool
	SysUsage() any
}

type OfflineRealtimeHarness struct {
	clock    monotonicClock
	sampler  hostProcessSampler
	recorder *RealtimeRecorder
}

func NewOfflineRealtimeHarness() (*OfflineRealtimeHarness, error) {
	clock := systemMonotonicClock{}
	sampler, err := newHostProcessSampler()
	if err != nil {
		return nil, err
	}
	return &OfflineRealtimeHarness{
		clock:    clock,
		sampler:  sampler,
		recorder: newRealtimeRecorder(clock),
	}, nil
}

func (h *OfflineRealtimeHarness) Recorder() *RealtimeRecorder { return h.recorder }

// MeasureSyntheticProcess starts one caller-supplied synthetic process and
// observes only numeric kernel/process counters. The command must have an
// explicit environment and ephemeral working directory so it cannot inherit
// host credential/routing state. Standard streams and extra file payloads are
// rejected. Network isolation and read-only source remain responsibilities of
// the outer disposable-container boundary.
func (h *OfflineRealtimeHarness) MeasureSyntheticProcess(
	ctx context.Context,
	cmd *exec.Cmd,
	options ProcessMeasurementOptions,
) (ProcessMeasurement, error) {
	if cmd == nil || cmd.Path == "" {
		return ProcessMeasurement{}, fmt.Errorf("synthetic process command is required")
	}
	if cmd.Process != nil {
		return ProcessMeasurement{}, fmt.Errorf("synthetic process is already started")
	}
	if cmd.Env == nil || cmd.Dir == "" {
		return ProcessMeasurement{}, fmt.Errorf("synthetic process requires explicit environment and working directory")
	}
	if cmd.Stdin != nil || cmd.Stdout != nil || cmd.Stderr != nil || len(cmd.ExtraFiles) != 0 {
		return ProcessMeasurement{}, fmt.Errorf("synthetic process content streams and extra files are forbidden")
	}
	if options.SampleInterval <= 0 || options.SampleInterval > time.Second {
		return ProcessMeasurement{}, fmt.Errorf("sample interval must be within (0, 1s]")
	}
	if options.StartupTimeout < 0 || options.StartupTimeout > 10*time.Second {
		return ProcessMeasurement{}, fmt.Errorf("startup timeout must be within [0, 10s]")
	}
	if options.StartupTimeout == 0 {
		options.StartupTimeout = 2 * time.Second
	}
	if err := validateSyntheticContainment(cmd, options.SandboxRoot); err != nil {
		return ProcessMeasurement{}, err
	}
	if options.RequireProcessTree {
		return ProcessMeasurement{}, &UnsupportedHostMetricError{Metric: "process-tree", Reason: "cgroup delegation is not safely available in this collector"}
	}
	if err := configureSyntheticProcess(cmd); err != nil {
		return ProcessMeasurement{}, err
	}
	if err := ctx.Err(); err != nil {
		return ProcessMeasurement{}, err
	}

	started := h.clock.Now()
	if err := cmd.Start(); err != nil {
		return ProcessMeasurement{}, fmt.Errorf("start synthetic process: %w", err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()

	result := ProcessMeasurement{
		SchemaVersion:        ProcessMeasurementSchemaVersion,
		Source:               h.sampler.Source(),
		ContentCapture:       false,
		ExitCode:             -1,
		SampleCadence:        options.SampleInterval,
		ContainmentValidated: true,
		ExecutableName:       SyntheticExecutableIdentity,
		ArgvCount:            len(cmd.Args),
		PathPolicy:           syntheticAllowedPath,
		ProcessTreeComplete:  false,
		ProcessTreeResidual:  "cgroup-delegation-unavailable; process-tree acceptance is STOP",
	}
	if err := h.observeInitialProcessSample(ctx, cmd.Process.Pid, started, options, waited, &result); err != nil {
		terminateSyntheticProcess(cmd)
		select {
		case <-waited:
		default:
		}
		return ProcessMeasurement{}, err
	}

	ticker := time.NewTicker(options.SampleInterval)
	defer ticker.Stop()
	cancelled := false
	var waitErr error
	for {
		select {
		case waitErr = <-waited:
			return h.finishProcessMeasurement(cmd, started, result, cancelled, waitErr)
		case <-ticker.C:
			err := h.observeProcessSample(cmd.Process.Pid, started, &result)
			if measurement, completed, finishErr := h.finishAfterCompletedWait(cmd, started, result, cancelled, err, waited, options.SampleInterval); completed {
				return measurement, finishErr
			}
			if err == nil || errors.Is(err, errObservedProcessExited) {
				continue
			}
			terminateSyntheticProcess(cmd)
			<-waited
			return ProcessMeasurement{}, err
		case <-ctx.Done():
			cancellation := h.recorder.BeginCancellationRelease()
			cancelled = true
			if err := terminateSyntheticProcess(cmd); err != nil && !errors.Is(err, os.ErrProcessDone) {
				return ProcessMeasurement{}, fmt.Errorf("cancel synthetic process: %w", err)
			}
			waitErr = <-waited
			duration, err := cancellation.End()
			if err != nil {
				return ProcessMeasurement{}, err
			}
			result.CancellationRelease = duration
			return h.finishProcessMeasurement(cmd, started, result, cancelled, waitErr)
		}
	}
}

func (h *OfflineRealtimeHarness) finishAfterCompletedWait(
	cmd *exec.Cmd,
	started time.Time,
	result ProcessMeasurement,
	cancelled bool,
	sampleErr error,
	waited <-chan error,
	reconcileWindow time.Duration,
) (ProcessMeasurement, bool, error) {
	if !errors.Is(sampleErr, ErrUnsupportedHostMetric) || reconcileWindow <= 0 {
		return ProcessMeasurement{}, false, nil
	}
	timer := time.NewTimer(reconcileWindow)
	defer timer.Stop()
	select {
	case waitErr := <-waited:
		measurement, err := h.finishProcessMeasurement(cmd, started, result, cancelled, waitErr)
		return measurement, true, err
	case <-timer.C:
		return ProcessMeasurement{}, false, nil
	}
}

func (h *OfflineRealtimeHarness) observeInitialProcessSample(
	ctx context.Context,
	pid int,
	started time.Time,
	options ProcessMeasurementOptions,
	waited <-chan error,
	result *ProcessMeasurement,
) error {
	deadline := h.clock.Now().Add(options.StartupTimeout)
	var lastErr error
	for {
		select {
		case <-waited:
			return errObservedProcessExited
		default:
		}
		lastErr = h.observeProcessSample(pid, started, result)
		if lastErr == nil {
			select {
			case <-waited:
				return errObservedProcessExited
			default:
			}
			return nil
		}
		if !errors.Is(lastErr, ErrUnsupportedHostMetric) && !errors.Is(lastErr, errObservedProcessExited) {
			return lastErr
		}
		if !h.clock.Now().Before(deadline) {
			return &UnsupportedHostMetricError{Metric: "process-samples", Reason: "no complete observed sample within startup window"}
		}

		timer := time.NewTimer(options.SampleInterval)
		select {
		case <-waited:
			return errObservedProcessExited
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func validateSyntheticContainment(cmd *exec.Cmd, sandboxRoot string) error {
	root, err := filepath.Abs(sandboxRoot)
	if err != nil || sandboxRoot == "" {
		return fmt.Errorf("synthetic process requires an absolute sandbox root")
	}
	if root != filepath.Clean(sandboxRoot) {
		return fmt.Errorf("synthetic process requires a canonical absolute sandbox root")
	}
	if !pathWithin(root, cmd.Dir) {
		return fmt.Errorf("synthetic process working directory must remain inside sandbox root")
	}

	requiredPaths := map[string]bool{
		"HOME": false, "XDG_CONFIG_HOME": false, "XDG_DATA_HOME": false,
		"XDG_CACHE_HOME": false, "TMPDIR": false,
	}
	seen := make(map[string]bool, len(cmd.Env))
	for _, entry := range cmd.Env {
		key, value, found := strings.Cut(entry, "=")
		if !found || key == "" || seen[key] {
			return fmt.Errorf("synthetic process environment contains an invalid or duplicate key")
		}
		seen[key] = true
		if _, required := requiredPaths[key]; required {
			if !pathWithin(root, value) {
				return fmt.Errorf("synthetic process %s must remain inside sandbox root", key)
			}
			requiredPaths[key] = true
			continue
		}
		switch key {
		case "PATH":
			if value != syntheticAllowedPath {
				return fmt.Errorf("synthetic process PATH is not the immutable allowlist")
			}
		case "LANG", "LC_ALL", "TZ":
			if !safeID(value, 64) {
				return fmt.Errorf("synthetic process locale environment is invalid")
			}
		default:
			if !strings.HasPrefix(key, "AGENT_BRAIN_SYNTHETIC_") || !safeID(key, 96) || !safeID(value, 128) {
				return fmt.Errorf("synthetic process environment key %q is not allowlisted", key)
			}
		}
	}
	for key, present := range requiredPaths {
		if !present {
			return fmt.Errorf("synthetic process environment lacks required isolated %s", key)
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	if candidate == "" || !filepath.IsAbs(candidate) {
		return false
	}
	clean := filepath.Clean(candidate)
	relative, err := filepath.Rel(root, clean)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (h *OfflineRealtimeHarness) observeProcessSample(pid int, started time.Time, result *ProcessMeasurement) error {
	sample, err := h.sampler.Sample(pid)
	if err != nil {
		return err
	}
	sample.Elapsed = h.clock.Now().Sub(started)
	if sample.Elapsed < 0 {
		return fmt.Errorf("monotonic clock moved backwards")
	}
	result.Samples = append(result.Samples, sample)
	result.CPUTime = maxDuration(result.CPUTime, sample.CPUTime)
	result.LastRSSBytes = sample.RSSBytes
	result.PeakRSSBytes = max64(result.PeakRSSBytes, sample.RSSBytes)
	result.PeakMemoryBytes = max64(result.PeakMemoryBytes, sample.PeakMemoryBytes)
	result.LastOpenFDs = sample.OpenFDs
	result.PeakOpenFDs = maxInt(result.PeakOpenFDs, sample.OpenFDs)
	result.LastOpenSockets = sample.OpenSockets
	result.PeakOpenSockets = maxInt(result.PeakOpenSockets, sample.OpenSockets)
	return nil
}

func (h *OfflineRealtimeHarness) finishProcessMeasurement(
	cmd *exec.Cmd,
	started time.Time,
	result ProcessMeasurement,
	cancelled bool,
	waitErr error,
) (ProcessMeasurement, error) {
	if len(result.Samples) == 0 || cmd.ProcessState == nil {
		return ProcessMeasurement{}, &UnsupportedHostMetricError{Metric: "process-samples", Reason: "no complete observed sample"}
	}
	result.Duration = h.clock.Now().Sub(started)
	if result.Duration < 0 {
		return ProcessMeasurement{}, fmt.Errorf("monotonic clock moved backwards")
	}
	result.ExitCode = cmd.ProcessState.ExitCode()
	result.CPUTime = maxDuration(result.CPUTime, cmd.ProcessState.UserTime()+cmd.ProcessState.SystemTime())
	peakMemory, err := h.sampler.FinalPeakMemoryBytes(cmd.ProcessState)
	if err != nil {
		return ProcessMeasurement{}, err
	}
	result.PeakMemoryBytes = max64(result.PeakMemoryBytes, peakMemory)
	if cancelled {
		result.Termination = ProcessCancelled
	} else if waitErr != nil || !cmd.ProcessState.Success() {
		result.Termination = ProcessFailed
	} else {
		result.Termination = ProcessCompleted
	}
	if err := result.Validate(); err != nil {
		return ProcessMeasurement{}, err
	}
	return result, nil
}

func (m ProcessMeasurement) Validate() error {
	if m.SchemaVersion != ProcessMeasurementSchemaVersion || m.Source != "linux-proc-and-rusage-observed" || m.ContentCapture || len(m.Samples) == 0 {
		return fmt.Errorf("incomplete host process measurements")
	}
	if m.Termination != ProcessCompleted && m.Termination != ProcessFailed && m.Termination != ProcessCancelled {
		return fmt.Errorf("invalid process termination")
	}
	if m.Duration < 0 || m.CPUTime < 0 || m.LastRSSBytes < 0 || m.PeakRSSBytes < m.LastRSSBytes || m.SampleCadence <= 0 || !m.ContainmentValidated ||
		m.PeakMemoryBytes < m.PeakRSSBytes || m.LastOpenFDs < 0 || m.PeakOpenFDs < m.LastOpenFDs ||
		m.LastOpenSockets < 0 || m.PeakOpenSockets < m.LastOpenSockets || m.PeakOpenSockets > m.PeakOpenFDs {
		return fmt.Errorf("invalid or unreconciled host process measurements")
	}
	if m.ExecutableName != SyntheticExecutableIdentity || m.ArgvCount <= 0 || m.PathPolicy != syntheticAllowedPath || m.ProcessTreeComplete || m.ProcessTreeResidual == "" {
		return fmt.Errorf("missing trusted executable or process-tree safety provenance")
	}
	if (m.Termination == ProcessCancelled && m.CancellationRelease <= 0) ||
		(m.Termination != ProcessCancelled && m.CancellationRelease != 0) {
		return fmt.Errorf("missing or invalid cancellation release measurement")
	}
	lastElapsed := time.Duration(-1)
	lastCPU := time.Duration(-1)
	for _, sample := range m.Samples {
		if sample.Elapsed < 0 || sample.Elapsed < lastElapsed || sample.Elapsed > m.Duration || sample.CPUTime < 0 || sample.CPUTime < lastCPU || sample.RSSBytes < 0 ||
			sample.PeakMemoryBytes < sample.RSSBytes || sample.OpenFDs < 0 || sample.OpenSockets < 0 || sample.OpenSockets > sample.OpenFDs {
			return fmt.Errorf("invalid host process sample")
		}
		lastElapsed = sample.Elapsed
		lastCPU = sample.CPUTime
	}
	if m.CPUTime < lastCPU {
		return fmt.Errorf("final CPU time does not reconcile with samples")
	}
	return nil
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
