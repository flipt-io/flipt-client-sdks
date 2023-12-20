# Flipt Client Ruby

The `flipt-client-ruby` directory contains the Ruby source code for the Flipt client-side evaluation client.

## Installation

Currently, to use this client, you'll need to build the dynamic library and the gem locally and install it. This is a temporary solution until we can figure out a better way to package and distribute the libraries.

The dynamic library will contain the functionality necessary for the client to make calls to the Flipt engine via FFI. See [flipt-engine](../flipt-engine) for more information on the Flipt engine and FFI.

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [Ruby](https://www.ruby-lang.org/en/documentation/installation/)
- [Make](https://www.gnu.org/software/make/)

### Automated Build

1. Build and copy the dynamic library to the `flipt-client-ruby/lib/ext` directory for yoru platform. This will also build and install the `flipt_client` gem on your local machine. You can do this by running the following command from the root of the repository:

    ```bash
    make ruby
    ```

### Manual Build

1. Build the Rust dynamic library

    ```bash
    cargo build --release
    ```

This should generate a `target/` directory in the root of this repository, which contains the dynamically linked library built for your platform.

2. You'll need to copy the dynamic library to the `flipt-client-ruby/lib/ext/$OS_$ARCH/` directory. This is a temporary solution until we can figure out a better way to package the libraries with the gem.

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

3. You can then build the gem and install it locally. You can do this by running the following command from the `flipt-client-ruby` directory:

    ```bash
    rake build
    gem install pkg/flipt_client-{version}.gem
    ```

## Usage

In your Ruby code you can import this client and use it as so:

```ruby
require 'flipt_client'

# namespace is the first positional argument and is optional here and will have a value of "default" if not specified.
# opts is the second positional argument and is also optional, the structure is:
# {
#   "url": "http://localhost:8080",
#   "update_interval": 120,
#   "auth_token": "secret"
# }
# 
# You can replace the url with where your upstream Flipt instance points to, the update interval for how long you are willing
# to wait for updated flag state, and the auth token if your Flipt instance requires it.
client = Flipt::EvaluationClient.new()
resp = client.evaluate_variant({ flag_key: 'buzz', entity_id: 'someentity', context: { fizz: 'buzz' } })

puts resp
```

## Load Test

1. To run the load test, you'll need to have Flipt running locally. You can do this by running the following command from the root of the repository:

    ```bash
    docker run -d \
        -p 8080:8080 \
        -p 9000:9000 \
        docker.flipt.io/flipt/flipt:latest
    ```

2. You'll also need to have the `flipt_client` gem installed locally. See [Installation](#installation) above.
3. In the Flipt UI (<http://localhost:8080>) you'll also need to create a new boolean flag with the key `my-feature` in the default namespace.
4. You can then run the load test by running the following command from this folder:

    ```bash
    bundle exec ruby load_test.rb
    ```
