#!/usr/bin/env bash
# SOPS + age secrets helper. Encrypted files (deploy/secrets/*.sops.yaml) are committed; the age
# private key (deploy/secrets/age.key) is NOT — it comes from the team vault / CI. See
# deploy/secrets/README.md and the repo-root .sops.yaml.
#
#   scripts/secrets.sh encrypt <file.sops.yaml>   # encrypt in place (uses .sops.yaml creation_rules)
#   scripts/secrets.sh decrypt <file.sops.yaml>   # decrypt to stdout (does NOT write plaintext to disk)
#   scripts/secrets.sh edit    <file.sops.yaml>   # open decrypted in $EDITOR, re-encrypt on save
#   scripts/secrets.sh apply   <file.sops.yaml>   # decrypt + kubectl apply (never writes plaintext)
set -euo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"
# The private key: explicit env wins; else the gitignored dev key in the repo.
export SOPS_AGE_KEY_FILE="${SOPS_AGE_KEY_FILE:-$HERE/deploy/secrets/age.key}"

cmd="${1:-}"; file="${2:-}"
[ -z "$cmd" ] || [ -z "$file" ] && { echo "usage: $0 {encrypt|decrypt|edit|apply} <file>"; exit 2; }

case "$cmd" in
  encrypt) sops --encrypt --in-place "$file" ;;
  decrypt) sops --decrypt "$file" ;;
  edit)    sops "$file" ;;
  apply)   sops --decrypt "$file" | kubectl apply -f - ;;
  *)       echo "unknown command: $cmd"; echo "usage: $0 {encrypt|decrypt|edit|apply} <file>"; exit 2 ;;
esac
