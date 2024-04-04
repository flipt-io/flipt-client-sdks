# Flipt Client Ruby

[![Gem Version](https://badge.fury.io/rb/flipt_client.svg)](https://badge.fury.io/rb/flipt_client)

The `flipt-client-ruby` library contains the Ruby source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
gem install flipt_client
```

## Using System Libffi

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
