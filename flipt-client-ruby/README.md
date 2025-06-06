# Flipt Client Ruby

[![flipt-client-ruby](https://badge.fury.io/rb/flipt_client.svg)](https://badge.fury.io/rb/flipt_client)

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

### Streaming (Flipt Cloud/Flipt v2)

[Flipt Cloud](https://flipt.io/cloud) and [Flipt v2](https://docs.flipt.io/v2) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

### Retries

The SDK will automatically retry fetching (or initiating streaming) flag state if the client is unable to reach the Flipt server temporarily.

The SDK will retry up to 3 times with an exponential backoff interval between retries. The base delay is 1 second and the maximum delay is 30 seconds.

Retriable errors include:

- `429 Too Many Requests`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`
- Other potential transient network or DNS errors

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

## Migration Notes

### Pre-1.0.0 -> 1.0.0

This section is for users who are migrating from a previous (pre-1.0.0) version of the SDK.

- `Flipt::EvaluationClient` has been renamed to `Flipt::Client`. Update all usages and imports accordingly. A deprecation warning is emitted if you use the old class.
- All evaluation methods now use **keyword arguments** (e.g., `flag_key:`, `entity_id:`, `context:`) instead of a single hash argument. Update your method calls to use keyword arguments.
- All evaluation methods now return **response model objects** (`VariantEvaluationResponse`, `BooleanEvaluationResponse`, `BatchEvaluationResponse`, `ErrorEvaluationResponse`) instead of raw hashes. Update your code to use attribute readers (e.g., `resp.flag_key`, `resp.enabled`).
- Batch evaluation responses now contain an array of model objects, not hashes.
- Error handling is now standardized and idiomatic. All errors inherit from `Flipt::Error` (with subclasses like `ValidationError`, `EvaluationError`). Update your rescue blocks accordingly.
- The minimum supported Ruby version is now 2.7.0.
- The client constructor now accepts keyword arguments for configuration (e.g., `url:`, `namespace:`, `authentication:`). The `environment` option is now supported.
- The API and documentation are more idiomatic and Ruby-like throughout.

## Usage

In your Ruby code you can import this client and use it as so:

```ruby
require 'flipt_client'

# namespace is an optional keyword argument and will default to "default" if not specified.
# opts is a hash of options for the client:
# {
#   url: "http://localhost:8080",
#   update_interval: 120,
#   authentication: Flipt::ClientTokenAuthentication.new("secret")
# }
client = Flipt::Client.new(url: 'http://localhost:8080', authentication: Flipt::ClientTokenAuthentication.new('secret'))

resp = client.evaluate_variant(flag_key: 'buzz', entity_id: 'someentity', context: { fizz: 'buzz' })
puts resp.flag_key # => 'buzz'
puts resp.match # => true
puts resp.reason # => 'MATCH_EVALUATION_REASON'

resp = client.evaluate_boolean(flag_key: 'my-feature', entity_id: 'someentity')
puts resp.enabled # => true
```

### Constructor Arguments

The `Flipt::Client` constructor accepts the following keyword arguments:

- `environment`: The environment to fetch flag state from. Defaults to `default`.
- `namespace`: The namespace to fetch flag state from. Defaults to `default`.
- `url`: The URL of the upstream Flipt instance. Defaults to `http://localhost:8080`.
- `request_timeout`: Timeout (in seconds) for requests. Defaults to no timeout.
- `update_interval`: Interval (in seconds) to fetch new flag state. Defaults to 120 seconds.
- `authentication`: The authentication strategy to use. Defaults to no authentication. See [Authentication](#authentication).
- `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state.
- `fetch_mode`: The fetch mode to use. Defaults to polling.
- `error_strategy`: The error strategy to use. Defaults to fail. See [Error Strategies](#error-strategies).
- `snapshot`: The snapshot to use when initializing the client. Defaults to no snapshot. See [Snapshotting](#snapshotting).

### Authentication

The `Flipt::Client` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The client supports the following error strategies:

- `fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### Response Models

All evaluation methods return response model objects:

- `evaluate_variant` returns a `Flipt::VariantEvaluationResponse`
- `evaluate_boolean` returns a `Flipt::BooleanEvaluationResponse`
- `evaluate_batch` returns a `Flipt::BatchEvaluationResponse`

### Error Handling

All errors inherit from `Flipt::Error`. Common subclasses include:

- `Flipt::ValidationError`
- `Flipt::EvaluationError`

You can rescue these errors as needed:

```ruby
begin
  client.evaluate_variant(flag_key: 'missing', entity_id: 'user')
rescue Flipt::EvaluationError => e
  puts "Evaluation failed: #{e.message}"
end
```

### Snapshotting

The client supports snapshotting of flag state as well as seeding the client with a snapshot for evaluation. This is helpful if you want to use the client in an environment where the Flipt server is not guaranteed to be available or reachable on startup.

To get the snapshot for the client, you can use the `get_snapshot` method. This returns a base64 encoded JSON string that represents the flag state for the client.

You can set the snapshot for the client using the `snapshot` option when constructing a client.

**Note:** You most likely will want to also set the `error_strategy` to `fallback` when using snapshots. This will ensure that you wont get an error if the Flipt server is not available or reachable even on the initial fetch.

You also may want to store the snapshot in a local file so that you can use it to seed the client on startup.

> [!IMPORTANT]
> If the Flipt server becomes reachable after the setting the snapshot, the client will replace the snapshot with the new flag state from the Flipt server.

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
