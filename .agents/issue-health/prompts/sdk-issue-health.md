# Flipt Client SDK Issue Health

This repository contains Flipt client SDKs and their shared Rust evaluation
engines. Analyze newly opened issues for actionability and routing, not code
review.

## Repository-specific issue types

Treat requests for a new client SDK language as `feature` issues when the issue
asks for a language that is not currently supported.

When suggesting labels, use the `targetRepoLabels` supplied by the workflow as
the source of truth for labels that exist in this repository. Prefer an existing
label from that list when it accurately describes the issue type, affected SDK,
affected engine, or missing-information state.

Do not invent repository-specific label names and do not suggest labels only
because they exist. Suggest labels only when the issue content clearly supports
them.
