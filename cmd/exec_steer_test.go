package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrayCodeAI/reflect/internal/engine"
	"github.com/GrayCodeAI/reflect/internal/engine/streaming"
)

// TestPumpSteerReaderLines is table-driven over the line protocol: each
// non-empty, trimmed line becomes one steering message.
func TestPumpSteerReaderLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"simple lines", "first\nsecond\n", []string{"first", "second"}},
		{"blank lines dropped", "first\n\n\nsecond\n", []string{"first", "second"}},
		{"whitespace trimmed", "  padded  \n\ttabbed\t\n", []string{"padded", "tabbed"}},
		{"crlf handled", "one\r\ntwo\r\n", []string{"one", "two"}},
		{"empty input", "\n\n", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sq := streaming.NewSteeringQueue()
			pumpSteerReader(sq, strings.NewReader(tc.in))

			drained := sq.Drain()
			if len(drained) != len(tc.want) {
				t.Fatalf("steer lines = %v, want %v", steerContents(drained), tc.want)
			}
			for i := range drained {
				if drained[i].Content != tc.want[i] {
					t.Fatalf("steer lines = %v, want %v", steerContents(drained), tc.want)
				}
			}
		})
	}
}

func steerContents(msgs []streaming.SteeringMessage) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Content)
	}
	return out
}

func TestPumpExecSteeringFile(t *testing.T) {
	sess := engine.NewSession("", "test-model", "sys", nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "steer.txt")
	if err := os.WriteFile(path, []byte("line one\n\n  line two  \nline three\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := pumpExecSteering(sess, path); err != nil {
		t.Fatalf("pumpExecSteering: %v", err)
	}
	got := steerContents(sess.Persistence().Steering().Drain())
	want := []string{"line one", "line two", "line three"}
	if len(got) != len(want) {
		t.Fatalf("drained %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("drained %v, want %v", got, want)
		}
	}
}

func TestPumpExecSteeringMissingFile(t *testing.T) {
	sess := engine.NewSession("", "test-model", "sys", nil)
	if err := pumpExecSteering(sess, filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("expected an error for a missing steer file")
	}
}

func TestExecSteerFileFlagRegistered(t *testing.T) {
	flag := execCmd.Flags().Lookup("steer-file")
	if flag == nil {
		t.Fatal("exec --steer-file flag is not registered")
	}
	if flag.DefValue != "" {
		t.Fatalf("steer-file default = %q, want empty", flag.DefValue)
	}
}
