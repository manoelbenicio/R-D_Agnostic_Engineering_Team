//go:build linux

package observability

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const syntheticHelperMode = "AGENT_BRAIN_SYNTHETIC_PROCESS_MODE"

func TestSyntheticProcessHelper(t *testing.T) {
	mode := os.Getenv(syntheticHelperMode)
	if mode == "" {
		return
	}
	switch mode {
	case "measure":
		memory := make([]byte, 8<<20)
		for index := 0; index < len(memory); index += os.Getpagesize() {
			memory[index] = byte(index)
		}
		pipes := make([]*os.File, 0, 8)
		for range 4 {
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatalf("create synthetic pipe: %v", err)
			}
			pipes = append(pipes, reader, writer)
		}
		deadline := time.Now().Add(220 * time.Millisecond)
		value := uint64(1)
		for time.Now().Before(deadline) {
			value = value*1664525 + 1013904223
		}
		for _, file := range pipes {
			_ = file.Close()
		}
		runtime.KeepAlive(memory)
		runtime.KeepAlive(value)
	case "cancel":
		time.Sleep(5 * time.Second)
	case "exit":
		return
	default:
		t.Fatalf("unknown synthetic helper mode")
	}
}

func TestOfflineRealtimeHarnessStopsWhenProcessExitsBeforeFirstSample(t *testing.T) {
	harness, err := NewOfflineRealtimeHarness()
	if err != nil {
		t.Fatalf("new offline real-time harness: %v", err)
	}
	waited := make(chan error)
	close(waited)
	result := ProcessMeasurement{}
	err = harness.observeInitialProcessSample(context.Background(), os.Getpid(), time.Now(), ProcessMeasurementOptions{SampleInterval: 50 * time.Millisecond, StartupTimeout: time.Second}, waited, &result)
	if !errors.Is(err, errObservedProcessExited) {
		t.Fatalf("early process exit was not reported fail-fast: %v", err)
	}
}

func TestOfflineRealtimeHarnessRejectsProcessTreeWithoutCgroupDelegation(t *testing.T) {
	harness, err := NewOfflineRealtimeHarness()
	if err != nil {
		t.Fatalf("new offline real-time harness: %v", err)
	}
	cmd, sandbox := syntheticHelperCommand(t, "measure")
	_, err = harness.MeasureSyntheticProcess(context.Background(), cmd, ProcessMeasurementOptions{SampleInterval: time.Millisecond, SandboxRoot: sandbox, RequireProcessTree: true})
	if !errors.Is(err, ErrUnsupportedHostMetric) || !strings.Contains(err.Error(), "process-tree") {
		t.Fatalf("unsafe process-tree request was not stopped: %v", err)
	}
}

func TestOfflineRealtimeHarnessMeasuresShortLivedLocalProcess(t *testing.T) {
	harness, err := NewOfflineRealtimeHarness()
	if err != nil {
		t.Fatalf("new offline real-time harness: %v", err)
	}
	cmd, sandbox := syntheticHelperCommand(t, "measure")
	measurement, err := harness.MeasureSyntheticProcess(context.Background(), cmd, ProcessMeasurementOptions{SampleInterval: 5 * time.Millisecond, SandboxRoot: sandbox})
	if err != nil {
		t.Fatalf("measure synthetic process: %v", err)
	}
	if measurement.Termination != ProcessCompleted || measurement.ExitCode != 0 {
		t.Fatalf("unexpected process termination: %+v", measurement)
	}
	if measurement.Duration <= 0 || measurement.CPUTime <= 0 || measurement.PeakRSSBytes <= 0 || measurement.PeakMemoryBytes <= 0 {
		t.Fatalf("missing observed time/memory metrics: %+v", measurement)
	}
	if measurement.PeakOpenFDs < 8 || measurement.PeakOpenSockets != 0 || len(measurement.Samples) < 2 {
		t.Fatalf("unexpected observed descriptor/socket metrics: %+v", measurement)
	}
	if measurement.Source != "linux-proc-and-rusage-observed" || measurement.ContentCapture {
		t.Fatalf("unsafe measurement semantics: %+v", measurement)
	}
	modeled := measurement
	modeled.Source = "deterministic-model-not-host-sampled"
	if err := modeled.Validate(); err == nil {
		t.Fatal("modeled resource source was accepted as a real-time measurement")
	}
}

