# Contributing

## Issues

Let us know how we can help!

* Use a **clear and descriptive title** for the issue to identify the problem.
* Describe the **exact steps** which reproduce the problem in as many details as possible.
* Include the language client you are using and the version.
* Include any **stack traces** with your error
* List versions you are using: Flipt, OS, etc.

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

1. **Constructor**: Should take in an optional namespace and engine options. Engine options should be a map or similar idomatic data structure with the following keys:
    1. `url`: The URL of the Flipt server to connect to.
    2. `update_interval`: The interval in seconds to wait for updated flag state.
    3. `auth_token`: The auth token to use when connecting to the Flipt server.

    Note: If no namespace is specified, the client should default to the `default` namespace.

2. **Evaluate_Variant** method: Should take in a flag key, entity ID, and context. Should return a variant result. Follow language conventions for naming.
3. **Evaluate_Boolean** method: Should take in a flag key, entity ID, and context. Should return a variant result. Follow language conventions for naming.
4. **Close** method (language dependent): Should close the engine and free any memory allocated by the engine. Follow language conventions for naming.

### 3. Setup Tests

Feel free to ask the team for help with this step.

1. Update the `test/main.go` file to include the new client in the list of clients to test.
2. Update `test/main.go` to run the tests for the new client using Dagger. See the existing clients for examples.
3. Ensure the tests pass locally by running `dagger run go run ./test/... --languages={language}` from the root of the repository. Note: You will need to have Docker, Go, and Dagger installed locally to run the tests.

### 4. Update README

Update the [README.md](./README.md) to include the new client in the list of clients. Follow the existing clients for examples.

### 5. Create a PR

Create a PR with the changes and we will review it as soon as possible. Please add any language specific idioms when it comes to packaging the client. This will allow us to automate and publish the client to the appropriate package manager.

### Conventional Commits

We use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages. Please adhere to this specification when contributing.

## Legal

By submitting a Pull Request, you disavow any rights or claims to any changes submitted to this project and assign the copyright of those changes to Flipt Software Inc.

If you cannot or do not want to reassign those rights (your employment contract for your employer may not allow this), you should not submit a PR. Open an issue and someone else can do the work.

This is a legal way of saying "If you submit a PR to us, that code becomes ours". 99.9% of the time that's what you intend anyways; we hope it doesn't scare you away from contributing.
