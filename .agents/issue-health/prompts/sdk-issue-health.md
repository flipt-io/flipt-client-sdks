# Flipt Client SDK Issue Health

This repository contains Flipt client SDKs and their shared Rust evaluation
engines. Analyze newly opened issues for actionability and routing, not code
review.

## Repository-specific issue types

Treat requests for a new client SDK language as `feature` issues when the issue
asks for a language that is not currently supported. If `targetRepoLabels`
contains `new-language`, suggest that label for these issues.
