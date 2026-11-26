#!/usr/bin/env bash
# Create test users in a Snapend through the Auth snap's anonymous login
# (PUT /v1/auth/login/anon with create_user=true), so a job has users to
# run over. Every run creates new users: usernames are <prefix>_<n> with a
# prefix unique to the run (byojob_<timestamp>).
#
# Usage:
#   ./seed_users.sh <snapend_id> [count]
#
# Examples:
#   ./seed_users.sh abc12345              # 100 new users
#   ./seed_users.sh abc12345 500          # 500 new users
#   PREFIX=byojob_test ./seed_users.sh abc12345 50   # fixed names; reruns log in
#
# Env overrides:
#   GATEWAY       default "https://gateway.snapser.com"
#   PREFIX        default "byojob_<yyyymmddhhmmss>"; usernames are ${PREFIX}_<n>.
#                 A fixed PREFIX makes reruns log in the same users.
#   START         default 1; first <n>
#   CONCURRENCY   default 10 parallel requests
#
# The Snapend must have the Auth snap with anonymous login enabled.
# Prereqs: curl, xargs.

set -euo pipefail

usage() {
  sed -n '2,/^set -e/p' "$0" | sed -n '/^#/p' | sed 's/^# \{0,1\}//'
  exit "${1:-1}"
}

[[ "${1:-}" == "-h" || "${1:-}" == "--help" ]] && usage 0
[[ $# -ge 1 && $# -le 2 ]] || usage 1

SNAPEND_ID="$1"
COUNT="${2:-100}"
GATEWAY="${GATEWAY:-https://gateway.snapser.com}"
PREFIX="${PREFIX:-byojob_$(date +%Y%m%d%H%M%S)}"
START="${START:-1}"
CONCURRENCY="${CONCURRENCY:-10}"

[[ "$COUNT" =~ ^[0-9]+$ && "$COUNT" -gt 0 ]] || { echo "count must be a positive number" >&2; exit 1; }
[[ "$PREFIX" =~ ^[A-Za-z0-9_-]{1,40}$ ]] || { echo "PREFIX must be 1-40 of [A-Za-z0-9_-]" >&2; exit 1; }

URL="${GATEWAY%/}/${SNAPEND_ID}/v1/auth/login/anon"
END=$((START + COUNT - 1))
RESULTS="$(mktemp)"
trap 'rm -f "$RESULTS"' EXIT

echo "==> Creating ${COUNT} users (${PREFIX}_${START} .. ${PREFIX}_${END})"
echo "    ${URL}, ${CONCURRENCY} at a time"

# Each request prints "<http code> <username>" on one line.
export URL PREFIX
seq "$START" "$END" | xargs -P "$CONCURRENCY" -I{} sh -c '
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 30 --retry 2 \
    -X PUT -H "Content-Type: application/json" \
    -d "{\"create_user\":true,\"username\":\"${PREFIX}_{}\"}" "$URL") || code=000
  echo "$code ${PREFIX}_{}"
' | tee "$RESULTS" | awk -v total="$COUNT" '
  { n++ } n % 50 == 0 { printf "    %d / %d\n", n, total }'

ok=$(grep -c '^200 ' "$RESULTS" || true)
failed=$((COUNT - ok))
echo "==> Done: ${ok} ok, ${failed} failed"
if [[ "$failed" -gt 0 ]]; then
  echo "    Failures by status:"
  grep -v '^200 ' "$RESULTS" | cut -d' ' -f1 | sort | uniq -c | sed 's/^/      /'
  echo "    First failures:"
  grep -v '^200 ' "$RESULTS" | head -5 | sed 's/^/      /'
  exit 1
fi
