# Contributing

## Issues

Let us know how we can help!

- Use a **clear and descriptive title** for the issue to identify the problem.
- Describe the **exact steps** to reproduce the problem in as many details as possible.
- Include the language client you are using and the version.
- Include any **stack traces** with your error
- List versions you are using: Flipt, OS, etc.

## Code

It's always best to open a dialogue before investing a lot of time into a fix or new functionality.

Functionality must meet the design goals and vision for the project to be accepted; we would be happy to discuss how your idea can best fit into the future of Flipt.

Join our [Discord](https://www.flipt.io/discord) to chat with the team about any feature ideas or open a [Discussion](https://github.com/flipt-io/flipt/discussions) here on GitHub.

## Adding a New Language Client

If you would like to add a new language client, please open an issue first to discuss the design and implementation.

You can open a new issue using the provided issue template: [New Language](https://github.com/flipt-io/flipt-client-sdks/issues/new?labels=new-language&template=new_language.yml)

You can also look for existing issues with the `new-language` label: <https://github.com/flipt-io/flipt-client-sdks/issues?q=is%3Aissue+is%3Aopen+label%3Anew-language>.

After the design and implementation have been discussed and agreed upon, please follow the steps below to add the new language client.

### 1. Create a New Directory

Create a new directory in the root of the repository with the name `flipt-client-{language}` (e.g. `flipt-client-java`).

### 2. Add the Client

Add the client to the new directory. The client should be a thin wrapper around the engine. See the existing clients for examples.

The structure of the client folder should be as follows:

1. A `README.md` with installation and usage instructions.
2. A `src` directory with the source code for the client (or follow appropriate for the language).
3. A `test` directory with the tests for the client (or follow structure appropriate for the language).

The client shape should be as follows:

1. **Constructor**: Should take in an optional namespace and engine options. Engine options should be a map or similar idiomatic data structure with the following keys:

   1. `url`: The URL of the Flipt server to connect to.
   2. `update_interval`: The interval in seconds to wait for updated flag state.
   3. `authentication_strategy`: The authentication strategy to use. Should be one of the following:
      - `client_token`: Client Token Bearer key authentication.
      - `jwt`: JWT Bearer key authentication.

   Note: If no namespace is specified, the client should default to the `default` namespace.

2. **Evaluate Variant** method: Should take in a flag key, entity ID, and context. Should return a variant result. Follow language conventions for naming.
3. **Evaluate Boolean** method: Should take in a flag key, entity ID, and context. Should return a boolean result. Follow language conventions for naming.
4. **Evaluate Batch** method: Should take in a list of flag keys, entity ID, and context. Should return a list of evaluation results. Follow language conventions for naming.
5. **List Flags** method: Should return a list of all flags in the namespace. Follow language conventions for naming.
6. **Close** method (language dependent): Should close the engine and free any memory allocated by the engine. Follow language conventions for naming.

### 3. Interface with the Engine

The client should interface with the engine to evaluate flags. The engine is responsible for connecting to the Flipt server, fetching flag state, and evaluating flags.

You don't have to build the engine from scratch in order to develop a new client. You can download the latest engine version from the [releases page](https://github.com/flipt-io/flipt-client-sdks/releases/latest) for your platform and use it while developing the client.

### 4. Setup Tests (optional)

Prerequisites:

- [Dagger](https://docs.dagger.io/install/#stable-release)
- [Go](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)

See the [test/README.md](./test/README.md) for more information on how to setup tests for the new client.

If you'd like to add tests using Dagger, you can follow the steps below. If you are not familiar with Dagger, you can skip this step and simply provide instructions for running the tests locally.

Feel free to ask the team for help with this step.

1. Update the `test/main.go` file to include the new client in the list of clients to test.
2. Update `test/main.go` to run the tests for the new client using Dagger. See the existing clients for examples.
3. Ensure the tests pass locally by running `dagger run go run ./test/... -sdks {comma separated list of languages}` from the root of the repository. Note: You will need to have Docker, Go, and Dagger installed locally to run the tests.

### 5. Update README

Update the [README.md](./README.md) to include the new client in the list of clients. Follow the existing clients for examples.

### 6. Setup Dependabot (optional)

Add a new section to the [dependabot.yml](./.github/dependabot.yml) file for the new language if applicable. Follow the existing languages for examples.

### 7. Create a PR

Create a PR with the changes and we will review it as soon as possible. Please add any language-specific idioms when it comes to packaging the client. This will allow us to automate and publish the client to the appropriate package manager.

### Conventional Commits

We use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages. Please adhere to this specification when contributing.

## Developer Certificate of Origin

We respect the intellectual property rights of others and we want to make sure
all incoming contributions are correctly attributed and licensed. A Developer
Certificate of Origin (DCO) is a lightweight mechanism to do that. The DCO is
a declaration attached to every commit. In the commit message of the contribution,
the developer simply adds a `Signed-off-by` statement and thereby agrees to the DCO,
which you can find below or at [DeveloperCertificate.org](http://developercertificate.org/).

```text
Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the
    best of my knowledge, is covered under an appropriate open
    source license and I have the right under that license to
    submit that work with modifications, whether created in whole
    or in part by me, under the same open source license (unless
    I am permitted to submit under a different license), as
    Indicated in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including
    all personal information I submit with it, including my
    sign-off) is maintained indefinitely and may be redistributed
    consistent with this project or the open source license(s)
    involved.
```

We require that every contribution to Flipt to be signed with a DCO. We require the
usage of known identity (such as a real or preferred name). We do not accept anonymous
contributors nor those utilizing pseudonyms. A DCO signed commit will contain a line like:

```text
Signed-off-by: Jane Smith <jane.smith@email.com>
```

You may type this line on your own when writing your commit messages. However, if your
user.name and user.email are set in your git configs, you can use `git commit` with `-s`
or `--signoff` to add the `Signed-off-by` line to the end of the commit message. We also
require revert commits to include a DCO.
