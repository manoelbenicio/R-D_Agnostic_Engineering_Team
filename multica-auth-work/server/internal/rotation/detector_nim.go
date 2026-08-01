package rotation

import "regexp"

// matchesNIMExhaustion detects quota/rate exhaustion emitted by the native
// NVIDIA NIM HTTP backend. The backend includes a stable "NIM API returned
// 429" prefix for non-2xx responses; textual limits require retry/reset
// context to avoid rotating on documentation or normal output.
func matchesNIMExhaustion(screenText string) bool {
	if screenText == "" {
		return false
	}
	if nimHTTP429Pattern.MatchString(screenText) {
		return true
	}
	return nimLimitPattern.MatchString(screenText) && nimResetPattern.MatchString(screenText)
}

var (
	nimHTTP429Pattern = regexp.MustCompile(`(?i)\bNIM\s+API\s+returned\s+429\b`)
	nimLimitPattern   = regexp.MustCompile(`(?i)\b(?:rate\s+limit\s+(?:exceeded|reached)|quota\s+(?:exceeded|depleted)|resource\s+exhausted|too\s+many\s+requests|credit(?:s|\s+balance)?\s+(?:exhausted|depleted)|usage\s+limit\s+reached)\b`)
	nimResetPattern   = regexp.MustCompile(`(?i)\b(?:resets?|try\s+again|retry(?:ing)?|cooldown|back[\s-]?off|after)\b`)
)
