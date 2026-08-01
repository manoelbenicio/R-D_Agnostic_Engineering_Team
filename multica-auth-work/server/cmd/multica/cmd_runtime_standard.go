package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

// runtime-standard is the pathless Runtime Manager CLI. Table output is an
// explicit allowlist and never prints account, filesystem, credential, argv,
// environment, or provider-response data. JSON output is limited to the
// server's frozen redacted Runtime Manager contracts.
var runtimeStandardCmd = &cobra.Command{
	Use:   "runtime-standard",
	Short: "Manage reusable runtime sessions and versioned runtime configuration",
}

var (
	runtimeStandardListCmd          = rmCommand("list", "List runtime standards", 0, runRuntimeStandardList)
	runtimeStandardGetCmd           = rmCommand("get <standard-id>", "Get a runtime standard", 1, runRuntimeStandardGet)
	runtimeStandardCreateVersionCmd = rmCommand("create-version <standard-id>", "Create an inactive immutable standard version", 1, runRuntimeStandardCreateVersion)
	runtimeStandardDiffCmd          = rmCommand("diff <standard-id> <version-id>", "Show a safe configuration diff", 2, runRuntimeStandardDiff)
	runtimeStandardValidateCmd      = rmCommand("validate <standard-id> <version-id>", "Validate a standard version", 2, runRuntimeStandardValidate)
	runtimeStandardActivateCmd      = rmCommand("activate <standard-id> <version-id>", "Activate a standard version with compare-and-swap", 2, runRuntimeStandardActivate)
	runtimeStandardRollbackCmd      = rmCommand("rollback <standard-id>", "Activate a prior immutable standard version", 1, runRuntimeStandardRollback)

	runtimeSessionCmd       = &cobra.Command{Use: "session", Short: "Manage reusable owner-global runtime sessions"}
	runtimeSessionListCmd   = rmCommand("list", "List reusable sessions", 0, runRuntimeSessionList)
	runtimeSessionCreateCmd = rmCommand("create", "Create an accountless reusable session", 0, runRuntimeSessionCreate)
	runtimeSessionEnrollCmd = rmCommand("enroll <session-id>", "Enroll an existing runtime and agent", 1, runRuntimeSessionEnroll)

	runtimeBindingCmd          = &cobra.Command{Use: "binding", Short: "Manage workspace runtime bindings"}
	runtimeBindingListCmd      = rmCommand("list", "List runtime bindings", 0, runRuntimeBindingList)
	runtimeBindingConfigureCmd = rmCommand("configure <binding-id>", "Create a model/reasoning-only runtime version", 1, runRuntimeBindingConfigure)
	runtimeBindingDiffCmd      = rmCommand("diff <binding-id> <version-id>", "Show a safe runtime configuration diff", 2, runRuntimeBindingDiff)
	runtimeBindingValidateCmd  = rmCommand("validate <binding-id> <version-id>", "Validate a runtime configuration version", 2, runRuntimeBindingValidate)
	runtimeBindingActivateCmd  = rmCommand("activate <binding-id> <version-id>", "Activate a runtime version with compare-and-swap", 2, runRuntimeBindingActivate)
	runtimeBindingRollbackCmd  = rmCommand("rollback <binding-id>", "Activate a prior immutable runtime version", 1, runRuntimeBindingRollback)

	runtimeHomeCmd       = &cobra.Command{Use: "home", Short: "Manage opaque subscription attachments"}
	runtimeHomeListCmd   = rmCommand("list", "List opaque credential-home projections", 0, runRuntimeHomeList)
	runtimeHomeAttachCmd = rmCommand("attach <binding-id>", "Attach an opaque home reference", 1, runRuntimeHomeAttach)
)

func rmCommand(use, short string, args int, run func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: exactArgs(args), RunE: run}
}

