package tool

import (
	"fmt"
	"strings"
)

// exploreBashAllow is the first-token allowlist for explore/plan (read-only) modes.
// Anything not listed is denied — fail-closed.
var exploreBashAllow = map[string]bool{
	"ls": true, "pwd": true, "dirname": true, "basename": true,
	"cat": true, "head": true, "tail": true, "wc": true,
	"file": true, "stat": true, "du": true, "df": true,
	"which": true, "type": true, "echo": true, "printf": true,
	"true": true, "false": true, "test": true, "[": true,
	"grep": true, "egrep": true, "fgrep": true, "rg": true,
	"find": true, "sort": true, "cut": true,
	"tr": true, "column": true, "comm": true, "diff": true, "cmp": true,
	"git": true, "go": true, "uname": true, "whoami": true,
	"id": true, "sw_vers": true, "tree": true,
	"hexdump": true, "od": true,
}

// Repository-configured diff/textconv/fsmonitor helpers are executable code.
// Keep ordinary git inspection available without allowing those callbacks to
// mutate files in a read-only session. The wrapper works in both bash and the
// POSIX shell used by sandbox backends. Validation allows only bare git calls.
const readOnlyGitPrelude = `git() {
  case "$1" in
    -C) (cd -- "$2" && shift 2 && git "$@") ;;
    --no-pager) shift; git "$@" ;;
    --literal-pathspecs) shift; GIT_LITERAL_PATHSPECS=1 git "$@" ;;
    diff) shift; command git --no-pager -c core.fsmonitor=false diff --no-ext-diff --no-textconv "$@" ;;
    show) shift; command git --no-pager -c core.fsmonitor=false show --no-ext-diff --no-textconv "$@" ;;
    log) shift; command git --no-pager -c core.fsmonitor=false log --no-ext-diff --no-textconv "$@" ;;
    blame) shift; command git --no-pager -c core.fsmonitor=false blame --no-textconv "$@" ;;
    stash) shift; case "$1" in
      show) shift; command git --no-pager -c core.fsmonitor=false stash show --no-ext-diff --no-textconv "$@" ;;
      list) shift; command git --no-pager -c core.fsmonitor=false stash list --no-ext-diff --no-textconv "$@" ;;
    esac ;;
    *) command git --no-pager -c core.fsmonitor=false "$@" ;;
  esac
}
`

// General interpreters, package scripts, pagers, and sed/awk programs can
// execute code or write files without shell redirects. Use native tools for
// those operations in read-only modes rather than guessing their effects.

// ExploreBashAllowed returns nil if command is safe for explore/plan modes.
// It is segment-aware: splits on ; && || | and checks each pipeline stage's
// primary command. Fail-closed on empty/unknown tokens, redirects, or
// mutating git/go subcommands.
func ExploreBashAllowed(command string) error {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return fmt.Errorf("empty command")
	}
	// Expansions can execute commands or turn data into shell syntax. These
	// modes accept literal inspection commands only.
	if strings.ContainsAny(cmd, "$`\\()") || strings.Contains(cmd, "<<") {
		return fmt.Errorf("explore bash: expansions, subshells and heredocs are not allowed")
	}
	// Hard deny obvious mutators / redirections even before segment parse.
	if strings.ContainsAny(cmd, "<>") {
		return fmt.Errorf("explore bash: redirects are not allowed")
	}
	segments := splitShellSegments(cmd)
	if len(segments) == 0 {
		return fmt.Errorf("explore bash: no executable segments")
	}
	for _, seg := range segments {
		if err := exploreSegmentAllowed(seg); err != nil {
			return err
		}
	}
	return nil
}

// splitShellSegments splits on top-level ; && || | (not inside quotes).
func splitShellSegments(s string) []string {
	var segs []string
	var b strings.Builder
	inSingle, inDouble := false, false
	escape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escape {
			b.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' && !inSingle {
			escape = true
			b.WriteByte(ch)
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			b.WriteByte(ch)
			continue
		}
		if !inSingle && !inDouble {
			// operators
			if ch == ';' || ch == '\n' {
				if t := strings.TrimSpace(b.String()); t != "" {
					segs = append(segs, t)
				}
				b.Reset()
				continue
			}
			if ch == '|' || ch == '&' {
				// || or && or |
				op := string(ch)
				if i+1 < len(s) && s[i+1] == ch {
					op = string(ch) + string(ch)
					i++
				}
				if t := strings.TrimSpace(b.String()); t != "" {
					segs = append(segs, t)
				}
				b.Reset()
				_ = op
				continue
			}
		}
		b.WriteByte(ch)
	}
	if t := strings.TrimSpace(b.String()); t != "" {
		segs = append(segs, t)
	}
	return segs
}

