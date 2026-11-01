package tool

import "testing"

func TestExploreBashAllowed_ReadOK(t *testing.T) {
	ok := []string{
		"ls -la",
		"pwd",
		"git status",
		"git log --oneline -5",
		"git diff HEAD~1",
		"rg TODO --type go",
		"cat README.md",
		"go list ./...",
		"go version",
		"ls | head",
		"echo hello && pwd",
		"FOO=bar ls",
	}
	for _, c := range ok {
		if err := ExploreBashAllowed(c); err != nil {
			t.Errorf("ExploreBashAllowed(%q)=%v want nil", c, err)
		}
	}
}

func TestExploreBashAllowed_MutationsDenied(t *testing.T) {
	deny := []string{
		"rm -rf /tmp/x",
		"git commit -am msg",
		"git push origin main",
		"git checkout -b feature",
		"go fmt ./...",
		"go get example.com/x",
		"npm install lodash",
		"sed -i 's/a/b/' f.go",
		"find . -delete",
		"echo hi > out.txt",
		"sudo ls",
		"chmod 777 x",
		"mv a b",
		"sed 'w output.txt' input.txt",
		"awk 'BEGIN {system(\"touch output\")}'",
		"python3 -c 'open(\"output\", \"w\").close()'",
		"node -e 'process.exit(0)'",
		"git -C /tmp commit -am msg",
		"git branch new-branch",
		"git tag new-tag",
		"git stash",
		"git config core.pager something",
		"git -c core.pager=something status",
		"git diff --output=output.txt",
		"git diff --ext-diff",
		"go env -w GOFLAGS=-x",
		"go test ./...",
		"go tool compile file.go",
		"npm test",
		"find . -fprintf output.txt '%p'",
		"sort -o output.txt input.txt",
		"sort --compress-program=program input.txt",
		"printf -v PATH /tmp",
		"rg --pre=program pattern",
		"echo hello\ntouch output.txt",
		"echo $(touch output.txt)",
		"PATH=/tmp ls",
		"./ls",
		"curl -o output.txt https://example.com",
	}
	for _, c := range deny {
		if err := ExploreBashAllowed(c); err == nil {
			t.Errorf("ExploreBashAllowed(%q)=nil want error", c)
		}
	}
}

func TestExploreBashAllowed_PipelineGate(t *testing.T) {
	// second stage mutating
	if err := ExploreBashAllowed("cat f | tee out"); err == nil {
		t.Fatal("tee must be denied")
	}
	if err := ExploreBashAllowed("ls; rm -rf x"); err == nil {
		t.Fatal("expected deny for multi-segment with rm")
	}
}
