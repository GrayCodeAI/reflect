package env

import "os"

// legacyPrefix is the pre-rename environment-variable prefix. Decoder was formerly
// named Hawk; reads fall back to HAWK_* so existing shell profiles, CI configs,
// and service units keep working without edits.
const (
	prefix       = "DECODER_"
	legacyPrefix = "HAWK_"
)

// GetenvDecoder reads a Decoder environment variable by its suffix (the part after
// "DECODER_"), falling back to the legacy HAWK_<suffix> name. The DECODER_ form wins
// when both are set, so users can migrate incrementally.
//
// Example: GetenvDecoder("STATE_DIR") reads DECODER_STATE_DIR, then HAWK_STATE_DIR.
func GetenvDecoder(suffix string) string {
	if v := os.Getenv(prefix + suffix); v != "" {
		return v
	}
	return os.Getenv(legacyPrefix + suffix)
}

// LookupDecoder is GetenvDecoder with an explicit presence flag. A variable set to an
// empty value counts as absent, matching GetenvDecoder's fallback behavior.
func LookupDecoder(suffix string) (string, bool) {
	if v := os.Getenv(prefix + suffix); v != "" {
		return v, true
	}
	if v := os.Getenv(legacyPrefix + suffix); v != "" {
		return v, true
	}
	return "", false
}
