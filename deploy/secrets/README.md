# Secrets (SOPS + age)

Secrets live in this repo **encrypted**, so they're versioned and reviewable without ever exposing the
material. We use [SOPS](https://getsops.io) with [age](https://age-encryption.org): SOPS encrypts only
the values of a Kubernetes Secret (`data` / `stringData`), leaving the manifest structure in cleartext
so diffs stay meaningful.

## The rule

- **Committed:** `*.sops.yaml` (values encrypted), `../../.sops.yaml` (recipients + rules), this README,
  and `*.example.yaml` plaintext templates (no real values).
- **Never committed:** the age **private key** (`age.key`) and any **decrypted** plaintext. Both are
  gitignored. The private key comes from the team vault / CI secret store; in prod the recipient is an
  age key whose private half lives in KMS/Vault.

## Files

| File | What |
|---|---|
| `../../.sops.yaml` | Creation rules: which paths are encrypted, for which age recipients, and that only `data`/`stringData` are encrypted. |
| `platform-auth.example.yaml` | Plaintext **template** of the shared `platform-auth` Secret (`PLATFORM_JWT_SECRET`, `SERVICE_TOKEN`). No real values. |
| `platform-auth.sops.yaml` | The real Secret, **encrypted**. Committed. Decrypt needs the age key. |
| `age.key` | The age **private** key. **Gitignored** — obtain from the team vault. |

## One-time setup (per operator)

```bash
# Generate your own age key, or obtain the shared dev key from the vault, and place it here:
age-keygen -o deploy/secrets/age.key            # prints the PUBLIC recipient (age1...)
# Add your public recipient to ../../.sops.yaml under `age:`, then re-key existing files:
SOPS_AGE_KEY_FILE=deploy/secrets/age.key sops updatekeys deploy/secrets/*.sops.yaml
```

## Day-to-day

```bash
# Edit a secret (decrypts into $EDITOR, re-encrypts on save):
make secrets-edit                 # or: scripts/secrets.sh edit deploy/secrets/platform-auth.sops.yaml

# Apply to the cluster (decrypt → kubectl apply; never writes plaintext to disk):
make secrets-apply                # or: scripts/secrets.sh apply deploy/secrets/platform-auth.sops.yaml

# Create a brand-new secret from the template:
cp deploy/secrets/platform-auth.example.yaml deploy/secrets/<name>.sops.yaml
# ...fill real values (e.g. openssl rand -base64 48), then:
scripts/secrets.sh encrypt deploy/secrets/<name>.sops.yaml
```

`scripts/secrets.sh` resolves the private key from `$SOPS_AGE_KEY_FILE`, falling back to
`deploy/secrets/age.key`.

## Before a real-money (prod) deploy

1. Generate a **prod** age key; store the private half in KMS/Vault (never in git).
2. Add its public recipient to `.sops.yaml` and `sops updatekeys` every `*.sops.yaml`.
3. **Rotate** `PLATFORM_JWT_SECRET` and `SERVICE_TOKEN` to fresh prod values (the committed dev values
   are for the local cluster only) and re-encrypt.
4. CI decrypts with the prod key at deploy time (`make secrets-apply`).
