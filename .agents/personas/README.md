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
