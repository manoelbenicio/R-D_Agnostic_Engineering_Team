# prodex pin and integrity verification

Status: F0 prep, documentation only. No deploy, release, publish, or redeem operation is part of this procedure.

This page documents the pre-launch verification lane for the pinned prodex source tree. It is based only on the official prodex repository at commit `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.

## Pin

- Requested pin: `v0.246.0`.
- Official prodex version at the pinned commit: `0.246.0`.
- Official prodex commit: `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.
- Official tag observed at that commit: `0.246.0`.
- `a validar`: an official `v0.246.0` tag alias was not confirmed in the audited checkout. Treat `0.246.0` plus the exact commit SHA as the source-of-truth pin unless an upstream signed or release reference confirms the `v` prefix.

## Source checkout verification

Run these checks against a fresh official prodex checkout before launch:

```bash
git fetch --tags origin
git checkout 7750da9b6a5c91a6d429e18e6a4d422cab4bc144
git rev-parse HEAD
git tag --points-at HEAD
```

Required results:

- `git rev-parse HEAD` returns `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.
- `git tag --points-at HEAD` includes `0.246.0`.
- `Cargo.toml` has `[package] version = "0.246.0"` and `license = "Apache-2.0"`.
- `Cargo.toml` has `[workspace.package] version = "0.246.0"` and `license = "Apache-2.0"`.
- `npm/prodex/package.json`, `npm/prodex-gateway-sdk/package.json`, and `npm/platforms/*/package.json` have `version = "0.246.0"` where applicable.
- The npm package repository fields point to `https://github.com/christiandoxa/prodex.git`.

Any mismatch blocks launch.

## Official release and CI gates

The audited prodex workflow confirms these gates:

- `.github/workflows/npm-publish.yml` publishes only on tags matching `0.*.*` or manual workflow dispatch, and the release ref must match the Cargo version when running from a tag.
- `.github/workflows/npm-publish.yml` `verify-ci` waits for `ci.yml` success for the target SHA before release build.
- `.github/workflows/npm-publish.yml` builds target binaries with `cargo build --release --locked --target ...` or `cross build --release --locked --target ...`.
- `.github/workflows/npm-publish.yml` runs native `prodex --version` smoke tests where supported.
- `.github/workflows/npm-publish.yml` attests each binary with `actions/attest-build-provenance@v4`.
- `.github/workflows/npm-publish.yml` generates `release-sbom.spdx.json` with `anchore/syft:v1.41.2` over the staged release tree.
- `.github/workflows/npm-publish.yml` publishes npm packages through `node scripts/npm/publish.mjs --root release --provenance`.
- `.github/workflows/npm-publish.yml` uploads the GitHub release assets and includes `release-sbom.spdx.json`.
- `.github/workflows/ci.yml` runs gitleaks with `ghcr.io/gitleaks/gitleaks:v8.30.1 detect --source /repo --no-git --redact --no-banner --config /repo/.gitleaks.toml`.
- `.github/workflows/ci.yml` installs `cargo-deny --version 0.19.0` and runs `cargo deny check advisories sources`.
- `.github/workflows/ci.yml` installs `cargo-audit --version 0.22.1` and runs `cargo audit`.

## Local pre-launch procedure

Use this as the local pre-launch verification checklist. Commands that mirror official workflow behavior are confirmed; commands marked `a validar` are launch controls that were not confirmed as first-class upstream workflow steps in the audited checkout.

1. Verify the source pin:

   ```bash
   git rev-parse HEAD
   git tag --points-at HEAD
   ```

2. Verify version and license metadata:

   ```bash
   rg -n '^(version|license) = "0\.246\.0"|license = "Apache-2\.0"' Cargo.toml
   find npm -maxdepth 3 -name package.json -print
   ```

3. Run release-prep guards without publishing:

   ```bash
   npm run release:prepare -- --dry-run
   npm run ci:preflight
   ```

   `release:prepare` is official and checks Cargo/npm/docs version sync, lockfiles, changelog freshness, docs lint, upstream Codex compatibility baseline, runtime test manifest, Cargo workspace publish order, cargo fmt, and cargo test or cargo check mode.

4. Run Rust supply-chain checks:

   ```bash
   cargo audit
   cargo deny check advisories sources
   ```

   The official `deny.toml` denies yanked crates, unknown registries, and unknown git sources.

5. Run secret scanning:

   ```bash
   docker run --rm -v "$PWD:/repo:ro" ghcr.io/gitleaks/gitleaks:v8.30.1 \
     detect --source /repo --no-git --redact --no-banner --config /repo/.gitleaks.toml
   ```

6. Build with locked dependencies:

   ```bash
   cargo build --release --locked --target x86_64-unknown-linux-gnu
   ```

   For the full release matrix, mirror the targets in `.github/workflows/npm-publish.yml`.

7. Verify binary provenance:

   - Confirm the release workflow produced `actions/attest-build-provenance@v4` attestations for each uploaded binary.
   - `a validar`: the exact attestation retrieval and verification command is not specified in the audited prodex repository. Use the hosting platform's official attestation verification path for the published release assets and require the subject to match the downloaded binary.

8. Verify SBOM:

   - Confirm the release includes `release-sbom.spdx.json`.
   - Confirm it was generated from the staged `release/` tree by `anchore/syft:v1.41.2` in SPDX JSON format.
   - `a validar`: the audited workflow uploads the SBOM but does not show a local SBOM verification command. Require local review or automated policy validation before launch.

9. Verify checksums:

   - `a validar`: no official checksum generation or `SHA256SUMS` publication step was found in the audited prodex workflow.
   - If checksums are provided by the release channel, verify each downloaded release asset against that upstream checksum before launch.
   - If no upstream checksum is provided, generate and record local SHA-256 values for the exact approved release assets and treat them as deployment evidence, not upstream proof:

     ```bash
     sha256sum release-assets/* > SHA256SUMS.local
     ```

10. Smoke the packaged npm tree:

    ```bash
    node scripts/ci/npm-package-smoke.mjs --binary-dir target/x86_64-unknown-linux-gnu/release
    ```

    The official smoke script stages a local package tree and requires `prodex --version` output to include `0.246.0`.

## Launch block conditions

Block launch if any condition is true:

- The checkout SHA is not `7750da9b6a5c91a6d429e18e6a4d422cab4bc144`.
- The tag at the checkout does not include official tag `0.246.0`.
- Any package or workspace version differs from `0.246.0`.
- Any official CI, release-prep, gitleaks, cargo-audit, or cargo-deny gate fails.
- Binary provenance attestation cannot be verified.
- The SBOM is missing or cannot be tied to the staged release tree.
- `a validar`: published checksums are missing or cannot be independently verified; if no official checksum exists, require recorded local checksum evidence before deployment approval.

## Official evidence

- `Cargo.toml`: package/workspace version, license, workspace crate version pins.
- `npm/prodex/package.json`: main npm package version and optional platform package version pins.
- `npm/prodex-gateway-sdk/package.json`: gateway SDK package version.
- `npm/platforms/*/package.json`: platform package versions and repository metadata.
- `deny.toml`: cargo-deny advisory/source policy.
- `.gitleaks.toml`: gitleaks configuration.
- `.github/workflows/ci.yml`: gitleaks, cargo-audit, cargo-deny, clippy, package-smoke, and other CI gates.
- `.github/workflows/npm-publish.yml`: release ref verification, locked target builds, binary provenance attestation, SBOM generation/upload, npm provenance publish, and GitHub release asset upload.
- `scripts/npm/release-prepare.mjs`: release preparation guard scope.
- `scripts/npm/stage.mjs`: staged npm package assembly.
- `scripts/npm/publish.mjs`: npm publish order and `--provenance` support.
- `scripts/ci/npm-package-smoke.mjs`: local package smoke behavior.
