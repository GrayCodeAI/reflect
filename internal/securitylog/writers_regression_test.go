package securitylog

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestIndependentWritersShareChain(t *testing.T) {
	dir := t.TempDir()
	first, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	var wg sync.WaitGroup
	for _, writer := range []*Log{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				if _, err := writer.Append(SeverityInfo, "test", "fixture", "", ""); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if count, err := Verify(dir); err != nil || count != 20 {
		t.Fatalf("count=%d error=%v", count, err)
	}
}

func TestIndependentWriterProcessesShareChain(t *testing.T) {
	dir := t.TempDir()
	var children []*exec.Cmd
	for range 3 {
		child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestSecurityLogWriterProcess$")
		child.Env = append(os.Environ(), "REFLECT_SECURITYLOG_TEST_DIR="+dir)
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		children = append(children, child)
	}
	for _, child := range children {
		if err := child.Wait(); err != nil {
			t.Error(err)
		}
	}
	if count, err := Verify(dir); err != nil || count != 15 {
		t.Fatalf("count=%d error=%v", count, err)
	}
}

func TestSecurityLogWriterProcess(t *testing.T) {
	dir := os.Getenv("REFLECT_SECURITYLOG_TEST_DIR")
	if dir == "" {
		t.Skip("subprocess fixture")
	}
	writer, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	for range 5 {
		if _, err := writer.Append(SeverityInfo, "test", "process fixture", "", ""); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCorruptLogIsPreservedAndNotExtended(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	if _, err := writer.Append(SeverityInfo, "test", "fixture", "", ""); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, logFileName)
	corrupt := []byte("{invalid chain}\n")
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Append(SeverityInfo, "test", "must not append", "", ""); err == nil {
		t.Fatal("append accepted a corrupt chain")
	}
	if reopened, err := New(dir); err == nil {
		_ = reopened.Close()
		t.Fatal("opening a corrupt chain succeeded")
	}
	if got, _ := os.ReadFile(path); string(got) != string(corrupt) {
		t.Fatal("corrupt evidence was rewritten")
	}
}
