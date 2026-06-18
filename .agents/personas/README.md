# Agent Personas

This directory contains repo-specific reviewer personas for the automated PR
review workflow. They are Markdown prompts consumed by `flipt-io/agents` as local
review overrides.

## Available personas

- `rust-engine-reviewer.md` — Rust evaluation and engine reviewer.
- `ffi-sdk-reviewer.md` — native FFI SDK reviewer.
- `wasm-sdk-reviewer.md` — WASM SDK reviewer.

Path routing lives in `.agents/prompts/00-sdk-review-routing.md`; keep this file
as an index so SDK directory changes have one source of truth.

## Conventions

- Produce one combined PR review; do not ask for separate reviews per SDK.
- Cite `file:line` for every finding.
- Separate blocking issues from suggestions.
- Defer mechanical formatting and lint-only findings to the existing CI jobs.
