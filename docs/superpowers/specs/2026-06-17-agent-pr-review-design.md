# Agent PR Review Workflow Design

## Goal

Add automated AI PR review to `flipt-client-sdks`, matching the main Flipt repo pattern while keeping the workflow safe for forked pull requests.

## Decisions

- Use one combined PR review per pull request.
- Run on `pull_request_target` for PRs targeting `main`.
- Skip Dependabot PRs.
- Checkout only the base repository state. Do not checkout or execute PR head code.
- Use `flipt-io/agents/actions/pr-review@main`.
- Use Cloudflare Workers AI model `cloudflare-workers-ai/@cf/moonshotai/kimi-k2.7-code`.
- Pass Cloudflare credentials through `CLOUDFLARE_API_KEY` and `CLOUDFLARE_ACCOUNT_ID` secrets.

## Files to add

### `.github/workflows/pr-review.yml`

A GitHub Actions workflow that:

- triggers on `pull_request_target` for `opened`, `synchronize`, and `reopened` PR events targeting `main`
- grants `contents: read` and `pull-requests: write`
- cancels older review runs for the same PR
- skips `dependabot[bot]`
- checks out the base branch only
- calls the central Flipt PR review action with the PR number and Kimi K2.7 Code model
- documents the `pull_request_target` security constraint inline

### `.agents/prompts/00-sdk-review-routing.md`

Repo-specific review guidance that tells the central reviewer to:

- produce one combined review
- focus on changed SDKs and engines only
- apply relevant local personas based on changed paths
- avoid lint-only findings handled by CI
- prioritize correctness, SDK API compatibility, resource cleanup, generated/binary packaging impact, and tests

### `.agents/personas/README.md`

A short index describing the local reviewer personas and how they are used.

### `.agents/personas/rust-engine-reviewer.md`

Persona for Rust engine changes under:

- `flipt-evaluation/`
- `flipt-engine-ffi/`
- `flipt-engine-wasm/`
- `flipt-engine-wasm-js/`

Focus areas:

- evaluation correctness
- FFI/WASM boundary safety
- ownership and lifetime mistakes
- serialization compatibility
- panic/error handling across language boundaries
- tests for evaluator behavior and engine exports

### `.agents/personas/ffi-sdk-reviewer.md`

Persona for FFI SDK changes under:

- `flipt-client-python/`
- `flipt-client-ruby/`
- `flipt-client-java/`
- `flipt-client-csharp/`
- `flipt-client-dart/`
- `flipt-client-kotlin-android/`
- `flipt-client-swift/`

Focus areas:

- native library loading
- lifecycle and close/free behavior
- streaming/polling state updates
- type conversion at FFI boundaries
- platform packaging and versioning
- preserving public SDK APIs

### `.agents/personas/wasm-sdk-reviewer.md`

Persona for WASM SDK changes under:

- `flipt-client-go/`
- `flipt-client-js/`
- deprecated JS package paths if touched
- React/browser/node wrapper code where present

Focus areas:

- WASM initialization and reuse
- async fetch/update behavior
- environment differences between browser, Node, and Go
- bundle/package contents
- public SDK API compatibility
- tests for evaluation, update, and error paths

## Out of scope

- Per-language persona files for every SDK.
- Running build, lint, or test commands inside the PR review workflow.
- Checking out or executing untrusted PR code.
- Replacing existing SDK CI workflows.

## Validation

After implementation:

- `git diff --check`
- inspect the workflow YAML for `pull_request_target`, Dependabot skip, base checkout only, and the exact model `cloudflare-workers-ai/@cf/moonshotai/kimi-k2.7-code`
- inspect `.agents/` files for clear routing and no placeholder text
