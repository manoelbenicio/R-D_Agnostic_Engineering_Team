package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newRuntimeManagerTestCmd(use, serverURL string) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	cmd.Flags().String("server-url", "", "")
	cmd.Flags().String("workspace-id", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.Flags().String("token", "", "")
	cmd.Flags().String("output", "table", "")
	_ = cmd.Flags().Set("server-url", serverURL)
	_ = cmd.Flags().Set("workspace-id", "ws-1")
	return cmd
}

func TestRuntimeStandardCommandsRegistered(t *testing.T) {
	for _, path := range [][]string{
		{"list"}, {"get"}, {"create-version"}, {"diff"}, {"validate"}, {"activate"}, {"rollback"},
		{"session", "list"}, {"session", "create"}, {"session", "enroll"},
		{"binding", "list"}, {"binding", "configure"}, {"binding", "diff"}, {"binding", "validate"}, {"binding", "activate"}, {"binding", "rollback"},
		{"home", "list"}, {"home", "attach"},
	} {
		cmd, _, err := runtimeStandardCmd.Find(path)
		if err != nil || cmd == nil || cmd.Name() != path[len(path)-1] {
			t.Fatalf("runtime-standard %s not registered: %v / %#v", strings.Join(path, " "), err, cmd)
		}
	}
}

func TestRunRuntimeSessionCreateUsesFrozenContractAndIdempotency(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")
	var body map[string]any
	var idempotency string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/runtime-sessions" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		idempotency = r.Header.Get("Idempotency-Key")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "session-1", "state": "ready"})
	}))
	defer srv.Close()

	cmd := newRuntimeManagerTestCmd("create", srv.URL)
	cmd.Flags().String("name", "", "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("runtime-kind", "", "")
	cmd.Flags().String("standard-id", "", "")
	_ = cmd.Flags().Set("name", "Reusable")
	_ = cmd.Flags().Set("provider", "anthropic")
	_ = cmd.Flags().Set("runtime-kind", "claude")
	_ = cmd.Flags().Set("standard-id", "standard-1")
	if err := runRuntimeSessionCreate(cmd, nil); err != nil {
		t.Fatalf("runRuntimeSessionCreate: %v", err)
	}
	if idempotency == "" {
		t.Fatal("missing Idempotency-Key")
	}
	if body["name"] != "Reusable" || body["provider"] != "anthropic" || body["runtime_kind"] != "claude" || body["standard_id"] != "standard-1" {
		t.Fatalf("body = %#v", body)
	}
	for _, forbidden := range []string{"account", "path", "credential"} {
		if strings.Contains(strings.ToLower(string(mustJSON(t, body))), forbidden) {
			t.Fatalf("body exposes %q: %#v", forbidden, body)
		}
	}
}

func TestRunRuntimeBindingConfigureClonesAndChangesOnlyModelReasoning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")
	active := map[string]any{
		"schema_version": "v1",
		"values": map[string]any{
			"transport_binding": "native_credential_home",
			"cli_kind":          "claude",
			"provider":          "anthropic",
			"model":             "sonnet",
			"reasoning_effort":  "medium",
			"limits":            map[string]any{"max_output_tokens": float64(4096)},
		},
		"delegability": map[string]any{"model": false},
	}
	var posted map[string]any
	var postKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/workspaces/ws-1/runtime-bindings/binding-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                   "binding-1",
				"active_configuration": active,
				"capabilities":         map[string]any{"models": map[string]any{"opus": map[string]any{"reasoning_efforts": []string{"high"}}}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/workspaces/ws-1/runtime-bindings/binding-1/configuration-versions":
			postKey = r.Header.Get("Idempotency-Key")
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "version-2", "state": "inactive", "apply_class": "restart"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	cmd := newRuntimeManagerTestCmd("configure", srv.URL)
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("reasoning-effort", "", "")
	cmd.Flags().String("reason", "", "")
	_ = cmd.Flags().Set("model", "opus")
	_ = cmd.Flags().Set("reasoning-effort", "high")
	_ = cmd.Flags().Set("reason", "approved capability")
	if err := runRuntimeBindingConfigure(cmd, []string{"binding-1"}); err != nil {
		t.Fatalf("runRuntimeBindingConfigure: %v", err)
	}
	if postKey == "" {
		t.Fatal("missing Idempotency-Key")
	}

	configuration := posted["configuration"].(map[string]any)
	values := configuration["values"].(map[string]any)
	if values["model"] != "opus" || values["reasoning_effort"] != "high" {
		t.Fatalf("values = %#v", values)
	}
	if values["transport_binding"] != "native_credential_home" || values["cli_kind"] != "claude" || values["provider"] != "anthropic" {
		t.Fatalf("non-model configuration changed: %#v", values)
	}
	if values["limits"].(map[string]any)["max_output_tokens"] != float64(4096) {
		t.Fatalf("limits changed: %#v", values["limits"])
	}
	if configuration["delegability"].(map[string]any)["model"] != false {
		t.Fatalf("delegability changed: %#v", configuration["delegability"])
	}
}

