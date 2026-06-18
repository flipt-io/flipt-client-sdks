# Agent PR Review Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one safe automated agent review workflow for pull requests in `flipt-client-sdks`.

**Architecture:** Use the central `flipt-io/agents/actions/pr-review` composite action and keep all repo-specific behavior in local `.agents/` Markdown overrides. The workflow runs from the trusted base branch with `pull_request_target`, never checks out PR head code, and posts one combined review.

**Tech Stack:** GitHub Actions, Flipt Agents PR review action, Cloudflare Workers AI model `cloudflare-workers-ai/@cf/moonshotai/kimi-k2.7-code`, Markdown reviewer prompts/personas.

---

## File Structure

- Create `.github/workflows/pr-review.yml`: PR review workflow, permissions, Dependabot skip, Cloudflare model configuration, security comments.
- Create `.agents/prompts/00-sdk-review-routing.md`: repo-level review routing and priorities.
- Create `.agents/personas/README.md`: persona index and conventions.
- Create `.agents/personas/rust-engine-reviewer.md`: Rust/evaluation/engine review lens.
- Create `.agents/personas/ffi-sdk-reviewer.md`: FFI SDK review lens.
- Create `.agents/personas/wasm-sdk-reviewer.md`: WASM SDK review lens.

No runtime tests are needed because this adds CI config and prompt files only. Use static checks: `git diff --check` and targeted file inspection.

---

### Task 1: Add PR Review Workflow

**Files:**
- Create: `.github/workflows/pr-review.yml`

- [ ] **Step 1: Create the workflow file**

Write `.github/workflows/pr-review.yml` with this exact content:

```yaml
name: PR Review

# AI code review for PRs targeting main, including PRs from forks.
#
# Uses `pull_request_target` (not `pull_request`) on purpose: a `pull_request`
# event from a fork gets a read-only token with no repo secrets, so the
# reviewer couldn't reach Cloudflare Workers AI or post a review. With
# `pull_request_target` the job runs in the base-repo context with a writable
# token and access to CLOUDFLARE_* secrets.
#
# SECURITY: this is safe ONLY because the reviewer never checks out or executes
# the PR's code — it reads the diff via the `gh` API. Do NOT add
# `ref: ${{ github.event.pull_request.head.* }}` to the checkout, and do not run
# any build steps, scripts, or Make targets from the PR in this workflow.

on:
  pull_request_target:
    branches: [main]
    types: [opened, synchronize, reopened]

permissions:
  contents: read
  pull-requests: write

concurrency:
  group: pr-review-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  review:
    # Skip Dependabot (it runs in a restricted context and we don't review it).
    if: ${{ github.event.pull_request.user.login != 'dependabot[bot]' }}
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      # Default ref is the base branch (main), NOT the PR head — keep it that way.
      - uses: actions/checkout@v6

      - uses: flipt-io/agents/actions/pr-review@main # pin to a tag/SHA to freeze the reviewer
        env:
          CLOUDFLARE_API_KEY: ${{ secrets.CLOUDFLARE_API_KEY }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
        with:
          pr-number: ${{ github.event.pull_request.number }}
          model: cloudflare-workers-ai/@cf/moonshotai/kimi-k2.7-code
```

- [ ] **Step 2: Verify key workflow safety properties**

Run:

```bash
grep -n "pull_request_target\|branches: \[main\]\|dependabot\|actions/checkout@v6\|kimi-k2.7-code" .github/workflows/pr-review.yml
```

Expected output includes lines for:

```text
pull_request_target:
branches: [main]
dependabot[bot]
actions/checkout@v6
cloudflare-workers-ai/@cf/moonshotai/kimi-k2.7-code
```

- [ ] **Step 3: Commit the workflow**

Run:

```bash
git add .github/workflows/pr-review.yml
git commit -m "ci: add agent pr review workflow"
```

---

### Task 2: Add Repo Review Routing Prompt

**Files:**
- Create: `.agents/prompts/00-sdk-review-routing.md`

- [ ] **Step 1: Create prompt directory and file**

Run:

