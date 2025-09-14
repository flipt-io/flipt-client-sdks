# Releasing

This describes the release process of the engines and SDKs.

## Overview

The release process is make up of two parts:

1. The release of the engines
2. The release of the SDKs

The engine is released first because the SDKs depend on the engine.

The entire process is made up of a series of GitHub Actions workflows that are triggered by a GitHub release.

## Release Process

### FFI

The FFI engine is released first because the SDKs depend on the engine.

#### Engine

The FFI engine gets built for each supported platform and is published to the GitHub release.

Releasing the engine is a three-step process:

1. Create a new release in the [flipt-engine-ffi](./flipt-engine-ffi) package by updating the Cargo.toml file to a new version, tag, and push.
2. The `release-ffi-engine` workflow will run, creating a new GitHub release.
3. The `package-ffi-engine` workflow will run, building the engine for each supported platform and publishing the artifacts to the GitHub release.

### WASM

The WASM engines are built and bundled when the SDKs that depend on them are released.

### SDKs

We use a Python script to release the SDKs. The script is located in the [release](./release) directory. See the [README](./release/README.md) for more information.
