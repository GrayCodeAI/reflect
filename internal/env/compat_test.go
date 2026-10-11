package env

import "testing"

func TestGetenvDecoderPrefersCurrentName(t *testing.T) {
	t.Setenv("DECODER_STATE_DIR", "/current")
	t.Setenv("HAWK_STATE_DIR", "/legacy")
	if got := GetenvDecoder("STATE_DIR"); got != "/current" {
		t.Fatalf("GetenvDecoder = %q, want /current", got)
	}
}

func TestGetenvDecoderFallsBackToLegacy(t *testing.T) {
	t.Setenv("DECODER_STATE_DIR", "")
	t.Setenv("HAWK_STATE_DIR", "/legacy")
	if got := GetenvDecoder("STATE_DIR"); got != "/legacy" {
		t.Fatalf("GetenvDecoder = %q, want /legacy", got)
	}
}

func TestGetenvDecoderEmptyWhenUnset(t *testing.T) {
	t.Setenv("DECODER_STATE_DIR", "")
	t.Setenv("HAWK_STATE_DIR", "")
	if got := GetenvDecoder("STATE_DIR"); got != "" {
		t.Fatalf("GetenvDecoder = %q, want empty", got)
	}
}

func TestLookupDecoderReportsPresence(t *testing.T) {
	t.Setenv("DECODER_FEATURE", "")
	t.Setenv("HAWK_FEATURE", "1")
	v, ok := LookupDecoder("FEATURE")
	if !ok || v != "1" {
		t.Fatalf("LookupDecoder = (%q, %v), want (1, true)", v, ok)
	}

	t.Setenv("HAWK_FEATURE", "")
	if _, ok := LookupDecoder("FEATURE"); ok {
		t.Fatal("LookupDecoder reported presence for an unset variable")
	}
}
