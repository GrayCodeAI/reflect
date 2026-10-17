package env

import "os"

// legacyPrefix is the pre-rename environment-variable prefix. Reflect was formerly
// named Hawk; reads fall back to HAWK_* so existing shell profiles, CI configs,
// and service units keep working without edits.
const (
	prefix       = "REFLECT_"
	legacyPrefix = "HAWK_"
)

// GetenvReflect reads a Reflect environment variable by its suffix (the part after
// "REFLECT_"), falling back to the legacy HAWK_<suffix> name. The REFLECT_ form wins
// when both are set, so users can migrate incrementally.
//
// Example: GetenvReflect("STATE_DIR") reads REFLECT_STATE_DIR, then HAWK_STATE_DIR.
func GetenvReflect(suffix string) string {
	if v := os.Getenv(prefix + suffix); v != "" {
		return v
	}
	return os.Getenv(legacyPrefix + suffix)
}

// LookupReflect is GetenvReflect with an explicit presence flag. A variable set to an
// empty value counts as absent, matching GetenvReflect's fallback behavior.
func LookupReflect(suffix string) (string, bool) {
	if v := os.Getenv(prefix + suffix); v != "" {
		return v, true
	}
	if v := os.Getenv(legacyPrefix + suffix); v != "" {
		return v, true
	}
	return "", false
}