func init() {
	runtimeStandardCmd.AddCommand(runtimeStandardListCmd, runtimeStandardGetCmd, runtimeStandardCreateVersionCmd, runtimeStandardDiffCmd, runtimeStandardValidateCmd, runtimeStandardActivateCmd, runtimeStandardRollbackCmd)
	runtimeStandardCmd.AddCommand(runtimeSessionCmd, runtimeBindingCmd, runtimeHomeCmd)
	runtimeSessionCmd.AddCommand(runtimeSessionListCmd, runtimeSessionCreateCmd, runtimeSessionEnrollCmd)
	runtimeBindingCmd.AddCommand(runtimeBindingListCmd, runtimeBindingConfigureCmd, runtimeBindingDiffCmd, runtimeBindingValidateCmd, runtimeBindingActivateCmd, runtimeBindingRollbackCmd)
	runtimeHomeCmd.AddCommand(runtimeHomeListCmd, runtimeHomeAttachCmd)

	for _, cmd := range []*cobra.Command{
		runtimeStandardListCmd, runtimeStandardGetCmd, runtimeStandardCreateVersionCmd, runtimeStandardDiffCmd, runtimeStandardValidateCmd, runtimeStandardActivateCmd, runtimeStandardRollbackCmd,
		runtimeSessionListCmd, runtimeSessionCreateCmd, runtimeSessionEnrollCmd,
		runtimeBindingListCmd, runtimeBindingConfigureCmd, runtimeBindingDiffCmd, runtimeBindingValidateCmd, runtimeBindingActivateCmd, runtimeBindingRollbackCmd,
		runtimeHomeListCmd, runtimeHomeAttachCmd,
	} {
		cmd.Flags().String("output", "table", "Output format: table or json")
	}

	runtimeStandardCreateVersionCmd.Flags().String("configuration", "", "Runtime configuration document as JSON (required)")
	runtimeStandardCreateVersionCmd.Flags().String("reason", "", "Version reason (required)")
	runtimeStandardActivateCmd.Flags().String("expected-active-version-id", "", "Expected active version ID")
	runtimeStandardActivateCmd.Flags().String("reason", "", "Activation reason (required)")
	runtimeStandardRollbackCmd.Flags().String("target-version-id", "", "Prior immutable version ID (required)")
	runtimeStandardRollbackCmd.Flags().String("expected-active-version-id", "", "Expected active version ID")
	runtimeStandardRollbackCmd.Flags().String("reason", "", "Rollback reason (required)")

	runtimeSessionCreateCmd.Flags().String("name", "", "Session name (required)")
	runtimeSessionCreateCmd.Flags().String("provider", "", "Provider (required)")
	runtimeSessionCreateCmd.Flags().String("runtime-kind", "", "Runtime kind (required)")
	runtimeSessionCreateCmd.Flags().String("standard-id", "", "Runtime standard ID (required)")
	runtimeSessionEnrollCmd.Flags().String("runtime-id", "", "Existing runtime ID (required)")
	runtimeSessionEnrollCmd.Flags().String("agent-id", "", "Existing agent ID (required)")

	runtimeBindingConfigureCmd.Flags().String("model", "", "Pinned capability model (required)")
	runtimeBindingConfigureCmd.Flags().String("reasoning-effort", "", "Pinned capability reasoning effort")
	runtimeBindingConfigureCmd.Flags().String("reason", "", "Version reason (required)")
	runtimeBindingActivateCmd.Flags().String("expected-active-version-id", "", "Expected active version ID")
	runtimeBindingActivateCmd.Flags().Int64("expected-binding-generation", -1, "Expected binding generation (required)")
	runtimeBindingActivateCmd.Flags().String("reason", "", "Activation reason (required)")
	runtimeBindingRollbackCmd.Flags().String("target-version-id", "", "Prior immutable version ID (required)")
	runtimeBindingRollbackCmd.Flags().String("expected-active-version-id", "", "Expected active version ID")
	runtimeBindingRollbackCmd.Flags().String("reason", "", "Rollback reason (required)")

	runtimeHomeAttachCmd.Flags().String("home-ref", "", "Opaque home reference (required)")
	runtimeHomeAttachCmd.Flags().Int64("expected-binding-generation", -1, "Expected binding generation (required)")
	runtimeHomeAttachCmd.Flags().Int64("expected-catalog-generation", -1, "Expected catalog generation (required)")
}

