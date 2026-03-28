---
name: release-prep
description: Use when preparing a release, checking what needs releasing, determining version bumps, auditing unreleased changes across engines and client SDKs, or when the user asks "what needs to be released" or "do we need a release".
---

# Release Prep

Audit unreleased changes across engines and client SDKs, determine version bumps, and identify what needs releasing.

## Overview

This monorepo has three release layers that must be released in order:
1. **Engines** (Rust crates) - must be released first, SDKs depend on them
2. **FFI-based SDKs** - depend on `flipt-engine-ffi` binaries published to GitHub releases
3. **WASM/WASM-JS SDKs** - depend on `flipt-engine-wasm` / `flipt-engine-wasm-js` artifacts

## Step 1: Audit Engine Changes

### Find last engine release tags

Each engine has its own tag namespace:

```bash
# FFI engine (triggers: package-ffi-engine-{darwin,linux,windows,android,ios}.yml)
git tag --sort=-creatordate | grep "flipt-engine-ffi-v" | head -5

# WASM engine (triggers: package-wasm-engine.yml)
git tag --sort=-creatordate | grep "flipt-engine-wasm-v" | head -5

# WASM-JS engine (triggers: package-wasm-js-engine.yml)
git tag --sort=-creatordate | grep "flipt-engine-wasm-js-v" | head -5
```

### List unreleased engine commits

```bash
# Changes to engine code since last FFI release
git log --oneline <last-ffi-tag>..HEAD --no-merges -- flipt-evaluation/ flipt-engine-ffi/

# Changes to WASM engine since last WASM release
git log --oneline <last-wasm-tag>..HEAD --no-merges -- flipt-evaluation/ flipt-engine-wasm/

# Changes to WASM-JS engine since last WASM-JS release
git log --oneline <last-wasm-js-tag>..HEAD --no-merges -- flipt-evaluation/ flipt-engine-wasm-js/
```

### Check current versions

```bash
grep '^version' flipt-evaluation/Cargo.toml flipt-engine-ffi/Cargo.toml flipt-engine-wasm/Cargo.toml flipt-engine-wasm-js/Cargo.toml
```

### Determine version bumps (semver)

- **fix** only -> patch bump
- **feat** or new capability -> minor bump
- Breaking API change -> major bump

Only bump crates that have actual changes. `flipt-evaluation` only needs a bump if commits touched `flipt-evaluation/` directly.

## Step 2: Audit Client SDK Changes

For each SDK, find its latest release tag and list unreleased commits:

```bash
git tag --sort=-creatordate | grep "flipt-client-" | head -30
git log --oneline <last-sdk-tag>..HEAD --no-merges -- <sdk-directory>/
```

### SDK reference table

| SDK | Directory | Tag Prefix | Engine Dependency |
|-----|-----------|------------|-------------------|
| Python | `flipt-client-python/` | `flipt-client-python-v` | FFI |
| Ruby | `flipt-client-ruby/` | `flipt-client-ruby-v` | FFI |
| Java | `flipt-client-java/` | `flipt-client-java-v` | FFI |
| C# | `flipt-client-csharp/` | `flipt-client-csharp-v` | FFI |
| Dart | `flipt-client-dart/` | `flipt-client-dart-v` | FFI |
| Kotlin Android | `flipt-client-kotlin-android/` | `flipt-client-kotlin-android-v` | FFI |
| Swift | `flipt-client-swift/` | `flipt-client-swift-v` | FFI |
| Go | `flipt-client-go/` | `flipt-client-go-v` | WASM |
| JS | `flipt-client-js/` | `flipt-client-js-v` | WASM-JS |
| React | `flipt-client-react/` | `flipt-client-react-v` | WASM-JS |

## Step 3: Categorize and Prioritize

Present a summary table classifying each SDK:

- **Must release**: Has feat/fix commits in its own directory (someone contributed a fix and it's not released yet)
- **Should release**: Benefits from engine bump + has dep updates
- **No release needed**: No changes since last release, no engine bump affects it

### Output format

```
## Engine Release Status
| Engine | Current Version | Last Tag | Unreleased Commits | Bump Needed |
|--------|----------------|----------|-------------------|-------------|

## SDK Release Status
| SDK | Last Release | Unreleased Commits | Key Changes | Priority |
|-----|-------------|-------------------|-------------|----------|
```

## Step 4: Release Engines (if needed)

Engine release is a three-step process per RELEASE.md:

1. **Bump version** in `Cargo.toml` for affected crates
2. Run `cargo check` to verify compilation and update `Cargo.lock`
3. **Create a PR** with the version bump, merge it
4. **Create and push tags** from main after merge:
   - `flipt-engine-ffi-v{version}` — triggers CI to build and publish FFI binaries to GitHub release
   - `flipt-engine-wasm-v{version}` — for bookkeeping (WASM is bundled with its SDKs at SDK release time)
   - `flipt-engine-wasm-js-v{version}` — for bookkeeping (WASM-JS is bundled with its SDKs at SDK release time)

**Only FFI-based SDKs need to wait for engine CI** — the FFI tag triggers builds that publish platform binaries. WASM/WASM-JS engines are built and bundled during SDK release, so their SDKs can be released immediately after tagging.

## Step 5: Release Client SDKs

Use the non-interactive CLI script (preferred for automation):

```bash
cd release && .venv/bin/python release_cli.py --sdk flipt-client-go --bump patch             # dry-run (default)
cd release && .venv/bin/python release_cli.py --sdk flipt-client-go --bump patch --publish   # real release
cd release && .venv/bin/python release_cli.py --sdk flipt-client-js --bump patch --publish --pr
```

If the venv doesn't exist yet: `cd release && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt`

Options:
- `--sdk NAME`: SDK to release (required)
- `--bump patch|minor|major`: Version bump type
- `--version X.Y.Z`: Explicit version instead of bump
- `--publish`: Actually release (default is dry-run)
- `--pr`: Create a PR instead of pushing to main

The interactive version (`python release.py`) is also available for manual use.

Release order doesn't matter between SDKs, but FFI-based SDKs must wait for engine artifacts to be published.
