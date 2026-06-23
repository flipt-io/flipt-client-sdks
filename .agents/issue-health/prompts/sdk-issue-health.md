# Flipt Client SDK Issue Health

This repository contains Flipt client SDKs and their shared Rust evaluation
engines. Analyze newly opened issues for actionability and routing, not code
review.

## Repository-specific issue types

Treat requests for a new client SDK language as `feature` issues and suggest the
existing `new-language` label when the issue asks for a language that is not
currently supported.

When missing information blocks action, suggest the existing `needs more info`
label rather than `needs-info`.

## Useful existing labels

Prefer these existing labels when they accurately describe the issue: `bug`,
`documentation`, `enhancement`, `question`, `new-language`, `needs more info`,
`go`, and `javascript`.

Do not suggest labels only because they exist. Suggest labels only when the issue
content clearly supports them.