func runRuntimeStandardList(cmd *cobra.Command, _ []string) error {
	return rmList(cmd, "/api/runtime-standards", []string{"ID", "NAME", "ACTIVE_VERSION", "STATE"}, []string{"id", "name", "active_version_id", "state"})
}

func runRuntimeStandardGet(cmd *cobra.Command, args []string) error {
	client, ctx, cancel, err := rmClient(cmd, false)
	if err != nil {
		return err
	}
	defer cancel()
	var item map[string]any
	if err := client.GetJSON(ctx, "/api/runtime-standards/"+rmEscape(args[0]), &item); err != nil {
		return fmt.Errorf("get runtime standard: %w", err)
	}
	return rmOutput(cmd, item, []string{"ID", "NAME", "ACTIVE_VERSION", "STATE"}, []string{"id", "name", "active_version_id", "state"})
}

func runRuntimeStandardCreateVersion(cmd *cobra.Command, args []string) error {
	raw := requiredString(cmd, "configuration")
	reason := requiredString(cmd, "reason")
	if raw == "" || reason == "" {
		return fmt.Errorf("--configuration and --reason are required")
	}
	var configuration map[string]any
	if err := json.Unmarshal([]byte(raw), &configuration); err != nil {
		return fmt.Errorf("invalid --configuration JSON: %w", err)
	}
	return rmPostOutput(cmd, "/api/runtime-standards/"+rmEscape(args[0])+"/versions", map[string]any{"configuration": configuration, "reason": reason})
}

func runRuntimeStandardDiff(cmd *cobra.Command, args []string) error {
	return rmDiff(cmd, "/api/runtime-standards/"+rmEscape(args[0]), args[1])
}

func runRuntimeStandardValidate(cmd *cobra.Command, args []string) error {
	return rmPostOutput(cmd, "/api/runtime-standards/"+rmEscape(args[0])+"/versions/"+rmEscape(args[1])+"/validate", map[string]any{})
}

func runRuntimeStandardActivate(cmd *cobra.Command, args []string) error {
	reason := requiredString(cmd, "reason")
	if reason == "" {
		return fmt.Errorf("--reason is required")
	}
	expected, _ := cmd.Flags().GetString("expected-active-version-id")
	return rmPostOutput(cmd, "/api/runtime-standards/"+rmEscape(args[0])+"/versions/"+rmEscape(args[1])+"/activate", map[string]any{"expected_active_version_id": nilIfEmpty(expected), "reason": reason})
}

func runRuntimeStandardRollback(cmd *cobra.Command, args []string) error {
	target := requiredString(cmd, "target-version-id")
	reason := requiredString(cmd, "reason")
	if target == "" || reason == "" {
		return fmt.Errorf("--target-version-id and --reason are required")
	}
	expected, _ := cmd.Flags().GetString("expected-active-version-id")
	return rmPostOutput(cmd, "/api/runtime-standards/"+rmEscape(args[0])+"/rollback", map[string]any{"target_version_id": target, "expected_active_version_id": nilIfEmpty(expected), "reason": reason})
}

func runRuntimeSessionList(cmd *cobra.Command, _ []string) error {
	return rmList(cmd, "/api/runtime-sessions", []string{"ID", "NAME", "PROVIDER", "RUNTIME_KIND", "STANDARD_ID", "STATE"}, []string{"id", "name", "provider", "runtime_kind", "standard_id", "state"})
}

