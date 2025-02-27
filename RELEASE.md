# Releasing

This describes the release process of the engines and SDKs.

## Overview

The release process is make up of two parts:

1. The release of the engines
2. The release of the SDKs

The engine is released first because the SDKs depend on the engine.

The entire process is made up of a series of GitHub Actions workflows that are triggered by a GitHub release.

We use [release-please](https://github.com/googleapis/release-please) to generate the CHANGELOG and create the GitHub releases for the engine only.

## Release Process

### FFI

The FFI engine is released first because the SDKs depend on the engine.

#### Engine

The FFI engine gets built for each supported platform and is published to the GitHub release.

Releasing the engine is a three-step process:

1. Create a new release in the [flipt-engine-ffi](./flipt-engine-ffi) package by merging a change to the `main` branch that would trigger a release via conventional commits.
2. Wait for the release-please workflow to complete. This will create a new release and tag for the engine.
3. The `package-ffi-engine` workflow will run, building the engine for each supported platform and publishing the artifacts to the GitHub release.

### WASM

The WASM engines are built and bundled when the SDKs that depend on them are released.

### SDKs

We use a Python script to release the SDKs. The script is located in the [release](./release) directory. See the [README](./release/README.md) for more information.
