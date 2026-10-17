package sandbox

import (
	"context"
	"os"
	"testing"
)

func TestNormalizeMode(t *testing.T) {
	cases := []struct {
		in   string
		want Mode
	}{
		{"", ModeOff},
		{"off", ModeOff},
		{"docker", ModeDocker},
		{"container", ModeDocker},
		{"DOCKER", ModeDocker},
		{"bogus", ModeOff},
	}
	for _, c := range cases {
		if got := NormalizeMode(c.in); got != c.want {
			t.Errorf("NormalizeMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEnabled(t *testing.T) {
	if Enabled(Config{Mode: ModeOff}) {
		t.Error("off should not be enabled")
	}
	if !Enabled(Config{Mode: ModeDocker}) {
		t.Error("docker should be enabled")
	}
}

func TestRunOffReturnsError(t *testing.T) {
	_, err := Run(context.Background(), Config{Mode: ModeOff}, "/tmp", "echo hi")
	if err == nil {
		t.Fatal("expected error for off mode")
	}
}

func TestRunDockerIfAvailable(t *testing.T) {
	if !DockerAvailable() {
		t.Skip("docker not available")
	}
	out, err := Run(context.Background(), Config{Mode: ModeDocker, Image: "alpine:latest"}, os.TempDir(), "echo sandbox-ok")
	if err != nil {
		t.Fatalf("sandbox run failed: %v\n%s", err, out)
	}
	if !contains(out, "sandbox-ok") {
		t.Fatalf("expected sandbox-ok in output, got: %s", out)
	}
}

func TestMicrosandboxIfAvailable(t *testing.T) {
	if !MicrosandboxAvailable() {
		t.Skip("msb not available")
	}
	out, err := Run(context.Background(), Config{Mode: ModeMicrosandbox, Image: "python"}, os.TempDir(), "echo msb-ok")
	if err != nil {
		t.Fatalf("msb run failed: %v\n%s", err, out)
	}
	if !contains(out, "msb-ok") {
		t.Fatalf("expected msb-ok in output, got: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}
