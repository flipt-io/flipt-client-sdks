# Releasing

This describes the release process of the engine and SDKs.

## Overview

The release process is make up of two parts:

1. The release of the engine
2. The release of the SDKs

The engine is released first because the SDKs depend on the engine.

The entire process is made up of a series of GitHub Actions workflows that are triggered by a GitHub release.

We use [release-please](https://github.com/googleapis/release-please>) to generate the CHANGELOGs and create the GitHub releases and tags for each component (engine and each SDK).

Release Please supports this via a [monorepo](https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md) configuration where we can version and release each component independently.

## Release Process

### Engine

<p align="center">
    <img src=".github/images/release-engine.png" width=600 />
</p>

1. Create a new release in the [flipt-engine](./flipt-engine) by merging a change to the `main` branch that would trigger a release via conventional commits.
2. Wait for the release-please workflow to complete. This will create a new release and tag for the engine.
3. The `package-engine` workflow will run, building the engine for each supported platform and publishing the artifacts to the GitHub release.

### SDKs

<p align="center">
    <img src=".github/images/release-sdk.png" width=600 />
</p>

#### Single SDK

1. Create a new release in the SDK by merging a change to the `main` branch that would trigger a release via conventional commits for that SDK. This will create a new release and tag for the SDK.
2. To update the SDK to use the new engine, you'll need to trigger a change that release-please will pick up.
3. To do this, you can run the following command from the root of the repository:

    ```bash
    ./build/bump.sh <language|all>
    ```

    Where `<language>` is the name of the SDK you want to trigger a change for.

    ```bash
    ./build/bump.sh ruby
    ```

4. Create a commit for the version you want to release: <https://github.com/googleapis/release-please#how-do-i-change-the-version-number>
5. Push the commit to the `main` branch.
6. Wait for the release-please workflow to complete. This will create a release PR for the SDK.
7. Merge the release PR for the SDK.
8. Release please will create a new release and tag for the SDK.
9. The `package-[name]-sdk` workflow will run, building the SDK for each supported platform and publishing the package to the supported package manager (e.g. RubyGems for Ruby).

#### All SDKs

1. Repeat step 3 above, but instead of passing a language, pass `all`.

    ```bash
    ./build/bump.sh all
    ```

2. Create a `feat` or `fix` commit depending on the type of change you want to release. See: [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).
3. Push or merge the commit to the `main` branch.
4. Wait for the release-please workflow to complete. This will create a release PR, updating all of the SDKs appropriately.
5. Merge the release PR.
6. Release please will create a new release and tag for each SDK.
7. The `package-[name]-sdk` workflow will run, building the SDK for each supported platform and publishing the package to the supported package manager (e.g. RubyGems for Ruby).