func runRuntimeSessionCreate(cmd *cobra.Command, _ []string) error {
	body := map[string]any{}
	for _, flag := range []string{"name", "provider", "runtime-kind", "standard-id"} {
		value := requiredString(cmd, flag)
		if value == "" {
			return fmt.Errorf("--%s is required", flag)
		}
		body[strings.ReplaceAll(flag, "-", "_")] = value
	}
	return rmPostOutput(cmd, "/api/runtime-sessions", body)
}

func runRuntimeSessionEnroll(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	runtimeID := requiredString(cmd, "runtime-id")
	agentID := requiredString(cmd, "agent-id")
	if runtimeID == "" || agentID == "" {
		return fmt.Errorf("--runtime-id and --agent-id are required")
	}
	path := "/api/workspaces/" + rmEscape(workspaceID) + "/runtime-sessions/" + rmEscape(args[0]) + "/enroll"
	return rmPostOutput(cmd, path, map[string]any{"runtime_id": runtimeID, "agent_id": agentID})
}

func runRuntimeBindingList(cmd *cobra.Command, _ []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	return rmList(cmd, "/api/workspaces/"+rmEscape(workspaceID)+"/runtime-bindings", []string{"ID", "RUNTIME_ID", "PROVIDER", "STATE", "GENERATION", "HOME_REF", "ACTIVE_VERSION"}, []string{"id", "runtime_id", "provider", "state", "generation", "home_ref", "active_configuration_version_id"})
}

func runRuntimeBindingConfigure(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	model := requiredString(cmd, "model")
	reason := requiredString(cmd, "reason")
	reasoning, _ := cmd.Flags().GetString("reasoning-effort")
	if model == "" || reason == "" {
		return fmt.Errorf("--model and --reason are required")
	}

	client, ctx, cancel, err := rmClient(cmd, true)
	if err != nil {
		return err
	}
	defer cancel()
	base := "/api/workspaces/" + rmEscape(workspaceID) + "/runtime-bindings/" + rmEscape(args[0])
	var binding map[string]any
	if err := client.GetJSON(ctx, base, &binding); err != nil {
		return fmt.Errorf("get runtime binding: %w", err)
	}
	if !rmCapabilitySupports(binding["capabilities"], model, reasoning) {
		return fmt.Errorf("model/reasoning selection is not present in the pinned capability projection")
	}
	configuration, ok := cloneMap(binding["active_configuration"])
	if !ok {
		return fmt.Errorf("runtime binding has no active configuration to clone")
	}
	values, ok := configuration["values"].(map[string]any)
	if !ok {
		return fmt.Errorf("active configuration has no values")
	}
	values["model"] = model
	if reasoning == "" {
		delete(values, "reasoning_effort")
	} else {
		values["reasoning_effort"] = reasoning
	}
	var created map[string]any
	if err := client.PostJSON(ctx, base+"/configuration-versions", map[string]any{"configuration": configuration, "reason": reason}, &created); err != nil {
		return fmt.Errorf("create runtime configuration version: %w", err)
	}
	return rmOutput(cmd, created, []string{"ID", "STATE", "APPLY_CLASS", "VALIDATION"}, []string{"id", "state", "apply_class", "validation.status"})
}

func runRuntimeBindingDiff(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	return rmDiff(cmd, "/api/workspaces/"+rmEscape(workspaceID)+"/runtime-bindings/"+rmEscape(args[0]), args[1])
}

func runRuntimeBindingValidate(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	path := "/api/workspaces/" + rmEscape(workspaceID) + "/runtime-bindings/" + rmEscape(args[0]) + "/configuration-versions/" + rmEscape(args[1]) + "/validate"
	return rmPostOutput(cmd, path, map[string]any{})
}

func runRuntimeBindingActivate(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	reason := requiredString(cmd, "reason")
	generation, _ := cmd.Flags().GetInt64("expected-binding-generation")
	if reason == "" || generation < 0 {
		return fmt.Errorf("--reason and --expected-binding-generation are required")
	}
	expected, _ := cmd.Flags().GetString("expected-active-version-id")
	path := "/api/workspaces/" + rmEscape(workspaceID) + "/runtime-bindings/" + rmEscape(args[0]) + "/configuration-versions/" + rmEscape(args[1]) + "/activate"
	return rmPostOutput(cmd, path, map[string]any{"expected_active_version_id": nilIfEmpty(expected), "expected_binding_generation": generation, "reason": reason})
}