```bash
mkdir -p .agents/prompts
cat > .agents/prompts/00-sdk-review-routing.md <<'PROMPT'
# Flipt Client SDK PR Review Routing

This repository contains Flipt client SDKs and their shared Rust evaluation engines.
Produce **one combined PR review** for the pull request. Do not post separate reviews
per language or package.

## Changed-path routing

Apply these local personas when their paths are touched:

- `.agents/personas/rust-engine-reviewer.md` for `flipt-evaluation/`,
  `flipt-engine-ffi/`, `flipt-engine-wasm/`, and `flipt-engine-wasm-js/`.
- `.agents/personas/ffi-sdk-reviewer.md` for `flipt-client-python/`,
  `flipt-client-ruby/`, `flipt-client-java/`, `flipt-client-csharp/`,
  `flipt-client-dart/`, `flipt-client-kotlin-android/`, and
  `flipt-client-swift/`.
- `.agents/personas/wasm-sdk-reviewer.md` for `flipt-client-go/`,
  `flipt-client-js/`, `flipt-client-node/`, `flipt-client-browser/`, and
  `flipt-client-react/`.

If a PR touches multiple groups, synthesize the relevant findings into one review.
If a PR touches docs, release tooling, GitHub Actions, or test harness files, review
those changes directly using `AGENTS.md` and the central code-review guidance.

## Review priorities

Prioritize findings that affect:

1. Evaluation correctness across SDKs.
2. Public SDK API compatibility and documented behavior.
3. Resource lifecycle: `Close`, native handle freeing, stream shutdown, polling loops,
   WASM instance reuse, and background goroutine/task cleanup.
4. FFI/WASM boundary safety: type conversion, ownership, null/error handling, panics,
   serialization compatibility, and platform-specific loading.
5. Packaging and release impact: bundled engine artifacts, generated bindings, version
   files, platform metadata, and CI workflow triggers.
6. Tests that prove behavior, not just successful calls.

Do not spend review budget on mechanical formatting or lint findings that this repo's
CI already checks. Flag style only when it hides a real correctness, API, or maintenance
problem.
PROMPT
```

- [ ] **Step 2: Verify no unfinished marker text**

Run:

```bash
grep -nE 'REPLACE_ME|FIXME|XXX' .agents/prompts/00-sdk-review-routing.md || true
```

Expected output: empty.

- [ ] **Step 3: Commit the prompt**

Run:

```bash
git add .agents/prompts/00-sdk-review-routing.md
git commit -m "chore: add sdk review routing prompt"
```

---

### Task 3: Add Reviewer Personas

**Files:**
- Create: `.agents/personas/README.md`
- Create: `.agents/personas/rust-engine-reviewer.md`
- Create: `.agents/personas/ffi-sdk-reviewer.md`
- Create: `.agents/personas/wasm-sdk-reviewer.md`

- [ ] **Step 1: Create persona directory and README**

Run:

```bash
mkdir -p .agents/personas
cat > .agents/personas/README.md <<'README'
# Agent Personas

This directory contains repo-specific reviewer personas for the automated PR
review workflow. They are Markdown prompts consumed by `flipt-io/agents` as local
review overrides.

## Available personas

- `rust-engine-reviewer.md` — Rust evaluation and engine reviewer for
  `flipt-evaluation/` and `flipt-engine-*` changes.
- `ffi-sdk-reviewer.md` — native FFI SDK reviewer for Python, Ruby, Java, C#,
  Dart, Kotlin Android, and Swift changes.
- `wasm-sdk-reviewer.md` — WASM SDK reviewer for Go and JavaScript-family SDK
  changes.

## Conventions

- Produce one combined PR review; do not ask for separate reviews per SDK.
- Cite `file:line` for every finding.
- Separate blocking issues from suggestions.
- Defer mechanical formatting and lint-only findings to the existing CI jobs.
README
```

- [ ] **Step 2: Create Rust engine persona**

Run:

```bash
cat > .agents/personas/rust-engine-reviewer.md <<'RUST'
# Rust Engine Reviewer Persona

You are a senior Rust reviewer for the Flipt client evaluation engines. Use this
persona for changes under `flipt-evaluation/`, `flipt-engine-ffi/`,
`flipt-engine-wasm/`, and `flipt-engine-wasm-js/`.

## Focus

Review for issues that SDK CI and Rust linting will not reliably catch:

- Evaluation semantics changed unintentionally across boolean, variant, batch,
  list-flags, or snapshot behavior.
- FFI exports panic, leak memory, double-free memory, or return ambiguous null/error
  states to host languages.
- WASM exports expose incompatible serialization, initialization, or error shapes.
- Rust ownership/lifetime choices are safe internally but unsafe once exposed across
  C/WASM boundaries.
- HTTP, polling, streaming, or TLS behavior changes in a way that breaks existing
  SDK assumptions.
- Generated headers, wasm artifacts, or version metadata should be updated with the
  engine change.
- Tests assert only success and miss behavior, edge cases, or cross-engine parity.

## What to ignore

- Pure `rustfmt` formatting.
- Clippy-only style nits unless they hide a real bug.
- Broad architecture rewrites unrelated to the changed lines.

## Output expectations

Fold findings into the single combined PR review. For each finding, cite
`file:line`, explain the cross-SDK impact, and suggest the smallest safe fix.
RUST
```

- [ ] **Step 3: Create FFI SDK persona**

Run:

