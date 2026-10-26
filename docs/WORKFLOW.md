# Resolve → Reflect → Record

`reflect work` is the first local GrayCode factory path. Resolve owns the
ticket, Reflect owns the coding run, Relay supplies the model, and Record owns
the immutable checkpoint linked to the resulting commit.

## Run a ticket

From a Reflect source checkout with sibling Resolve and Record checkouts:

```bash
go build -modfile=go.local.mod -o reflect ./cmd/reflect
(cd ../resolve && go build -o resolve ./cmd/resolve)
(cd ../record && go build -o record ./cmd/record)
./reflect work <ticket-id> --repo /path/to/project \
  --resolve-bin ../resolve/resolve --record-bin ../record/record \
  --verify 'go test ./...'
```

Create a ticket in the project first with `resolve new --repo /path/to/project
--title '...' --body '...'`. The target repository must have a clean working
tree and a valid Git commit. The verification command is supplied explicitly;
Resolve's current ticket schema does not yet have a verification field.

The work command creates a branch named `reflect/<ticket-id>-<suffix>` and a
separate worktree under the repository parent's `.reflect-worktrees` directory.
It claims the ticket, invokes `reflect exec --json` in that
worktree, stages and commits the agent's changes, then runs the verification
command against that exact commit. It requires a clean worktree after the check
and creates a Record checkpoint with the Reflect
session ID and verification result, then comments on the Resolve ticket and
moves it to `in_review`.

The branch and worktree remain available for inspection. The output includes
their paths, the exact commit, and the checkpoint ID. The default leaves all
work local. Pass `--pr` to push the branch to `origin` and open a draft GitHub
PR through `gh`; `--base` selects the target branch when it differs from the
branch where the ticket run started. Review and merge remain manual.

## Failure and permissions

If Reflect, verification, commit, or Record fails, the worktree remains for
inspection. A failed verification may leave an unverified commit, but creates no
checkpoint or review-ready ticket status.
Resolve remains `in_progress` until a verified checkpoint is created. After a
late failure, inspect the worktree and complete the handoff manually; automatic
resume and retry are future work.

The default `--auto semi` permits agent reads and file writes but denies
unattended Bash requests. `--auto full` allows more shell work, subject to
Reflect's normal permission checks. The `--verify` command is an operator
supplied shell command executed in the worktree; use a command you trust.

Record currently stores summary and verification metadata locally under the
repository's Git common directory. It does not capture transcripts or sync
checkpoints to remotes. Resolve events can sync through its own `sync` command.
