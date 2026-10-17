package env

import "testing"

func TestGetenvReflectPrefersCurrentName(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", "/current")
	t.Setenv("HAWK_STATE_DIR", "/legacy")
	if got := GetenvReflect("STATE_DIR"); got != "/current" {
		t.Fatalf("GetenvReflect = %q, want /current", got)
	}
}

func TestGetenvReflectFallsBackToLegacy(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", "")
	t.Setenv("HAWK_STATE_DIR", "/legacy")
	if got := GetenvReflect("STATE_DIR"); got != "/legacy" {
		t.Fatalf("GetenvReflect = %q, want /legacy", got)
	}
}

func TestGetenvReflectEmptyWhenUnset(t *testing.T) {
	t.Setenv("REFLECT_STATE_DIR", "")
	t.Setenv("HAWK_STATE_DIR", "")
	if got := GetenvReflect("STATE_DIR"); got != "" {
		t.Fatalf("GetenvReflect = %q, want empty", got)
	}
}

func TestLookupReflectReportsPresence(t *testing.T) {
	t.Setenv("REFLECT_FEATURE", "")
	t.Setenv("HAWK_FEATURE", "1")
	v, ok := LookupReflect("FEATURE")
	if !ok || v != "1" {
		t.Fatalf("LookupReflect = (%q, %v), want (1, true)", v, ok)
	}

	t.Setenv("HAWK_FEATURE", "")
	if _, ok := LookupReflect("FEATURE"); ok {
		t.Fatal("LookupReflect reported presence for an unset variable")
	}
}