func runRuntimeBindingRollback(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	target := requiredString(cmd, "target-version-id")
	reason := requiredString(cmd, "reason")
	if target == "" || reason == "" {
		return fmt.Errorf("--target-version-id and --reason are required")
	}
	expected, _ := cmd.Flags().GetString("expected-active-version-id")
	path := "/api/workspaces/" + rmEscape(workspaceID) + "/runtime-bindings/" + rmEscape(args[0]) + "/rollback"
	return rmPostOutput(cmd, path, map[string]any{"target_version_id": target, "expected_active_version_id": nilIfEmpty(expected), "reason": reason})
}

func runRuntimeHomeList(cmd *cobra.Command, _ []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	return rmList(cmd, "/api/workspaces/"+rmEscape(workspaceID)+"/credential-homes", []string{"HOME_REF", "PROVIDER", "STATE", "HEALTH", "CATALOG_GENERATION"}, []string{"home_ref", "provider", "state", "health", "catalog_generation"})
}

func runRuntimeHomeAttach(cmd *cobra.Command, args []string) error {
	workspaceID, err := requireWorkspaceID(cmd)
	if err != nil {
		return err
	}
	homeRef := requiredString(cmd, "home-ref")
	bindingGeneration, _ := cmd.Flags().GetInt64("expected-binding-generation")
	catalogGeneration, _ := cmd.Flags().GetInt64("expected-catalog-generation")
	if homeRef == "" || bindingGeneration < 0 || catalogGeneration < 0 {
		return fmt.Errorf("--home-ref, --expected-binding-generation, and --expected-catalog-generation are required")
	}
	path := "/api/workspaces/" + rmEscape(workspaceID) + "/runtime-bindings/" + rmEscape(args[0]) + "/home-assignments"
	return rmPostOutput(cmd, path, map[string]any{"home_ref": homeRef, "expected_binding_generation": bindingGeneration, "expected_catalog_generation": catalogGeneration})
}

func rmClient(cmd *cobra.Command, mutation bool) (*cli.APIClient, context.Context, context.CancelFunc, error) {
	client, err := newAPIClient(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	if mutation {
		client.HTTPClient.Transport = idempotencyTransport{base: client.HTTPClient.Transport, key: newIdempotencyKey()}
	}
	ctx, cancel := cli.APIContext(context.Background())
	return client, ctx, cancel, nil
}

func rmPostOutput(cmd *cobra.Command, path string, body map[string]any) error {
	client, ctx, cancel, err := rmClient(cmd, true)
	if err != nil {
		return err
	}
	defer cancel()
	var result map[string]any
	if err := client.PostJSON(ctx, path, body, &result); err != nil {
		return fmt.Errorf("runtime manager mutation: %w", err)
	}
	return rmOutput(cmd, result, []string{"ID", "STATE", "ACTIVE_VERSION", "APPLY_CLASS", "VALIDATION"}, []string{"id", "state", "active_version_id", "apply_class", "validation.status"})
}

func rmList(cmd *cobra.Command, path string, headers, fields []string) error {
	client, ctx, cancel, err := rmClient(cmd, false)
	if err != nil {
		return err
	}
	defer cancel()
	var page struct {
		Items      []map[string]any `json:"items"`
		NextCursor any              `json:"next_cursor"`
	}
	if err := client.GetJSON(ctx, path, &page); err != nil {
		return fmt.Errorf("runtime manager list: %w", err)
	}
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, page)
	}
	rows := make([][]string, 0, len(page.Items))
	for _, item := range page.Items {
		rows = append(rows, rmRow(item, fields))
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func rmOutput(cmd *cobra.Command, item map[string]any, headers, fields []string) error {
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, item)
	}
	cli.PrintTable(os.Stdout, headers, [][]string{rmRow(item, fields)})
	return nil
}

