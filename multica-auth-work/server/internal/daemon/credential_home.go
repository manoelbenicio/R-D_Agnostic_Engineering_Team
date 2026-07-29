package daemon

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	credentialSlotsRootEnv = "MULTICA_CREDENTIAL_SLOTS_ROOT"
	assignmentFileName     = "multica-assignments.v1.json"
)

var credentialAssignmentMu sync.Mutex

type credentialAssignmentDocument struct {
	Version     int               `json:"version"`
	Assignments map[string]string `json:"assignments"`
}

func canonicalCredentialProvider(provider string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "agy", "antigravity":
		return "antigravity", true
	case "codex":
		return "codex", true
	case "kiro":
		return "kiro", true
	default:
		return "", false
	}
}

func credentialSlotAllowlistEnv(provider string) string {
	return "MULTICA_CREDENTIAL_SLOT_ALLOWLIST_" + strings.ToUpper(provider)
}

func credentialProviderPath(slotRoot, provider string) (home, requiredRelative string, requiredDirectory bool) {
	switch provider {
	case "antigravity":
		return filepath.Join(slotRoot, "home"), filepath.Join(".gemini", "antigravity-cli", "antigravity-oauth-token"), false
	case "codex":
		return filepath.Join(slotRoot, "codex"), "auth.json", false
	case "kiro":
		return filepath.Join(slotRoot, "xdg-data"), filepath.Join("kiro-cli", "data.sqlite3"), false
	default:
		return "", "", false
	}
}

func parseCredentialSlotAllowlist(provider string) ([]string, error) {
	raw := strings.TrimSpace(os.Getenv(credentialSlotAllowlistEnv(provider)))
	if raw == "" {
		return nil, fmt.Errorf("credential isolation: %s is required", credentialSlotAllowlistEnv(provider))
	}
	seen := map[string]struct{}{}
	slots := make([]string, 0)
	for _, field := range strings.Split(raw, ",") {
		value := strings.TrimSpace(field)
		value = strings.TrimPrefix(value, "slot-")
		if value == "" {
			return nil, fmt.Errorf("credential isolation: empty slot in %s", credentialSlotAllowlistEnv(provider))
		}
		for _, r := range value {
			if r < '0' || r > '9' {
				return nil, fmt.Errorf("credential isolation: invalid slot %q in %s", field, credentialSlotAllowlistEnv(provider))
			}
		}
		name := "slot-" + value
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		slots = append(slots, name)
	}
	sort.Strings(slots)
	return slots, nil
}

func resolvePhysicalDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) == string(filepath.Separator) {
		return "", fmt.Errorf("credential isolation: path must be absolute and non-root")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("credential isolation: inspect %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("credential isolation: %s must be a physical directory", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("credential isolation: resolve %s: %w", path, err)
	}
	return filepath.Clean(resolved), nil
}

func pathWithinRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != "." && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func validateCredentialSlot(slotsRoot, slotName, provider string) (string, error) {
	slotPath := filepath.Join(slotsRoot, slotName)
	slotRoot, err := resolvePhysicalDirectory(slotPath)
	if err != nil {
		return "", err
	}
	if !pathWithinRoot(slotsRoot, slotRoot) {
		return "", fmt.Errorf("credential isolation: slot %s escapes slots root", slotName)
	}
	homePath, requiredRelative, requiredDirectory := credentialProviderPath(slotRoot, provider)
	home, err := resolvePhysicalDirectory(homePath)
	if err != nil {
		return "", err
	}
	if !pathWithinRoot(slotRoot, home) {
		return "", fmt.Errorf("credential isolation: provider home for %s escapes slot %s", provider, slotName)
	}
	required := filepath.Join(home, requiredRelative)
	info, err := os.Lstat(required)
	if err != nil {
		return "", fmt.Errorf("credential isolation: required %s artifact missing in %s: %w", provider, slotName, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || (!requiredDirectory && !info.Mode().IsRegular()) ||
		(requiredDirectory && !info.IsDir()) {
		return "", fmt.Errorf("credential isolation: required %s artifact in %s has an invalid type", provider, slotName)
	}
	return home, nil
}

func rendezvousCredentialSlot(key string, slots []string) string {
	var selected string
	var selectedScore uint64
	for _, slot := range slots {
		sum := sha256.Sum256([]byte(key + "\x00" + slot))
		score := binary.BigEndian.Uint64(sum[:8])
		if selected == "" || score > selectedScore || (score == selectedScore && slot < selected) {
			selected = slot
			selectedScore = score
		}
	}
	return selected
}

func loadCredentialAssignments(path string) (credentialAssignmentDocument, error) {
	document := credentialAssignmentDocument{Version: 1, Assignments: map[string]string{}}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return document, nil
	}
	if err != nil {
		return document, fmt.Errorf("credential isolation: read assignments: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil || document.Version != 1 || document.Assignments == nil {
		return credentialAssignmentDocument{}, fmt.Errorf("credential isolation: invalid assignment document")
	}
	return document, nil
}

func persistCredentialAssignments(path string, document credentialAssignmentDocument) error {
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("credential isolation: encode assignments: %w", err)
	}
	raw = append(raw, '\n')
	temp, err := os.CreateTemp(filepath.Dir(path), ".multica-assignments-")
	if err != nil {
		return fmt.Errorf("credential isolation: create assignment temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("credential isolation: restrict assignment temp file: %w", err)
	}
	if _, err := temp.Write(raw); err != nil {
		temp.Close()
		return fmt.Errorf("credential isolation: write assignments: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("credential isolation: sync assignments: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("credential isolation: close assignments: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("credential isolation: publish assignments: %w", err)
	}
	return os.Chmod(path, 0o600)
}

func resolveCredentialAccountHome(agentID, provider string) (string, error) {
	canonical, required := canonicalCredentialProvider(provider)
	if !required {
		return "", nil
	}
	if strings.TrimSpace(agentID) == "" {
		return "", fmt.Errorf("credential isolation: agent id is required for provider %s", canonical)
	}
	configuredRoot := strings.TrimSpace(os.Getenv(credentialSlotsRootEnv))
	if configuredRoot == "" {
		return "", fmt.Errorf("credential isolation: %s is required for provider %s", credentialSlotsRootEnv, canonical)
	}
	slotsRoot, err := resolvePhysicalDirectory(configuredRoot)
	if err != nil {
		return "", err
	}
	slots, err := parseCredentialSlotAllowlist(canonical)
	if err != nil {
		return "", err
	}

	validHomes := make(map[string]string, len(slots))
	for _, slot := range slots {
		home, validateErr := validateCredentialSlot(slotsRoot, slot, canonical)
		if validateErr != nil {
			return "", validateErr
		}
		validHomes[slot] = home
	}

	credentialAssignmentMu.Lock()
	defer credentialAssignmentMu.Unlock()
	assignmentPath := filepath.Join(filepath.Dir(slotsRoot), assignmentFileName)
	document, err := loadCredentialAssignments(assignmentPath)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(agentID) + "|" + canonical
	if assigned := document.Assignments[key]; assigned != "" {
		if home, ok := validHomes[assigned]; ok {
			return home, nil
		}
		// The operator allowlist is authoritative. A persisted assignment that
		// leaves it must never be used, but it must not wedge the agent forever:
		// rendezvous below selects from the currently validated slots and the
		// replacement is published through the same atomic assignment write.
		delete(document.Assignments, key)
	}
	selected := rendezvousCredentialSlot(key, slots)
	if selected == "" {
		return "", fmt.Errorf("credential isolation: no eligible slot for provider %s", canonical)
	}
	document.Assignments[key] = selected
	if err := persistCredentialAssignments(assignmentPath, document); err != nil {
		return "", err
	}
	return validHomes[selected], nil
}

// resolveCredentialModelDiscoveryHomes returns every validated provider home
// in the operator allowlist. Model discovery has no agent identity to bind, so
// it must not create or reuse a task-affinity assignment. Callers may try the
// homes in order and stop at the first live provider session.
func resolveCredentialModelDiscoveryHomes(provider string) ([]string, error) {
	canonical, required := canonicalCredentialProvider(provider)
	if !required {
		return nil, nil
	}
	configuredRoot := strings.TrimSpace(os.Getenv(credentialSlotsRootEnv))
	if configuredRoot == "" {
		return nil, fmt.Errorf("credential isolation: %s is required for provider %s", credentialSlotsRootEnv, canonical)
	}
	slotsRoot, err := resolvePhysicalDirectory(configuredRoot)
	if err != nil {
		return nil, err
	}
	slots, err := parseCredentialSlotAllowlist(canonical)
	if err != nil {
		return nil, err
	}
	homes := make([]string, 0, len(slots))
	for _, slot := range slots {
		home, validateErr := validateCredentialSlot(slotsRoot, slot, canonical)
		if validateErr != nil {
			return nil, validateErr
		}
		homes = append(homes, home)
	}
	if len(homes) == 0 {
		return nil, fmt.Errorf("credential isolation: no eligible slot for provider %s", canonical)
	}
	return homes, nil
}