```bash
cat > .agents/personas/ffi-sdk-reviewer.md <<'FFI'
# FFI SDK Reviewer Persona

You are a senior SDK reviewer for Flipt's native FFI clients. Use this persona
for changes under `flipt-client-python/`, `flipt-client-ruby/`,
`flipt-client-java/`, `flipt-client-csharp/`, `flipt-client-dart/`,
`flipt-client-kotlin-android/`, and `flipt-client-swift/`.

## Focus

Review for issues that are common at native SDK boundaries:

- Native library loading breaks a supported platform, package layout, or CI artifact
  expectation.
- Client lifecycle is wrong: native handles are leaked, freed twice, used after
  `Close`, or left running after polling/streaming stops.
- FFI type conversion is lossy or unsafe: strings, bytes, booleans, maps, variants,
  contexts, errors, or null values cross the boundary incorrectly.
- Streaming and polling behavior diverges from the documented SDK behavior.
- Public SDK APIs, option names, defaults, or error shapes change without a clear
  compatibility reason.
- Platform packaging/version files were missed for a release-impacting change.
- Tests do not prove behavior users observe, especially evaluation results,
  cleanup, update mode, and error paths.

## Language-specific reminders

- Python/Ruby: check native loading paths, object finalization, context managers or
  block usage, and exception shapes.
- Java/Kotlin: check JNI loading, Android ABI assumptions, thread cleanup, and checked
  versus runtime exception behavior.
- C#: check P/Invoke signatures, disposal patterns, nullable values, and platform RID
  packaging.
- Dart/Swift: check FFI signatures, async/lifecycle behavior, mobile platform loading,
  and main-thread assumptions.

## What to ignore

- Formatter-only changes.
- Lint-only naming issues unless public API compatibility is affected.
- Suggestions to add new abstractions without a concrete bug.

## Output expectations

Fold findings into the single combined PR review. For each finding, cite
`file:line`, name the affected SDK/platform, and suggest the smallest safe fix.
FFI
```

- [ ] **Step 4: Create WASM SDK persona**

Run:

```bash
cat > .agents/personas/wasm-sdk-reviewer.md <<'WASM'
# WASM SDK Reviewer Persona

You are a senior SDK reviewer for Flipt's WASM-based clients. Use this persona
for changes under `flipt-client-go/`, `flipt-client-js/`, `flipt-client-node/`,
`flipt-client-browser/`, and `flipt-client-react/`.

## Focus

Review for issues that are common in WASM SDK wrappers:

- WASM module initialization, caching, or reuse is racy, repeated unnecessarily, or
  incompatible with supported runtimes.
- Browser, Node, React, and Go environments diverge in fetch behavior, URL handling,
  timers, cancellation, or error propagation.
- Evaluation APIs return shapes that break documented SDK compatibility.
- Polling/update loops leak timers, goroutines, promises, event listeners, or WASM
  instances after `Close`/cleanup.
- Bundle contents, generated WASM artifacts, package exports, or version metadata are
  missing for a packaging-impacting change.
- Tests only verify calls succeed and do not assert evaluation outputs, update behavior,
  cleanup, or error handling.

## Language-specific reminders

- Go: check `context.Context` handling, goroutine/timer cleanup, data races, and idiomatic
  errors.
- JavaScript/TypeScript: check ESM/CJS/browser package exports, typed public APIs,
  async initialization, and React hook cleanup when relevant.

## What to ignore

- Formatter-only changes.
- Bundler or lint noise already enforced by CI.
- Speculative rewrites that are not required to fix the changed behavior.

## Output expectations

Fold findings into the single combined PR review. For each finding, cite
`file:line`, name the affected runtime, and suggest the smallest safe fix.
WASM
```

- [ ] **Step 5: Verify persona index and files**

Run:

```bash
find .agents/personas -maxdepth 1 -type f -print | sort
```

Expected output:

```text
.agents/personas/README.md
.agents/personas/ffi-sdk-reviewer.md
.agents/personas/rust-engine-reviewer.md
.agents/personas/wasm-sdk-reviewer.md
```

- [ ] **Step 6: Verify no unfinished marker text**

Run:

```bash
grep -RInE 'REPLACE_ME|FIXME|XXX' .agents/personas .agents/prompts || true
```

Expected output: empty.

- [ ] **Step 7: Commit the personas**

Run:

```bash
git add .agents/personas
git commit -m "chore: add sdk reviewer personas"
```

---

### Task 4: Final Static Validation

**Files:**
- Inspect: `.github/workflows/pr-review.yml`
- Inspect: `.agents/prompts/00-sdk-review-routing.md`
- Inspect: `.agents/personas/*.md`

- [ ] **Step 1: Check whitespace and patch formatting**

Run:

```bash
git diff --check HEAD~3..HEAD
```

Expected output: empty.

If the commit count differs because tasks were squashed or amended, run instead:

```bash
git diff --check main...HEAD
```

Expected output: empty.

- [ ] **Step 2: Verify workflow does not checkout PR head or run code**

Run:

```bash
grep -nE 'pull_request\.head|github\.event\.pull_request\.head|npm |pnpm |yarn |cargo |go test|mise run|make ' .github/workflows/pr-review.yml || true
```

Expected output includes only the security comment lines mentioning PR head, and no build/test command lines.

- [ ] **Step 3: Verify the exact model**

Run:

```bash
grep -n 'cloudflare-workers-ai/@cf/moonshotai/kimi-k2.7-code' .github/workflows/pr-review.yml
```

Expected output includes the `model:` line.

- [ ] **Step 4: Verify working tree**

Run:

```bash
git status --short
```

Expected output: empty.