func rmRow(item map[string]any, fields []string) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, rmValue(item, field))
	}
	return row
}

func rmValue(item map[string]any, field string) string {
	var value any = item
	for _, part := range strings.Split(field, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return ""
		}
		value = object[part]
	}
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

var safeRuntimeConfigFields = []string{"transport_binding", "cli_kind", "provider", "model", "reasoning_effort", "max_input_tokens", "max_output_tokens", "max_tool_calls"}

func rmDiff(cmd *cobra.Command, path, versionID string) error {
	client, ctx, cancel, err := rmClient(cmd, false)
	if err != nil {
		return err
	}
	defer cancel()
	var detail map[string]any
	if err := client.GetJSON(ctx, path, &detail); err != nil {
		return fmt.Errorf("get version detail: %w", err)
	}
	before, _ := detail["active_configuration"].(map[string]any)
	var after map[string]any
	for _, key := range []string{"configuration_versions", "versions"} {
		if versions, ok := detail[key].([]any); ok {
			for _, raw := range versions {
				version, _ := raw.(map[string]any)
				if rmValue(version, "id") == versionID {
					after, _ = version["configuration"].(map[string]any)
				}
			}
		}
	}
	if after == nil {
		return fmt.Errorf("version %q not found", versionID)
	}
	diffs := rmSafeDiff(before, after)
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, diffs)
	}
	rows := make([][]string, 0, len(diffs))
	for _, diff := range diffs {
		rows = append(rows, []string{rmValue(diff, "field"), rmValue(diff, "before"), rmValue(diff, "after")})
	}
	cli.PrintTable(os.Stdout, []string{"FIELD", "BEFORE", "AFTER"}, rows)
	return nil
}

func rmSafeDiff(before, after map[string]any) []map[string]any {
	left, right := rmSafeConfig(before), rmSafeConfig(after)
	result := []map[string]any{}
	for _, field := range safeRuntimeConfigFields {
		if left[field] != right[field] {
			result = append(result, map[string]any{"field": field, "before": left[field], "after": right[field]})
		}
	}
	return result
}

func rmSafeConfig(configuration map[string]any) map[string]string {
	values, _ := configuration["values"].(map[string]any)
	limits, _ := values["limits"].(map[string]any)
	result := map[string]string{}
	for _, field := range safeRuntimeConfigFields[:5] {
		result[field] = rmValue(values, field)
	}
	for _, field := range safeRuntimeConfigFields[5:] {
		result[field] = rmValue(limits, field)
	}
	return result
}

func rmCapabilitySupports(raw any, model, reasoning string) bool {
	projection, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	models := projection["models"]
	var selected map[string]any
	switch typed := models.(type) {
	case map[string]any:
		selected, _ = typed[model].(map[string]any)
	case []any:
		for _, rawModel := range typed {
			candidate, _ := rawModel.(map[string]any)
			if rmValue(candidate, "model_id") == model || rmValue(candidate, "id") == model {
				selected = candidate
				break
			}
		}
	}
	if selected == nil {
		return false
	}
	if reasoning == "" {
		return true
	}
	efforts, _ := selected["reasoning_efforts"].([]any)
	for _, effort := range efforts {
		if fmt.Sprint(effort) == reasoning {
			return true
		}
	}
	return false
}

func cloneMap(value any) (map[string]any, bool) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var clone map[string]any
	if json.Unmarshal(data, &clone) != nil || clone == nil {
		return nil, false
	}
	return clone, true
}

func requiredString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return strings.TrimSpace(value)
}
func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func rmEscape(value string) string { return url.PathEscape(value) }

type idempotencyTransport struct {
	base http.RoundTripper
	key  string
}

func (t idempotencyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set("Idempotency-Key", t.key)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

func newIdempotencyKey() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}
	return "runtime-manager-cli"
}
