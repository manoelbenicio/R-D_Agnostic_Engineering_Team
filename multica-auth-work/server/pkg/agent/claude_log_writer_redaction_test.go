package agent

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// These tests exercise logWriter.Write in isolation with only synthetic,
// obviously-fake sentinel values (never real credentials). They verify the
// credential-isolation 5.4 hardening: raw subprocess stderr routed into
// logWriter must not surface a recognizable secret shape in the captured
// log output, while remaining otherwise useful and preserving the
// io.Writer byte-count contract.

func newCapturingLogWriter(buf *bytes.Buffer, prefix string) *logWriter {
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return newLogWriter(logger, prefix)
}

func TestLogWriterRedactsAPIKeySentinel(t *testing.T) {
	var buf bytes.Buffer
	w := newCapturingLogWriter(&buf, "[claude:stderr] ")

	const sentinel = "sk-proj-SYNTHETIC0000000000000000000000000000"
	input := []byte("fatal: request failed: OPENAI_API_KEY=" + sentinel)

	n, err := w.Write(input)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if n != len(input) {
		t.Fatalf("Write returned n=%d, want %d (len(p))", n, len(input))
	}

	got := buf.String()
	if strings.Contains(got, sentinel) {
		t.Fatalf("synthetic API key sentinel leaked into captured log output: %q", got)
	}
	if !strings.Contains(got, "[REDACTED") {
		t.Fatalf("expected a redaction placeholder in captured output, got: %q", got)
	}
}

func TestLogWriterRedactsBearerTokenSentinel(t *testing.T) {
	var buf bytes.Buffer
	w := newCapturingLogWriter(&buf, "[claude:stderr] ")

	const sentinel = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJTWU5USEVUSUMifQ.abc123synthetic"
	input := []byte("Authorization: Bearer " + sentinel)

	if _, err := w.Write(input); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, sentinel) {
		t.Fatalf("synthetic bearer token sentinel leaked into captured log output: %q", got)
	}
}

func TestLogWriterRedactsErrorBodyTokenField(t *testing.T) {
	var buf bytes.Buffer
	w := newCapturingLogWriter(&buf, "[claude:stderr] ")

	const sentinel = "synthetic-error-body-token-sentinel"
	input := []byte(`{"error":"invalid_grant","access_token":"` + sentinel + `"}`)

	if _, err := w.Write(input); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, sentinel) {
		t.Fatalf("synthetic error-body token field leaked into captured log output: %q", got)
	}
}

func TestLogWriterPreservesSafeStderrContent(t *testing.T) {
	var buf bytes.Buffer
	w := newCapturingLogWriter(&buf, "[claude:stderr] ")

	input := []byte("warning: deprecated flag --legacy-mode, use --mode=v2 instead")

	if _, err := w.Write(input); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "deprecated flag") {
		t.Fatalf("safe, non-sensitive stderr content was altered/dropped: %q", got)
	}
	if !strings.Contains(got, "[claude:stderr] ") {
		t.Fatalf("expected prefix to be preserved in captured output: %q", got)
	}
}

func TestLogWriterEmptyOrWhitespaceEmitsNothing(t *testing.T) {
	cases := []string{"", "   ", "\n\n\t \n", "\r\n"}
	for _, tc := range cases {
		var buf bytes.Buffer
		w := newCapturingLogWriter(&buf, "[claude:stderr] ")

		n, err := w.Write([]byte(tc))
		if err != nil {
			t.Fatalf("Write(%q) returned error: %v", tc, err)
		}
		if n != len(tc) {
			t.Fatalf("Write(%q) returned n=%d, want %d", tc, n, len(tc))
		}
		if buf.Len() != 0 {
			t.Fatalf("Write(%q) emitted log output for whitespace-only input: %q", tc, buf.String())
		}
	}
}

func TestLogWriterReturnedByteCountMatchesInputRegardlessOfRedaction(t *testing.T) {
	var buf bytes.Buffer
	w := newCapturingLogWriter(&buf, "[claude:stderr] ")

	// A long line whose redacted form is drastically shorter than the raw
	// input; the returned byte count must still reflect bytes consumed from
	// p, per the io.Writer contract, not the length of the redacted text
	// that ends up in the log.
	const sentinel = "PASSWORD=hunter2-synthetic-not-real-0000000000000000000000000000"
	input := []byte(sentinel)

	n, err := w.Write(input)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if n != len(input) {
		t.Fatalf("Write returned n=%d, want %d (len(p)); io.Writer byte-count contract violated", n, len(input))
	}
}
