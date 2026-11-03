package tool

import "testing"

func TestShellSafetyAllowsNamedGoDiagnostics(t *testing.T) {
	for _, command := range []string{
		"go env GOTOOLCHAIN",
		"go env GOTOOLCHAIN GOFLAGS GOROOT",
		"go env -json GOPATH GOMOD GOWORK",
		"cd /tmp && go env GOCACHE; go version",
		"export PATH=\"$HOME/.local/go/bin:$PATH\"; cd /tmp && go env GOTOOLCHAIN",
	} {
		t.Run(command, func(t *testing.T) {
			if err := validateShellCommand(t.Context(), command); err != nil {
				t.Fatalf("safe diagnostic blocked: %v", err)
			}
		})
	}
}

func TestShellSafetyStillBlocksEnvironmentDumps(t *testing.T) {
	for _, command := range []string{
		"env", "printenv", "go env", "go env -json", "go env GOPROXY", "go env GOAUTH",
		"go env -w GOFLAGS=-x", "go env -u GOFLAGS", "go env GOTOOLCHAIN && env",
		"go env GOTOOLCHAIN | printenv", "go env GOTOOLCHAIN; printenv OPENAI_API_KEY",
		"go env GOTOOLCHAIN $(printenv)", "go env GOTOOLCHAIN\nprintenv", "echo 'go env GOTOOLCHAIN'",
	} {
		t.Run(command, func(t *testing.T) {
			if err := validateShellCommand(t.Context(), command); err == nil {
				t.Fatal("environment dump protection was bypassed")
			}
		})
	}
}
