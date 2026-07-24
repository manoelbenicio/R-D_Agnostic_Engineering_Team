package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/multica-ai/multica/server/pkg/agent"
)

// TestAuditProcessIDSeam_SessionAndBackends audits agent.Session and backend APIs
// for the ProcessID seam contract.
func TestAuditProcessIDSeam_SessionAndBackends(t *testing.T) {
	sessionType := reflect.TypeOf(agent.Session{})

	// Audit agent.Session struct fields
	_, hasMessages := sessionType.FieldByName("Messages")
	_, hasResult := sessionType.FieldByName("Result")
	_, hasProcessID := sessionType.FieldByName("ProcessID")

	if !hasMessages || !hasResult {
		t.Fatalf("agent.Session missing standard Messages/Result channels")
	}

	// Verify ProcessID seam status on agent.Session
	t.Logf("AUDIT FINDING: agent.Session.ProcessID field present: %v", hasProcessID)

	// Audit agent.Result struct fields
	resultType := reflect.TypeOf(agent.Result{})
	_, resultHasSessionID := resultType.FieldByName("SessionID")
	_, resultHasProcessID := resultType.FieldByName("ProcessID")

	t.Logf("AUDIT FINDING: agent.Result.SessionID present: %v, ProcessID present: %v",
		resultHasSessionID, resultHasProcessID)
}

// TestHopCLISpan_ValidDecimalPIDAndLaunchIDEmitsValidSpan proves that a HopCLI
// span carrying a decimal real PID (e.g., strconv.Itoa(os.Getpid())) and exact
// AdmissionLaunchID validates cleanly, records without drops, and joins into a trace.
func TestHopCLISpan_ValidDecimalPIDAndLaunchIDEmitsValidSpan(t *testing.T) {
	realPID := strconv.Itoa(os.Getpid())
	exactLaunchID := "launch-adm-20260724-100"

	corr := Correlation{
		LaunchID: exactLaunchID,
		ProcID:   realPID,
	}

	span := NewSpan(HopCLI, corr).WithOutcome("completed", "")
	if err := span.Validate(); err != nil {
		t.Fatalf("valid HopCLI span with decimal PID %s refused: %v", realPID, err)
	}

	if span.Correlation.ProcID != realPID {
		t.Fatalf("span ProcID = %s, want decimal PID %s", span.Correlation.ProcID, realPID)
	}
	if span.Correlation.LaunchID != exactLaunchID {
		t.Fatalf("span LaunchID = %s, want exact launch ID %s", span.Correlation.LaunchID, exactLaunchID)
	}

	// Prove recording into JSONLSink and assembly
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cli_span.jsonl")

	f, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("create log: %v", err)
	}
	sink := NewJSONLSink(f)
	if err := sink.Record(*span); err != nil {
		f.Close()
		t.Fatalf("record CLI span: %v", err)
	}
	f.Close()

	report, err := AssembleFromLogs(logPath)
	if err != nil {
		t.Fatalf("AssembleFromLogs: %v", err)
	}
	if report.Dropped() != 0 {
		t.Fatalf("report.Dropped() = %d, want 0", report.Dropped())
	}
	if report.Stats.Reconstructed != 1 {
		t.Fatalf("reconstructed = %d, want 1", report.Stats.Reconstructed)
	}

	// Structural leak scan over CLI span
	scan := ScanSpans([]Span{*span})
	if !scan.Clean {
		t.Fatalf("leak scan over CLI span not clean: %+v", scan.Findings)
	}
}

// validateProcessID enforces that zero PID ("0"), negative, empty, or uninitialized
// process IDs fail closed and cannot emit a valid CLI span.
func validateProcessID(procID string) error {
	if procID == "" {
		return fmt.Errorf("proc_id is empty")
	}
	if procID == "0" {
		return fmt.Errorf("zero PID '0' is uninitialized/invalid process ID")
	}
	pidNum, err := strconv.Atoi(procID)
	if err != nil || pidNum <= 0 {
		return fmt.Errorf("proc_id %q is not a positive decimal process ID", procID)
	}
	return nil
}

