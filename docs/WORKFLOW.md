# Request → Resolve → Recall → Reflect/Relay → Record → Rally

`reflect work` accepts a plain request. It creates a Resolve work item for
tracking, reads project context from Recall, runs the coding agent through
Relay, verifies the result, records a checkpoint, saves a sourced outcome in
Recall, and posts progress to Rally. You can also pass an existing Resolve ID.

## Build from a request

From a Reflect source checkout with sibling product checkouts:

```bash
go build -modfile=go.local.mod -o reflect ./cmd/reflect
(cd ../resolve && go build -o resolve ./cmd/resolve)
(cd ../record && go build -o record ./cmd/record)
(cd ../recall && go build -o recall ./cmd/recall)
(cd ../rally && go build -o rally ./cmd/rally)
(cd ../rivet && go build -o rivet ./cmd/rivet)
./reflect work --request 'Build the requested feature' --repo /path/to/project \
  --resolve-bin ../resolve/resolve --record-bin ../record/record \
  --recall-bin ../recall/recall --rally-bin ../rally/rally \
  --rivet-bin ../rivet/rivet --verify 'go test ./...'
```

The target repository must have a clean working tree and a valid Git commit.
The verification command is supplied explicitly. Passing `--rivet-bin` runs it
through Rivet, which prints a compact result and retains the raw output in a
private log outside the worktree. The Record checkpoint cites that log. You can
instead pass `--verify-block NAME` to verify and run a Rivet block committed in
the target repository; Reflect finds `rivet` on `PATH` unless `--rivet-bin` is
given. A Resolve item is created
automatically to correlate the run; no ticket setup is needed. To use an
existing item, replace `--request '...'` with its ID. The default Recall and
Rally stores are under the user's config directory; use `--recall-store` and
`--rally-data` to select other local stores.

The work command creates a branch named `reflect/<ticket-id>-<suffix>` and a
separate worktree under the repository parent's `.reflect-worktrees` directory.
It claims the work item, posts a Rally start note, invokes `reflect exec --json` in that
worktree, stages and commits the agent's changes, then runs the verification
command against that exact commit. It requires a clean worktree after the check
and creates a Record checkpoint with the Reflect session ID and verification
result. A Recall outcome cites the Git commit, which remains resolvable from
another clone once the branch is pushed; the local checkpoint ID remains in
the outcome text. With `--pr`, the draft PR also includes the verification
result. Reflect then comments on the
Resolve item, moves it to `in_review`, and posts a final Rally note.

The branch and worktree remain available for inspection. The output includes
their paths, the exact commit, the checkpoint ID, and the memory ID. The default leaves all
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
