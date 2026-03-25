#!/usr/bin/env bash
# Merge a GitHub PR by number. Uses --merge (merge commit) so it works non-interactively.
# Usage: ./scripts/merge-pr.sh <pr-number>
#    or: make pr-merge PR=<pr-number>

set -e

if [[ -z "${1:-}" ]]; then
  echo "Usage: $0 <pr-number>" >&2
  echo "Example: $0 91" >&2
  exit 1
fi

gh pr merge "$1" --merge
