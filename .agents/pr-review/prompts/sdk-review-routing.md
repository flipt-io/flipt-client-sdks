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
If a PR touches docs, release tooling, GitHub Actions, test harness files, root
configuration, dependency automation, `package/`, `.mise.toml`, `Cargo.toml`, or
other repository metadata, review those changes directly using `AGENTS.md` plus
the central `flipt-io/agents` code-review skill guidance.

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
