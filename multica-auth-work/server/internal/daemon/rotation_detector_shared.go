package daemon

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/rotation"
)

// Daemon-layer reactive exhaustion detectors for the credential-isolation fix
// (vendors without a detector: Kiro, Cline, OpenCode).
//
// This file holds the SHARED scaffolding ported from the reference Python
// detector AOP/control-plane/rotation/detector.py (QuotaExhaustionDetector):
// detect_text (per-vendor regex) + detect_status_code (HTTP 429 vs 503) +
// reset-time parsing. Each vendor gets its own detector struct
// (KiroExhaustionDetector / ClineExhaustionDetector / OpenCodeExhaustionDetector)
// in its own file, implementing rotation.ExhaustionDetector so it can be swapped
// into daemon.go's rotationDetector field by the daemon owner without changing
// the rotation package and without editing daemon.go (don't-touch).
//
// Detection rules (doc 36 §2.1):
//   - HTTP 429  -> quota exhaustion (rotate).
//   - HTTP 503 / "high traffic" screen -> server overload (do NOT rotate;
//     retry/switch model).
//   - otherwise -> apply the vendor screen matcher; on match, parse the reset
//     hint (clock shapes); non-clock hints return nil so callers fall back to
//     the 5h window.
//
// No credential material is inspected — only public vendor screen text and HTTP
// status — so no secret can leak into logs.

// vendorScreenMatcher is the per-vendor screen-text predicate a daemon-layer
// rotation detector supplies. It returns true when the banner indicates quota
// exhaustion for that vendor.
type vendorScreenMatcher func(screenText string) bool

// detectExhaustion is the shared reactive detection flow. It mirrors
// QuotaExhaustionDetector.detect_text + detect_status_code from the reference
// Python detector, returning a rotation.DetectionResult so every vendor
// detector satisfies rotation.ExhaustionDetector. The vendor argument is
// reserved for detect_any-style extension / structured logging and never holds a
// secret (it is a public vendor name).
func detectExhaustion(matcher vendorScreenMatcher, vendor, screenText string, httpStatus int, now time.Time) rotation.DetectionResult {
	_ = vendor
	if httpStatus == 429 {
		return rotation.DetectionResult{
			Exhausted: true,
			Signal:    rotation.SignalHTTP429,
			ResetAt:   parseExhaustionResetAt(screenText, now),
		}
	}
	if httpStatus == 503 || daemonHighTrafficPattern.MatchString(screenText) {
		return rotation.DetectionResult{}
	}
	if matcher == nil || !matcher(screenText) {
		return rotation.DetectionResult{}
	}
	return rotation.DetectionResult{
		Exhausted: true,
		Signal:    rotation.SignalScreen,
		ResetAt:   parseExhaustionResetAt(screenText, now),
	}
}

// parseExhaustionResetAt parses the clock-shaped reset hint embedded in a vendor
// banner ("reset at 2pm", "resume using this model at 3:36 PM", "try again at
// 4:05 PM") into an absolute time on or after now. Non-clock hints (relative "in
// 30 minutes", date "on 2026-07-10") return nil so callers fall back to the 5h
// window (doc 36 §2.1). Ported to stay self-contained in package daemon
// (rotation.parseResetAt is private to package rotation).
func parseExhaustionResetAt(text string, now time.Time) *time.Time {
	match := daemonResetAtPattern.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	hour, err := strconv.Atoi(match[1])
	if err != nil || hour < 1 || hour > 12 {
		return nil
	}
	minute := 0
	if match[2] != "" {
		minute, err = strconv.Atoi(match[2])
		if err != nil || minute < 0 || minute > 59 {
			return nil
		}
	}
	meridiem := strings.ToLower(strings.ReplaceAll(match[3], ".", ""))
	switch meridiem {
	case "am":
		if hour == 12 {
			hour = 0
		}
	case "pm":
		if hour != 12 {
			hour += 12
		}
	default:
		return nil
	}
	reset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if reset.Before(now) {
		reset = reset.Add(24 * time.Hour)
	}
	return &reset
}

var (
	daemonHighTrafficPattern = regexp.MustCompile(`(?i)\bhigh traffic\b`)
	daemonResetAtPattern     = regexp.MustCompile(`(?i)\b(?:resets?|resume(?:\s+using\s+this\s+model)?|try\s+again)\b[^\n.]{0,120}?\bat\s+([0-9]{1,2})(?::([0-9]{2}))?\s*([ap]\.?m\.?)\b`)
)
