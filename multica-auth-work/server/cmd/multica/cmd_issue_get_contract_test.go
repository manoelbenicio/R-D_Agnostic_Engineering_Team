package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
	"github.com/spf13/cobra"
)

const (
	contractIssueID      = "1881a167-4bb6-4602-944b-f40ce4192fe6"
	contractOtherIssueID = "2881a167-4bb6-4602-944b-f40ce4192fe6"
)

func newIssueGetContractCommand(t *testing.T, serverURL, workspaceID string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "get"}
	cmd.Flags().String("server-url", "", "")
	cmd.Flags().String("workspace-id", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flags().String("output", "json", "")
	if err := cmd.Flags().Set("server-url", serverURL); err != nil {
		t.Fatal(err)
	}
	if workspaceID != "" {
		if err := cmd.Flags().Set("workspace-id", workspaceID); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

func contractIssueBody(id string) string {
	body, _ := json.Marshal(map[string]any{
		"id":         id,
		"identifier": "ORQ-38",
		"number":     38,
		"title":      "Fail-closed contract",
	})
	return string(body)
}

type contractHTTPResponse struct {
	status int
	body   string
}

func runIssueGetContractScenario(t *testing.T, workspaceID string, responses ...contractHTTPResponse) (string, error, int, []string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")
	t.Setenv("MULTICA_WORKSPACE_ID", "")
	t.Setenv("MULTICA_AGENT_ID", "")
	t.Setenv("MULTICA_TASK_ID", "")

	calls := 0
	var workspaces []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaces = append(workspaces, r.Header.Get("X-Workspace-ID"))
		if calls >= len(responses) {
			t.Fatalf("unexpected request %d to %s", calls+1, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/issues/"+contractIssueID {
			t.Fatalf("path = %q, want issue path", r.URL.Path)
		}
		response := responses[calls]
		calls++
		if response.status != 0 {
			w.WriteHeader(response.status)
		}
		if response.body != "" {
			_, _ = io.WriteString(w, response.body)
		}
	}))
	defer srv.Close()

	cmd := newIssueGetContractCommand(t, srv.URL, workspaceID)
	out, err := captureStdout(t, func() error {
		return runIssueGet(cmd, []string{contractIssueID})
	})
	return out, err, calls, workspaces
}

func TestRequireIssueIdentity(t *testing.T) {
	valid := map[string]any{
		"id":         contractIssueID,
		"identifier": "ORQ-38",
		"number":     float64(38),
	}
	identity, err := requireIssueIdentity(valid)
	if err != nil {
		t.Fatalf("valid identity rejected: %v", err)
	}
	if identity.ID != contractIssueID || identity.Identifier != "ORQ-38" || identity.Number != 38 {
		t.Fatalf("identity = %#v", identity)
	}

	cases := map[string]map[string]any{
		"missing id":         {"identifier": "ORQ-38", "number": float64(38)},
		"null id":            {"id": nil, "identifier": "ORQ-38", "number": float64(38)},
		"empty id":           {"id": "  ", "identifier": "ORQ-38", "number": float64(38)},
		"non UUID id":        {"id": "issue-38", "identifier": "ORQ-38", "number": float64(38)},
		"numeric id":         {"id": float64(1), "identifier": "ORQ-38", "number": float64(38)},
		"missing identifier": {"id": contractIssueID, "number": float64(38)},
		"null identifier":    {"id": contractIssueID, "identifier": nil, "number": float64(38)},
		"empty identifier":   {"id": contractIssueID, "identifier": "  ", "number": float64(38)},
		"numeric identifier": {"id": contractIssueID, "identifier": float64(38), "number": float64(38)},
		"missing number":     {"id": contractIssueID, "identifier": "ORQ-38"},
		"null number":        {"id": contractIssueID, "identifier": "ORQ-38", "number": nil},
		"string number":      {"id": contractIssueID, "identifier": "ORQ-38", "number": "38"},
		"boolean number":     {"id": contractIssueID, "identifier": "ORQ-38", "number": true},
		"integer Go type":    {"id": contractIssueID, "identifier": "ORQ-38", "number": 38},
		"zero number":        {"id": contractIssueID, "identifier": "ORQ-38", "number": float64(0)},
		"negative number":    {"id": contractIssueID, "identifier": "ORQ-38", "number": float64(-1)},
		"fraction number":    {"id": contractIssueID, "identifier": "ORQ-38", "number": 38.5},
		"NaN number":         {"id": contractIssueID, "identifier": "ORQ-38", "number": math.NaN()},
		"infinite number":    {"id": contractIssueID, "identifier": "ORQ-38", "number": math.Inf(1)},
		"overflow number":    {"id": contractIssueID, "identifier": "ORQ-38", "number": float64(1 << 31)},
	}
	for name, issue := range cases {
		name, issue := name, issue
		t.Run(name, func(t *testing.T) {
			if _, err := requireIssueIdentity(issue); err == nil {
				t.Fatalf("invalid identity accepted: %#v", issue)
			}
		})
	}
}

