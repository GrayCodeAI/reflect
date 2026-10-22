#!/usr/bin/env bash
# Enforce the repository commit policy:
#   1. Every commit in the change set has a verified signature.
#   2. No commit credits an AI assistant as a co-author or generator.
#
# Works for pull_request and push events. Requires GH_TOKEN (the workflow's
# GITHUB_TOKEN is enough) and GITHUB_REPOSITORY.
set -euo pipefail

repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
event="${GITHUB_EVENT_NAME:?GITHUB_EVENT_NAME is required}"

ai_author_re='^[[:space:]]*co-authored-by:.*(claude|anthropic|openai|chatgpt|gpt|copilot|cursor|gemini|codex|aider|devin|windsurf|tabnine|cody|replit|amazon[ -]?q|jetbrains[ -]?ai|sourcegraph)'
ai_generated_re='generated (with|by).*(claude|anthropic|openai|chatgpt|gpt|copilot|cursor|gemini|codex|aider|devin|windsurf|tabnine|cody|replit|amazon[ -]?q)'

commit_shas() {
    case "$event" in
        pull_request | pull_request_target)
            gh api --paginate "repos/$repo/pulls/${PR_NUMBER:?PR_NUMBER is required}/commits" --jq '.[].sha'
            ;;
        push)
            local before="${PUSH_BEFORE:-}" after="${PUSH_AFTER:?PUSH_AFTER is required}"
            if [[ -z "$before" || "$before" =~ ^0+$ ]]; then
                # New branch: check the pushed head only; its history predates
                # this policy.
                printf '%s\n' "$after"
            else
                gh api "repos/$repo/compare/${before}...${after}" --jq '.commits[].sha'
            fi
            ;;
        *)
            echo "commit policy: unsupported event '$event'" >&2
            exit 2
            ;;
    esac
}

failed=0
checked=0

while IFS= read -r sha; do
    [[ -z "$sha" ]] && continue
    checked=$((checked + 1))

    commit_json=$(gh api "repos/$repo/commits/$sha")
    verified=$(jq -r '.commit.verification.verified' <<<"$commit_json")
    message=$(jq -r '.commit.message' <<<"$commit_json")

    if [[ "$verified" != "true" ]]; then
        echo "::error::Commit ${sha:0:12} is not signed with a verified key."
        failed=1
    fi

    if grep -Eiq "$ai_author_re" <<<"$message"; then
        echo "::error::Commit ${sha:0:12} credits an AI assistant as co-author."
        grep -Ein "$ai_author_re" <<<"$message" | sed 's/^/    /'
        failed=1
    fi

    if grep -Eiq "$ai_generated_re" <<<"$message"; then
        echo "::error::Commit ${sha:0:12} contains an AI generation attribution."
        grep -Ein "$ai_generated_re" <<<"$message" | sed 's/^/    /'
        failed=1
    fi
done < <(commit_shas)

if [[ "$checked" -eq 0 ]]; then
    echo "commit policy: no commits to check"
    exit 0
fi

if [[ "$failed" -ne 0 ]]; then
    echo
    echo "Commit policy failed."
    echo "  - Sign commits with a key registered on GitHub (git commit -S)."
    echo "  - Remove AI co-author / generation trailers from the message."
    exit 1
fi

echo "commit policy: $checked commit(s) verified, no AI co-author trailers"
