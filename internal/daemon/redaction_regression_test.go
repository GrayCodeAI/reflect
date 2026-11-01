package daemon

import "testing"

func TestCredentialPathRedaction(t *testing.T) {
	for input, want := range map[string]string{
		"/botFAKE_AUDIT_TOKEN/sendMessage": "/bot<REDACTED>/sendMessage",
		"/botFAKE_AUDIT_TOKEN":             "/bot<REDACTED>",
		"/v1/status":                       "/v1/status",
	} {
		if got := redactURLPath(input); got != want {
			t.Errorf("redactURLPath(%q) = %q, want %q", input, got, want)
		}
	}
}