func TestRunIssueGetFailClosed(t *testing.T) {
	t.Run("requires workspace before request", func(t *testing.T) {
		out, err, calls, _ := runIssueGetContractScenario(t, "")
		if err == nil || !strings.Contains(err.Error(), "workspace_id is required") {
			t.Fatalf("error = %v, want workspace requirement", err)
		}
		if out != "" || calls != 0 {
			t.Fatalf("out = %q calls = %d, want empty/zero", out, calls)
		}
	})

	t.Run("validates both reads and propagates workspace", func(t *testing.T) {
		body := contractIssueBody(contractIssueID)
		out, err, calls, workspaces := runIssueGetContractScenario(t, "ws-contract",
			contractHTTPResponse{status: http.StatusOK, body: body},
			contractHTTPResponse{status: http.StatusOK, body: body},
		)
		if err != nil {
			t.Fatalf("runIssueGet: %v", err)
		}
		if calls != 2 || len(workspaces) != 2 || workspaces[0] != "ws-contract" || workspaces[1] != "ws-contract" {
			t.Fatalf("calls/workspaces = %d/%v", calls, workspaces)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("stdout is not JSON: %q: %v", out, err)
		}
		if got["id"] != contractIssueID || got["identifier"] != "ORQ-38" || got["number"] != float64(38) {
			t.Fatalf("stdout identity = %#v", got)
		}
	})

	for _, status := range []int{http.StatusFound, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		status := status
		t.Run("rejects second "+http.StatusText(status), func(t *testing.T) {
			out, err, calls, _ := runIssueGetContractScenario(t, "ws-contract",
				contractHTTPResponse{status: http.StatusOK, body: contractIssueBody(contractIssueID)},
				contractHTTPResponse{status: status, body: `{"title":"must-not-leak"}`},
			)
			var statusErr *cli.UnexpectedStatusError
			if !errors.As(err, &statusErr) || statusErr.Actual != status {
				t.Fatalf("error = %T %v, want unexpected status %d", err, err, status)
			}
			if out != "" || calls != 2 || strings.Contains(err.Error(), "must-not-leak") {
				t.Fatalf("out=%q calls=%d err=%v", out, calls, err)
			}
		})
	}

	for _, status := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		status := status
		t.Run("preserves HTTP error "+http.StatusText(status), func(t *testing.T) {
			out, err, calls, _ := runIssueGetContractScenario(t, "ws-contract",
				contractHTTPResponse{status: status, body: `{"error":"fail closed"}`},
			)
			var httpErr *cli.HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != status {
				t.Fatalf("error = %T %v, want HTTP %d", err, err, status)
			}
			if out != "" || calls != 1 {
				t.Fatalf("out=%q calls=%d", out, calls)
			}
		})
	}

	for name, body := range map[string]string{
		"invalid JSON":       `{`,
		"array":              `[]`,
		"multiple values":    contractIssueBody(contractIssueID) + "\n" + contractIssueBody(contractIssueID),
		"missing identifier": `{"id":"` + contractIssueID + `","number":38}`,
	} {
		name, body := name, body
		t.Run("fails fast on "+name, func(t *testing.T) {
			out, err, calls, _ := runIssueGetContractScenario(t, "ws-contract",
				contractHTTPResponse{status: http.StatusOK, body: body},
				contractHTTPResponse{status: http.StatusOK, body: contractIssueBody(contractIssueID)},
			)
			if err == nil || out != "" || calls != 1 {
				t.Fatalf("err=%v out=%q calls=%d", err, out, calls)
			}
		})
	}

	t.Run("rejects missing identity on final read", func(t *testing.T) {
		out, err, calls, _ := runIssueGetContractScenario(t, "ws-contract",
			contractHTTPResponse{status: http.StatusOK, body: contractIssueBody(contractIssueID)},
			contractHTTPResponse{status: http.StatusOK, body: `{"id":"` + contractIssueID + `","identifier":"ORQ-38"}`},
		)
		if err == nil || !strings.Contains(err.Error(), "missing number") || out != "" || calls != 2 {
			t.Fatalf("err=%v out=%q calls=%d", err, out, calls)
		}
	})

	t.Run("rejects identity change between reads", func(t *testing.T) {
		out, err, calls, _ := runIssueGetContractScenario(t, "ws-contract",
			contractHTTPResponse{status: http.StatusOK, body: contractIssueBody(contractIssueID)},
			contractHTTPResponse{status: http.StatusOK, body: contractIssueBody(contractOtherIssueID)},
		)
		if err == nil || !strings.Contains(err.Error(), "does not match resolved id") || out != "" || calls != 2 {
			t.Fatalf("err=%v out=%q calls=%d", err, out, calls)
		}
	})
}

func TestResolveIssueRefStrictPrefixRequiresExactListStatus(t *testing.T) {
	for _, status := range []int{http.StatusCreated, http.StatusFound} {
		status := status
		t.Run(http.StatusText(status), func(t *testing.T) {
			destinationCalls := 0
			destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				destinationCalls++
				_, _ = io.WriteString(w, `{"issues":[]}`)
			}))
			defer destination.Close()
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Location", destination.URL)
				w.WriteHeader(status)
			}))
			defer source.Close()

			client := cli.NewAPIClient(source.URL, "ws-contract", "")
			_, err := resolveIssueRefStrict(context.Background(), client, "abcd")
			var statusErr *cli.UnexpectedStatusError
			if !errors.As(err, &statusErr) || statusErr.Actual != status {
				t.Fatalf("error = %T %v, want exact status %d", err, err, status)
			}
			if destinationCalls != 0 {
				t.Fatalf("redirect destination calls=%d, want 0", destinationCalls)
			}
		})
	}
}