func TestRunRuntimeBindingActivateSendsCASBody(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/workspaces/ws-1/runtime-bindings/binding-1/configuration-versions/version-2/activate" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Fatal("missing Idempotency-Key")
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"active_version_id": "version-2"})
	}))
	defer srv.Close()

	cmd := newRuntimeManagerTestCmd("activate", srv.URL)
	cmd.Flags().String("expected-active-version-id", "", "")
	cmd.Flags().Int64("expected-binding-generation", -1, "")
	cmd.Flags().String("reason", "", "")
	_ = cmd.Flags().Set("expected-active-version-id", "version-1")
	_ = cmd.Flags().Set("expected-binding-generation", "7")
	_ = cmd.Flags().Set("reason", "validated")
	if err := runRuntimeBindingActivate(cmd, []string{"binding-1", "version-2"}); err != nil {
		t.Fatalf("runRuntimeBindingActivate: %v", err)
	}
	if body["expected_active_version_id"] != "version-1" || body["expected_binding_generation"] != float64(7) || body["reason"] != "validated" {
		t.Fatalf("CAS body = %#v", body)
	}
}

func TestRunRuntimeHomeAttachUsesOnlyOpaqueReferenceAndGenerations(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/ws-1/runtime-bindings/binding-1/home-assignments" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "binding-1"})
	}))
	defer srv.Close()
	cmd := newRuntimeManagerTestCmd("attach", srv.URL)
	cmd.Flags().String("home-ref", "", "")
	cmd.Flags().Int64("expected-binding-generation", -1, "")
	cmd.Flags().Int64("expected-catalog-generation", -1, "")
	_ = cmd.Flags().Set("home-ref", "home_opaque")
	_ = cmd.Flags().Set("expected-binding-generation", "3")
	_ = cmd.Flags().Set("expected-catalog-generation", "9")
	if err := runRuntimeHomeAttach(cmd, []string{"binding-1"}); err != nil {
		t.Fatalf("runRuntimeHomeAttach: %v", err)
	}
	if len(body) != 3 || body["home_ref"] != "home_opaque" || body["expected_binding_generation"] != float64(3) || body["expected_catalog_generation"] != float64(9) {
		t.Fatalf("body = %#v", body)
	}
}

func TestRuntimeManagerTableAndDiffNeverRenderUnknownSensitiveFields(t *testing.T) {
	before := map[string]any{"values": map[string]any{"model": "sonnet", "path": "/home/private", "account_email": "person@example.test", "secret": "token"}}
	after := map[string]any{"values": map[string]any{"model": "opus", "path": "/other/private", "account_email": "other@example.test", "secret": "new-token"}}
	encoded := string(mustJSON(t, rmSafeDiff(before, after)))
	if !strings.Contains(encoded, "model") {
		t.Fatalf("safe diff missing model: %s", encoded)
	}
	for _, forbidden := range []string{"/home/private", "/other/private", "example.test", "token", "path", "account_email", "secret"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("safe diff exposed %q: %s", forbidden, encoded)
		}
	}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": "binding-1", "runtime_id": "runtime-1", "provider": "anthropic", "state": "ready", "raw_path": "/home/private", "account": "person@example.test"}}, "next_cursor": nil})
	}))
	defer srv.Close()
	cmd := newRuntimeManagerTestCmd("list", srv.URL)
	out, err := captureRuntimeStdout(t, func() error { return runRuntimeBindingList(cmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "/home/private") || strings.Contains(out, "person@example.test") || strings.Contains(out, "raw_path") || strings.Contains(out, "account") {
		t.Fatalf("table leaked forbidden data: %s", out)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
