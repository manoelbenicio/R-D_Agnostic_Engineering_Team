package runtimeconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// DigestLength is the frozen digest encoding: a raw lowercase hex SHA-256, no
// algorithm prefix and no separator. The prefix was removed so a digest is
// directly comparable, sortable, and usable as an opaque identity token
// without any parsing step. Anything that is not exactly 64 lowercase hex
// characters is rejected rather than normalised, so an uppercase or prefixed
// value fails closed instead of silently matching.
const DigestLength = 64

type canonicalEffective struct {
	SchemaVersion Version `json:"schema_version"`
	Values        Values  `json:"values"`
	Delegable     []Field `json:"delegable"`
}

// CanonicalRedacted returns the canonical JSON used by Digest. The typed
// contract has no secret-bearing or arbitrary fields, so secret material is
// excluded by construction. Origins are also excluded to keep equivalent
// effective configurations identical regardless of their source layer.
func (e Effective) CanonicalRedacted() ([]byte, error) {
	if e.Version != VersionV1 {
		return nil, ValidationErrors{{Code: ErrInvalidVersion, Field: Field("version")}}
	}
	return canonicalRedacted(e.Values, e.Delegable)
}

func canonicalRedacted(values Values, policy map[Field]bool) ([]byte, error) {
	delegable := make([]Field, 0, len(allFields))
	for _, field := range allFields {
		if policy[field] {
			delegable = append(delegable, field)
		}
	}
	return json.Marshal(canonicalEffective{
		SchemaVersion: VersionV1,
		Values:        cloneValues(values),
		Delegable:     delegable,
	})
}

func digestFor(values Values, policy map[Field]bool) string {
	canonical, err := canonicalRedacted(values, policy)
	if err != nil {
		panic("runtimeconfig: canonical typed configuration could not be encoded: " + err.Error())
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

// ValidDigest reports whether digest is a well-formed raw lowercase hex
// SHA-256. It is a pure format check and says nothing about whether the digest
// matches any particular configuration.
func ValidDigest(digest string) bool {
	if len(digest) != DigestLength {
		return false
	}
	for i := 0; i < len(digest); i++ {
		c := digest[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}

// digestMatches reports whether digest is well formed and is the digest of the
// supplied effective values and policy.
func digestMatches(digest string, values Values, policy map[Field]bool) bool {
	return ValidDigest(digest) && digest == digestFor(values, policy)
}

// canonicalCapabilities is the canonical form hashed by CapabilityDigest. It
// mirrors the effective-configuration canonicalisation: a frozen schema
// version, no origins, no secret-bearing field, and no map iteration order
// leaking into the hash.
type canonicalCapabilities struct {
	SchemaVersion Version                              `json:"schema_version"`
	Provider      ProviderID                           `json:"provider"`
	Models        map[ModelID]canonicalModelCapability `json:"models"`
}

type canonicalModelCapability struct {
	ReasoningEfforts    []ReasoningEffort `json:"reasoning_efforts"`
	MaxInputTokens      int64             `json:"max_input_tokens"`
	MaxOutputTokens     int64             `json:"max_output_tokens"`
	ContextWindowTokens int64             `json:"context_window_tokens"`
	MaxToolCalls        int64             `json:"max_tool_calls"`
}

// CapabilityDigest returns the raw lowercase hex SHA-256 identity of the
// provider capability declaration that admitted a configuration. It exists
// because both accepted downstream contracts require this value and neither
// can synthesise it: storage declares capability_digest NOT NULL with a
// 64-hex check on the activation and snapshot tables, and the accepted client
// contract declares it non-optional on the activation result and on the
// validation result. Only this package holds the capability declaration, so
// only this package can produce its identity.
//
// The digest is the same encoding as an effective configuration digest, so a
// single ValidDigest check covers both and neither needs parsing.
//
// It fails closed rather than hashing whatever it was handed: an invalid
// declaration has no legitimate identity, and returning one would let an
// unvalidated capability set be recorded as the authority for an activation.
// Model keys are ordered by encoding/json and reasoning efforts are sorted, so
// two equal declarations always produce one digest regardless of how the
// caller happened to build the maps and slices.
func CapabilityDigest(capabilities ProviderCapabilities) (string, error) {
	errs := newErrorCollector()
	validateCapabilities(capabilities, errs)
	if failures := errs.errors(); len(failures) != 0 {
		return "", failures
	}

	models := make(map[ModelID]canonicalModelCapability, len(capabilities.Models))
	for model, capability := range capabilities.Models {
		efforts := make([]ReasoningEffort, 0, len(capability.ReasoningEfforts))
		efforts = append(efforts, capability.ReasoningEfforts...)
		sort.Slice(efforts, func(i, j int) bool { return efforts[i] < efforts[j] })
		models[model] = canonicalModelCapability{
			ReasoningEfforts:    efforts,
			MaxInputTokens:      capability.MaxInputTokens,
			MaxOutputTokens:     capability.MaxOutputTokens,
			ContextWindowTokens: capability.ContextWindowTokens,
			MaxToolCalls:        capability.MaxToolCalls,
		}
	}

	canonical, err := json.Marshal(canonicalCapabilities{
		SchemaVersion: VersionV1,
		Provider:      capabilities.Provider,
		Models:        models,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
