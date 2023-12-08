require "../flipt-client-ruby/lib/evaluation"

# note: this script assumes you have built the flipt-client Ruby gem locally via the instructions in the README
# and that the flag "my_feature" exists in the default namespace
# and that the flag is a boolean flag

FLIPT_URL = ENV.fetch("FLIPT_URL", "http://localhost:8080")

NUM_CLIENTS = 100
NUM_REQUESTS_PER_CLIENT = 1000

# create 100 instances of the client
clients = []

NUM_CLIENTS.times do
    clients << Flipt::EvaluationClient.new("default", {url: FLIPT_URL, auth_token: "secret"})
end

# capture start time
start_time = Time.now

# each client will make 1000 requests
clients.each do |client|
    NUM_REQUESTS_PER_CLIENT.times do
        entity_id = rand(1000000).to_s
        resp = client.evaluate_boolean({flag_key: "my-feature", entity_id: entity_id})
    end
end

puts "done: #{clients.length} clients made #{clients.length * NUM_REQUESTS_PER_CLIENT} requests in total"
puts "total time: #{Time.now - start_time} seconds"