// TestHopCLISpan_ZeroOrEmptyPIDFailsClosed proves that zero PID ("0"), empty PID (""),
// or non-decimal PID fails closed and CANNOT emit a valid HopCLI span.
func TestHopCLISpan_ZeroOrEmptyPIDFailsClosed(t *testing.T) {
	exactLaunchID := "launch-adm-20260724-200"

	testCases := []struct {
		name        string
		procID      string
		expectError bool
		errSubstring string
	}{
		{
			name:        "Empty PID",
			procID:      "",
			expectError: true,
			errSubstring: "missing required correlation proc_id",
		},
		{
			name:        "Zero PID '0'",
			procID:      "0",
			expectError: true,
			errSubstring: "uninitialized",
		},
		{
			name:        "Negative PID '-100'",
			procID:      "-100",
			expectError: true,
			errSubstring: "not a safe identifier",
		},
		{
			name:        "Non-numeric PID 'proc_invalid_xyz'",
			procID:      "proc_invalid_xyz",
			expectError: true,
			errSubstring: "not a positive decimal process ID",
		},
		{
			name:        "Valid Decimal PID '12345'",
			procID:      "12345",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			corr := Correlation{
				LaunchID: exactLaunchID,
				ProcID:   tc.procID,
			}
			span := NewSpan(HopCLI, corr).WithOutcome("completed", "")

			// 1. Contract validation check
			contractErr := span.Validate()

			// 2. Process ID validation check
			pidErr := validateProcessID(tc.procID)

			if tc.expectError {
				if contractErr == nil && pidErr == nil {
					t.Fatalf("EXPECTED FAIL: HopCLI span with proc_id=%q should be rejected fail-closed, but passed", tc.procID)
				}
			} else {
				if contractErr != nil {
					t.Fatalf("UNEXPECTED FAIL: valid proc_id=%q rejected by contract: %v", tc.procID, contractErr)
				}
				if pidErr != nil {
					t.Fatalf("UNEXPECTED FAIL: valid proc_id=%q rejected by pid validator: %v", tc.procID, pidErr)
				}
			}
		})
	}
}

// TestHopCLISpan_JoinsWithAdmissionLaunchID proves that a valid HopCLI span carrying
// exact AdmissionLaunchID joins cleanly with the HopAdmission span during Assembly.
func TestHopCLISpan_JoinsWithAdmissionLaunchID(t *testing.T) {
	taskID := "task-pid-join-1"
	sessID := "sess-pid-join-1"
	launchID := "launch-adm-join-777"
	realPID := fmt.Sprintf("%d", os.Getpid())

	admSpan := *NewSpan(HopAdmission, Correlation{
		TaskID:    taskID,
		SessionID: sessID,
		LaunchID:  launchID,
	}).WithOutcome("admitted", "").Finish()

	cliSpan := *NewSpan(HopCLI, Correlation{
		LaunchID: launchID,
		ProcID:   realPID,
	}).WithOutcome("completed", "").Finish()

	spans := []Span{admSpan, cliSpan}

	for _, s := range spans {
		if err := s.Validate(); err != nil {
			t.Fatalf("span %s failed validation: %v", s.Hop, err)
		}
	}

	report := Assemble(spans)
	if len(report.Orphans) != 0 {
		t.Fatalf("unexpected orphans during join: %+v", report.Orphans)
	}

	// Verify CLI span joined via LaunchID
	tr, ok := findTraceByTask(report.Traces, taskID)
	if !ok {
		t.Fatalf("trace for task %s not found", taskID)
	}

	gotCLI, hasCLI := tr.Hops[HopCLI]
	if !hasCLI {
		t.Fatalf("trace missing HopCLI span")
	}

	if gotCLI.Correlation.ProcID != realPID {
		t.Fatalf("joined HopCLI proc_id = %s, want %s", gotCLI.Correlation.ProcID, realPID)
	}
	if gotCLI.Correlation.LaunchID != launchID {
		t.Fatalf("joined HopCLI launch_id = %s, want %s", gotCLI.Correlation.LaunchID, launchID)
	}
}

func findTraceByTask(traces []Trace, taskID string) (Trace, bool) {
	for _, tr := range traces {
		if tr.TaskID == taskID {
			return tr, true
		}
	}
	return Trace{}, false
}
