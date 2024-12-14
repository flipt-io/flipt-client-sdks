# Flipt Client Ruby

[![Gem Version](https://badge.fury.io/rb/flipt_client.svg)](https://badge.fury.io/rb/flipt_client)

The `flipt-client-ruby` library contains the Ruby source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
gem install flipt_client
```

## How Does It Work?

The `flipt-client-ruby` library is a wrapper around the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-ruby` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `update_interval` option when constructing a client. The default interval is 120 seconds.

### Streaming (Flipt Cloud Only)

[Flipt Cloud](https://flipt.io/cloud) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64

### Using System Libffi

If you are experiencing segfaults when using this gem, you may need to configure `ffi` to use the system libffi instead of the bundled one.

You can do this before installing the gem by running the following command:

```bash
gem install ffi -- --enable-system-libffi        # to install the gem manually
bundle config build.ffi --enable-system-libffi   # for bundle install
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
#   "authentication": {
#     "client_token": "secret"
#   }
# }
#
# You can replace the url with where your upstream Flipt instance points to, the update interval for how long you are willing
# to wait for updated flag state, and the auth token if your Flipt instance requires it.
client = Flipt::EvaluationClient.new()
resp = client.evaluate_variant({ flag_key: 'buzz', entity_id: 'someentity', context: { fizz: 'buzz' } })

puts resp
```

### Constructor Arguments

The `Flipt::EvaluationClient` constructor accepts two optional arguments:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `opts`: A hash that supports several options for the client. The structure is:
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `update_interval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `fetch_mode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Load Test

1. To run the load test, you'll need to have Flipt running locally. You can do this by running the following command from the root of the repository:

   ```bash
   docker run -d \
       -p 8080:8080 \
       -p 9000:9000 \
       flipt/flipt:latest
   ```

2. You'll also need to have the `flipt_client` gem installed locally. See [Installation](#installation) above.
3. In the Flipt UI (<http://localhost:8080>) you'll also need to create a new boolean flag with the key `my-feature` in the default namespace.
4. You can then run the load test by running the following command from this folder:

   ```bash
   bundle exec ruby load_test.rb
   ```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
