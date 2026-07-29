package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestReadPasswordInputPreservesWhitespaceAndTrimsOnlyLineEnding(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{name: "lf", in: "  synthetic value  \n", want: "  synthetic value  "},
		{name: "crlf", in: "\tsynthetic value\t\r\n", want: "\tsynthetic value\t"},
		{name: "no ending", in: " synthetic value ", want: " synthetic value "},
		{name: "embedded newline", in: "synthetic\nvalue\n", want: "synthetic\nvalue"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readPasswordInput(strings.NewReader(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("password bytes = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadPasswordInputRejectsEmptyAndOversized(t *testing.T) {
	for _, input := range []string{"", "\n", "short-value", strings.Repeat("x", maxUserPasswordBytes+1)} {
		if _, err := readPasswordInput(strings.NewReader(input)); err == nil {
			t.Fatalf("input length %d was accepted", len(input))
		}
	}
}

func TestReadPasswordLineDrainsAnOverlongTerminalLine(t *testing.T) {
	input := strings.NewReader(strings.Repeat("x", maxUserPasswordBytes+20) + "\nwould-be-shell-input")
	if _, err := readPasswordLine(input); err == nil {
		t.Fatal("overlong terminal password was accepted")
	}
	if input.Len() != 0 {
		t.Fatalf("terminal input was left queued: %d bytes", input.Len())
	}
}

func TestUserPasswordUpdateRejectsArgumentAndPasswordFlag(t *testing.T) {
	cmd := newUserPasswordUpdateCmd()
	if err := cmd.Args(cmd, []string{"synthetic"}); err == nil {
		t.Fatal("positional password was accepted")
	}
	if err := cmd.Flags().Set("password", "synthetic"); err == nil {
		t.Fatal("password value flag was accepted")
	}
}

func TestRunUserPasswordUpdateReadsExplicitStdinAndCallsDedicatedEndpoint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "synthetic-token")

	var requestPassword string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/me/password" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer synthetic-token" {
			t.Fatal("missing authenticated CLI request")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requestPassword = body["new_password"]
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("MULTICA_SERVER_URL", srv.URL)

	cmd := newUserPasswordUpdateCmd()
	cmd.SetIn(strings.NewReader("  synthetic value  \r\n"))
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.Flags().Set("password-stdin", "true"); err != nil {
		t.Fatal(err)
	}
	if err := runUserPasswordUpdate(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if requestPassword != "  synthetic value  " {
		t.Fatal("password whitespace was not preserved in request")
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "synthetic value") {
		t.Fatal("CLI output echoed the password")
	}
	if !strings.Contains(stdout.String(), "Password updated") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestPasswordPromptFailsClosedWhenInputIsNotTerminal(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	var prompt bytes.Buffer
	if _, err := readPasswordFromTerminal(reader, &prompt); err == nil || !strings.Contains(err.Error(), "--password-stdin") {
		t.Fatalf("non-terminal prompt error = %v", err)
	}
	if prompt.Len() != 0 {
		t.Fatalf("prompt was printed before terminal validation: %q", prompt.String())
	}
}
