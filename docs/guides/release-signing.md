# Release Signing Keys

Nexus release updates require a signed `release-manifest.json`. CI fails fast if signing secrets are not configured.

## Required GitHub Actions secrets

- `NEXUS_UPDATE_MANIFEST_PRIVATE_KEY_B64`
- `NEXUS_UPDATE_MANIFEST_PUBLIC_KEY_B64`

The private key signs the release manifest in CI. The public key is embedded into released binaries and used by clients to verify signatures.

## Create keys (Ed25519)

Use Python:

```bash
python3 -m pip install cryptography
python3 - <<'PY'
import base64
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

priv = Ed25519PrivateKey.generate()
pub = priv.public_key()

print("PRIVATE_B64=" + base64.b64encode(priv.private_bytes_raw()).decode())
print("PUBLIC_B64=" + base64.b64encode(pub.public_bytes_raw()).decode())
PY
```

Add `PRIVATE_B64` to `NEXUS_UPDATE_MANIFEST_PRIVATE_KEY_B64` and `PUBLIC_B64` to `NEXUS_UPDATE_MANIFEST_PUBLIC_KEY_B64` in repository secrets.

## Verify setup locally

1. Build release artifacts into a local folder.
2. Export `NEXUS_UPDATE_MANIFEST_PRIVATE_KEY_B64`.
3. Run:

```bash
python3 scripts/release/generate_release_manifest.py \
  --version 0.0.0-test \
  --artifacts-dir /path/to/artifacts \
  --base-url https://example.invalid/download \
  --manifest-out /path/to/artifacts/release-manifest.json \
  --signature-out /path/to/artifacts/release-manifest.sig
```

If key setup is correct, both files are generated.

## Rotation policy

When rotating keys:

1. Generate a new key pair.
2. Update both secrets in GitHub Actions.
3. Cut a new release so binaries embed the new public key.
4. Keep old release artifacts unchanged; they continue to verify with the old embedded key in old binaries.

## Recovery if private key is lost

1. Generate a new key pair.
2. Update both secrets.
3. Release immediately so all new installers/clients carry the new public key.
4. Announce key rotation in release notes.

## How popular projects handle this

Common patterns in production projects:

- **Detached signatures + checksums** for release artifacts and metadata.
- **Immutable public verification key embedded in client**.
- **CI-only private key access** via secret manager.
- **Fail-closed updater behavior** if signature verification fails.
- **Key rotation with overlap/migration planning** for long-lived clients.

Nexus follows this same model with a signed release manifest plus per-artifact checksums.