func TestOfflineRealtimeHarnessMeasuresCancellationRelease(t *testing.T) {
	harness, err := NewOfflineRealtimeHarness()
	if err != nil {
		t.Fatalf("new offline real-time harness: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	cmd, sandbox := syntheticHelperCommand(t, "cancel")
	measurement, err := harness.MeasureSyntheticProcess(ctx, cmd, ProcessMeasurementOptions{SampleInterval: 5 * time.Millisecond, SandboxRoot: sandbox})
	if err != nil {
		t.Fatalf("measure cancelled synthetic process: %v", err)
	}
	if measurement.Termination != ProcessCancelled || measurement.CancellationRelease <= 0 || measurement.Duration < 70*time.Millisecond {
		t.Fatalf("unexpected cancellation measurement: %+v", measurement)
	}
	recorded, err := harness.Recorder().Snapshot()
	if err != nil {
		t.Fatalf("recorder snapshot: %v", err)
	}
	if len(recorded.CancellationRelease) != 1 || recorded.CancellationRelease[0] != measurement.CancellationRelease {
		t.Fatalf("cancellation recorder mismatch: %+v vs %+v", recorded, measurement)
	}
}

func TestLinuxProcessSamplerReturnsExplicitUnsupportedMetricError(t *testing.T) {
	root := t.TempDir()
	processRoot := filepath.Join(root, fmt.Sprint(os.Getpid()))
	if err := os.MkdirAll(processRoot, 0o700); err != nil {
		t.Fatalf("create fake proc process: %v", err)
	}
	if err := os.WriteFile(filepath.Join(processRoot, "status"), []byte("Name:\tsynthetic\nState:\tR (running)\nVmRSS:\t1 kB\n"), 0o600); err != nil {
		t.Fatalf("write fake process status: %v", err)
	}
	sampler := &linuxProcessSampler{procRoot: root}
	_, err := sampler.Sample(os.Getpid())
	if err == nil || !errors.Is(err, ErrUnsupportedHostMetric) {
		t.Fatalf("missing VmHWM did not return explicit unsupported metric error: %v", err)
	}
	if !strings.Contains(err.Error(), "rss-and-peak-memory") {
		t.Fatalf("unsupported metric error lacks safe metric identity: %v", err)
	}
}

func TestReadLinuxMemoryTreatsTerminalStatesWithoutMemoryFieldsAsExit(t *testing.T) {
	for _, state := range []string{"Z (zombie)", "X (dead)"} {
		t.Run(state[:1], func(t *testing.T) {
			status := filepath.Join(t.TempDir(), "status")
			contents := []byte("Name:\tsynthetic\nState:\t" + state + "\n")
			if err := os.WriteFile(status, contents, 0o600); err != nil {
				t.Fatalf("write terminal process status: %v", err)
			}
			_, _, err := readLinuxMemory(status)
			if !errors.Is(err, errObservedProcessExited) {
				t.Fatalf("terminal process without memory fields was not classified as exited: %v", err)
			}
		})
	}
}

func TestReadLinuxMemoryRejectsRunningStateWithoutMemoryFields(t *testing.T) {
	status := filepath.Join(t.TempDir(), "status")
	if err := os.WriteFile(status, []byte("Name:\tsynthetic\nState:\tR (running)\n"), 0o600); err != nil {
		t.Fatalf("write running process status: %v", err)
	}
	_, _, err := readLinuxMemory(status)
	if !errors.Is(err, ErrUnsupportedHostMetric) || errors.Is(err, errObservedProcessExited) {
		t.Fatalf("running process without memory fields did not fail closed: %v", err)
	}
}

func TestOfflineRealtimeHarnessFinalizesCompletedWaitAfterMissingMetric(t *testing.T) {
	cmd, _ := syntheticHelperCommand(t, "measure")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start synthetic process: %v", err)
	}
	sampler := &linuxProcessSampler{procRoot: "/proc"}
	var observed HostProcessSample
	deadline := time.Now().Add(time.Second)
	for {
		var err error
		observed, err = sampler.Sample(cmd.Process.Pid)
		if err == nil {
			break
		}
		if (!errors.Is(err, ErrUnsupportedHostMetric) && !errors.Is(err, errObservedProcessExited)) || !time.Now().Before(deadline) {
			t.Fatalf("obtain complete real process sample: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	waitErr := cmd.Wait()
	if cmd.ProcessState == nil {
		t.Fatal("synthetic process has no real process state")
	}

	clock := systemMonotonicClock{}
	harness := &OfflineRealtimeHarness{
		clock:    clock,
		sampler:  sampler,
		recorder: newRealtimeRecorder(clock),
	}
	observed.Elapsed = time.Millisecond
	started := clock.Now().Add(-time.Second)
	result := ProcessMeasurement{
		SchemaVersion:        ProcessMeasurementSchemaVersion,
		Source:               harness.sampler.Source(),
		ExitCode:             -1,
		SampleCadence:        time.Millisecond,
		ContainmentValidated: true,
		ExecutableName:       SyntheticExecutableIdentity,
		ArgvCount:            len(cmd.Args),
		PathPolicy:           syntheticAllowedPath,
		ProcessTreeResidual:  "cgroup-delegation-unavailable; process-tree acceptance is STOP",
		CPUTime:              observed.CPUTime,
		LastRSSBytes:         observed.RSSBytes,
		PeakRSSBytes:         observed.RSSBytes,
		PeakMemoryBytes:      observed.PeakMemoryBytes,
		LastOpenFDs:          observed.OpenFDs,
		PeakOpenFDs:          observed.OpenFDs,
		LastOpenSockets:      observed.OpenSockets,
		PeakOpenSockets:      observed.OpenSockets,
		Samples:              []HostProcessSample{observed},
	}
	waited := make(chan error, 1)
	go func() {
		time.Sleep(time.Millisecond)
		waited <- waitErr
	}()
	status := filepath.Join(t.TempDir(), "status")
	if err := os.WriteFile(status, []byte("Name:\tsynthetic\nState:\tR (running)\n"), 0o600); err != nil {
		t.Fatalf("write missing-memory process status: %v", err)
	}
	_, _, sampleErr := readLinuxMemory(status)
	if !errors.Is(sampleErr, ErrUnsupportedHostMetric) {
		t.Fatalf("missing-memory sample did not fail closed: %v", sampleErr)
	}

	measurement, completed, err := harness.finishAfterCompletedWait(cmd, started, result, false, sampleErr, waited, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("finalize completed process: %v", err)
	}
	if !completed || measurement.Termination != ProcessCompleted || measurement.ExitCode != 0 {
		t.Fatalf("completed wait was not reconciled: completed=%v measurement=%+v", completed, measurement)
	}
	if len(measurement.Samples) != 1 || measurement.PeakMemoryBytes < observed.PeakMemoryBytes {
		t.Fatalf("real sample/rusage reconciliation missing: %+v", measurement)
	}

	empty := result
	empty.Samples = nil
	waited = make(chan error, 1)
	waited <- waitErr
	_, completed, err = harness.finishAfterCompletedWait(cmd, started, empty, false, sampleErr, waited, 100*time.Millisecond)
	if !completed || !errors.Is(err, ErrUnsupportedHostMetric) {
		t.Fatalf("completed wait without a real sample did not fail closed: completed=%v err=%v", completed, err)
	}
}

func TestProcessMeasurementRejectsNonMonotonicDurationWithoutCompareAPI(t *testing.T) {
	measurement := ProcessMeasurement{
		SchemaVersion: ProcessMeasurementSchemaVersion,
		Source:        "linux-proc-and-rusage-observed", ContentCapture: false,
		Termination: ProcessCompleted, Duration: 2 * time.Second, CPUTime: time.Second,
		LastRSSBytes: 1, PeakRSSBytes: 1, PeakMemoryBytes: 1,
		LastOpenFDs: 1, PeakOpenFDs: 1, LastOpenSockets: 0, PeakOpenSockets: 0,
		SampleCadence: time.Millisecond, ContainmentValidated: true,
		ExecutableName: SyntheticExecutableIdentity, ArgvCount: 1, PathPolicy: syntheticAllowedPath,
		ProcessTreeResidual: "cgroup-delegation-unavailable; process-tree acceptance is STOP",
		Samples:             []HostProcessSample{{Elapsed: time.Second, CPUTime: time.Second, RSSBytes: 1, PeakMemoryBytes: 1}, {Elapsed: 500 * time.Millisecond, CPUTime: time.Second, RSSBytes: 1, PeakMemoryBytes: 1}},
	}
	if err := measurement.Validate(); err == nil || !strings.Contains(err.Error(), "host process sample") {
		t.Fatalf("non-monotonic elapsed duration was accepted: %v", err)
	}
}

func TestProcessMeasurementRejectsHostileExecutableIdentity(t *testing.T) {
	measurement := ProcessMeasurement{SchemaVersion: ProcessMeasurementSchemaVersion, Source: "linux-proc-and-rusage-observed", Termination: ProcessCompleted, Duration: time.Second, CPUTime: time.Second, LastRSSBytes: 1, PeakRSSBytes: 1, PeakMemoryBytes: 1, LastOpenFDs: 1, PeakOpenFDs: 1, Samples: []HostProcessSample{{Elapsed: time.Millisecond, CPUTime: time.Millisecond, RSSBytes: 1, PeakMemoryBytes: 1}}, SampleCadence: time.Millisecond, ContainmentValidated: true, ExecutableName: "/tmp/hostile", ArgvCount: 1, PathPolicy: syntheticAllowedPath, ProcessTreeResidual: "cgroup-delegation-unavailable; process-tree acceptance is STOP"}
	if err := measurement.Validate(); err == nil {
		t.Fatal("hostile executable identity was accepted")
	}
}

func TestReadLinuxDescriptorsClassifiesProcSocketLinksWithoutStat(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink("socket:[123]", filepath.Join(root, "3")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/definitely/missing-target", filepath.Join(root, "4")); err != nil {
		t.Fatal(err)
	}
	fds, sockets, err := readLinuxDescriptors(root)
	if err != nil {
		t.Fatalf("read descriptor links: %v", err)
	}
	if fds != 2 || sockets != 1 {
		t.Fatalf("unexpected descriptor classification: fds=%d sockets=%d", fds, sockets)
	}
}

func TestOfflineRealtimeHarnessRejectsInheritedEnvironmentAndContentStreams(t *testing.T) {
	harness, err := NewOfflineRealtimeHarness()
	if err != nil {
		t.Fatalf("new offline real-time harness: %v", err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSyntheticProcessHelper$")
	sandbox := t.TempDir()
	cmd.Dir = sandbox
	if _, err := harness.MeasureSyntheticProcess(context.Background(), cmd, ProcessMeasurementOptions{SampleInterval: time.Millisecond, SandboxRoot: sandbox}); err == nil || !strings.Contains(err.Error(), "explicit environment") {
		t.Fatalf("inherited environment was not rejected: %v", err)
	}
	cmd, sandbox = syntheticHelperCommand(t, "measure")
	cmd.Stdout = os.Stdout
	if _, err := harness.MeasureSyntheticProcess(context.Background(), cmd, ProcessMeasurementOptions{SampleInterval: time.Millisecond, SandboxRoot: sandbox}); err == nil || !strings.Contains(err.Error(), "content streams") {
		t.Fatalf("process content stream was not rejected: %v", err)
	}
	cmd, sandbox = syntheticHelperCommand(t, "measure")
	cmd.Env = append(cmd.Env, "OPENAI_API_KEY=")
	if _, err := harness.MeasureSyntheticProcess(context.Background(), cmd, ProcessMeasurementOptions{SampleInterval: time.Millisecond, SandboxRoot: sandbox}); err == nil || !strings.Contains(err.Error(), "not allowlisted") {
		t.Fatalf("provider environment key was not rejected: %v", err)
	}
}

func syntheticHelperCommand(t *testing.T, mode string) (*exec.Cmd, string) {
	t.Helper()
	sandbox := t.TempDir()
	home := filepath.Join(sandbox, "home")
	work := filepath.Join(sandbox, "work")
	tmp := filepath.Join(sandbox, "tmp")
	for _, directory := range []string{home, work, tmp} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatalf("create synthetic directory: %v", err)
		}
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSyntheticProcessHelper$")
	cmd.Dir = work
	cmd.Env = []string{
		"PATH=" + syntheticAllowedPath,
		syntheticHelperMode + "=" + mode,
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(home, "xdg-config"),
		"XDG_DATA_HOME=" + filepath.Join(home, "xdg-data"),
		"XDG_CACHE_HOME=" + filepath.Join(home, "xdg-cache"),
		"TMPDIR=" + tmp,
	}
	return cmd, sandbox
}