func exploreSegmentAllowed(seg string) error {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return nil
	}
	// strip env assignments: FOO=bar cmd
	tokens := tokenizeShellWords(seg)
	for len(tokens) > 0 && strings.Contains(tokens[0], "=") && !strings.HasPrefix(tokens[0], "=") {
		name, _, _ := strings.Cut(tokens[0], "=")
		for _, prefix := range []string{"PATH", "LD_", "DYLD_", "GIT_", "GO", "BASH", "ENV", "SHELL", "CDPATH"} {
			if strings.HasPrefix(name, prefix) {
				return fmt.Errorf("explore bash: execution environment override %q is not allowed", name)
			}
		}
		tokens = tokens[1:]
	}
	if len(tokens) == 0 {
		return fmt.Errorf("explore bash: empty segment after env strip")
	}
	if len(tokens) == 0 {
		return fmt.Errorf("explore bash: no command token")
	}
	base := commandBase(tokens[0])
	if tokens[0] != base || !exploreBashAllow[base] {
		return fmt.Errorf("explore bash: command %q is not on the read-only allowlist", base)
	}
	switch base {
	case "git":
		return exploreGitAllowed(tokens[1:])
	case "go":
		return exploreGoAllowed(tokens[1:])
	case "find":
		for _, t := range tokens[1:] {
			if t == "-delete" || strings.HasPrefix(t, "-exec") || strings.HasPrefix(t, "-ok") || strings.HasPrefix(t, "-fprint") || t == "-fls" {
				return fmt.Errorf("explore bash: find %s is not allowed", t)
			}
		}
	case "sort", "tree":
		for _, arg := range tokens[1:] {
			if strings.HasPrefix(arg, "--o") || (strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg, "o")) {
				return fmt.Errorf("explore bash: file output is not allowed")
			}
			if base == "sort" && strings.HasPrefix(arg, "--c") {
				return fmt.Errorf("explore bash: sort compression programs are not allowed")
			}
		}
	case "printf":
		for _, arg := range tokens[1:] {
			if strings.HasPrefix(arg, "-v") {
				return fmt.Errorf("explore bash: assigning shell variables is not allowed")
			}
		}
	case "rg":
		for _, arg := range tokens[1:] {
			if strings.HasPrefix(arg, "--pre") || strings.HasPrefix(arg, "--hostname-bin") || arg == "-z" || arg == "--search-zip" {
				return fmt.Errorf("explore bash: external search programs are not allowed")
			}
		}
	case "file":
		for _, arg := range tokens[1:] {
			if strings.HasPrefix(arg, "--c") || (strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && strings.Contains(arg, "C")) {
				return fmt.Errorf("explore bash: compiling magic files is not allowed")
			}
		}
	}
	return nil
}

func exploreGitAllowed(args []string) error {
	// Parse the small set of read-only global options before the subcommand.
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "-C":
			if len(args) < 2 {
				return fmt.Errorf("explore bash: git -C needs a directory")
			}
			args = args[2:]
		case "--no-pager", "--literal-pathspecs":
			args = args[1:]
		case "--version":
			if len(args) == 1 {
				return nil
			}
			return fmt.Errorf("explore bash: unexpected git version arguments")
		default:
			return fmt.Errorf("explore bash: git option %q is not allowed", args[0])
		}
	}
	if len(args) == 0 {
		return nil
	}
	sub, rest := args[0], args[1:]
	for _, arg := range rest {
		if strings.HasPrefix(arg, "--out") || strings.HasPrefix(arg, "--ext") || strings.HasPrefix(arg, "--textconv") || arg == "--exec" || strings.HasPrefix(arg, "--filt") {
			return fmt.Errorf("explore bash: git option %q can mutate or execute code", arg)
		}
	}
	switch sub {
	case "status", "log", "show", "diff", "rev-parse", "rev-list", "describe", "ls-files", "ls-tree", "cat-file", "blame", "shortlog", "name-rev", "for-each-ref", "version":
		return nil
	case "branch", "tag":
		if len(rest) == 0 {
			return nil
		}
		if rest[0] == "--list" || rest[0] == "-l" {
			for _, arg := range rest[1:] {
				if strings.HasPrefix(arg, "-") {
					return fmt.Errorf("explore bash: only listing patterns are allowed")
				}
			}
			return nil
		}
		for _, arg := range rest {
			switch arg {
			case "--show-current", "--all", "--remotes", "-a", "-r", "-v", "-vv":
			default:
				return fmt.Errorf("explore bash: mutating git %s arguments are not allowed", sub)
			}
		}
		return nil
	case "stash":
		if len(rest) > 0 && (rest[0] == "list" || rest[0] == "show") {
			return nil
		}
	case "remote":
		if len(rest) == 0 || (len(rest) == 1 && rest[0] == "-v") {
			return nil
		}
	case "config":
		// Read forms only; never allow bare key/value or edit/unset operations.
		if len(rest) > 0 {
			switch rest[0] {
			case "--get", "--get-all", "--get-regexp", "--get-urlmatch", "--list", "-l":
				for _, arg := range rest[1:] {
					if strings.HasPrefix(arg, "-") {
						return fmt.Errorf("explore bash: extra git config options are not allowed")
					}
				}
				return nil
			}
		}
	}
	return fmt.Errorf("explore bash: git %s is not a read-only operation", sub)
}

func exploreGoAllowed(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "version":
		if len(args) == 1 {
			return nil
		}
	case "env":
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") && arg != "-json" && arg != "-changed" {
				return fmt.Errorf("explore bash: go env option %q is not read-only", arg)
			}
		}
		return nil
	case "list":
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "-") && arg != "-json" && arg != "-deps" && arg != "-e" && arg != "-m" && arg != "-mod=readonly" {
				return fmt.Errorf("explore bash: go list option %q is not allowed", arg)
			}
		}
		return nil
	}
	return fmt.Errorf("explore bash: go %s is not a read-only operation", args[0])
}

func commandBase(token string) string {
	token = strings.Trim(token, `"'`)
	// strip path
	if i := strings.LastIndex(token, "/"); i >= 0 {
		token = token[i+1:]
	}
	return strings.ToLower(token)
}

// tokenizeShellWords is a minimal whitespace tokenizer that respects quotes.
func tokenizeShellWords(s string) []string {
	var out []string
	var b strings.Builder
	inSingle, inDouble := false, false
	escape := false
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escape {
			b.WriteByte(ch)
			escape = false
			continue
		}
		if ch == '\\' && !inSingle {
			escape = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble && (ch == ' ' || ch == '\t' || ch == '\n') {
			flush()
			continue
		}
		b.WriteByte(ch)
	}
	flush()
	return out
}
