require 'rspec'
require 'json'
require_relative '../lib/evaluation'

RSpec.describe Flipt::EvaluationClient do
  let(:state) { File.read('spec/fixtures/state.json') }
  let(:client) { Flipt::EvaluationClient.new('default', state: state) }

  describe '#variant' do
    let (:evaluation_request) { {flag_key: "flag1", entity_id: "entity1", context: {fizz:"buzz"}} }
    it 'returns the variant' do
      resp = client.variant(evaluation_request)
      expect(resp["status"]).to eq("success")
      expect(resp["result"]).to_not be_nil
      expect(resp["result"]["flag_key"]).to eq("flag1")
      expect(resp["result"]["variant_key"]).to eq("variant1")
      expect(resp["result"]["match"]).to eq(true)
    end
  end

  describe '#boolean' do
    skip 'TODO: implement boolean evaluation spec'
  end
end