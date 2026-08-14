#!/usr/bin/env bash

set -euo pipefail

readonly excludes_file=".github/govulncheck-excludes.txt"
readonly output_file="$(mktemp)"
trap 'rm -f "$output_file"' EXIT

status=0
govulncheck -show version ./... >"$output_file" 2>&1 || status=$?
cat "$output_file"

if [[ "$status" -eq 0 ]]; then
  exit 0
fi

findings=()
while IFS= read -r finding; do
  findings+=("$finding")
done < <(
  sed -n 's/^[[:space:]]*Vulnerability #[0-9][0-9]*: \(GO-[0-9-][0-9-]*\)$/\1/p' "$output_file" |
    sort -u
)

if [[ "${#findings[@]}" -eq 0 ]]; then
  echo "::error::govulncheck failed without reporting a vulnerability ID"
  exit "$status"
fi

unexpected=()
for finding in "${findings[@]}"; do
  if ! awk '!/^#/ && NF { print $1 }' "$excludes_file" | grep -Fxq "$finding"; then
    unexpected+=("$finding")
  fi
done

if [[ "${#unexpected[@]}" -gt 0 ]]; then
  echo "::error::Unapproved reachable vulnerabilities: ${unexpected[*]}"
  exit "$status"
fi

echo "::warning::Accepted govulncheck exception(s): ${findings[*]}"
