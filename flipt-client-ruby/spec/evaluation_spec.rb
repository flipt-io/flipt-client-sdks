# frozen_string_literal: true

require_relative '../lib/evaluation'

RSpec.describe Flipt::EvaluationClient do
  describe '#initialize' do
    it 'initializes the engine' do
      url = ENV.fetch('FLIPT_URL', 'http://localhost:8080')
      client = Flipt::EvaluationClient.new('default', { url: url })
      expect(client).to be_a(Flipt::EvaluationClient)
    end
  end

  describe '#variant' do
    it 'returns a variant' do
      url = ENV.fetch('FLIPT_URL', 'http://localhost:8080')
      client = Flipt::EvaluationClient.new('default', { url: url })

      resp = client.variant({ flag_key: 'flag1', entity_id: 'someentity', context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['status']).to eq('success')
      expect(resp['error_message']).to be_nil
      expect(resp['result']['flag_key']).to eq('flag1')
      expect(resp['result']['match']).to eq(true)
      expect(resp['result']['reason']).to eq('MATCH_EVALUATION_REASON')
      expect(resp['result']['variant_key']).to eq('variant1')
      expect(resp['result']['segment_keys']).to eq(['segment1'])
    end
  end

  describe '#boolean' do
    it 'returns a boolean' do
      url = ENV.fetch('FLIPT_URL', 'http://localhost:8080')
      client = Flipt::EvaluationClient.new('default', { url: url })

      resp = client.boolean({ flag_key: 'flag_boolean', entity_id: 'someentity', context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['status']).to eq('success')
      expect(resp['error_message']).to be_nil
      expect(resp['result']['flag_key']).to eq('flag_boolean')
      expect(resp['result']['enabled']).to eq(true)
      expect(resp['result']['reason']).to eq('MATCH_EVALUATION_REASON')
    end
  end
end
