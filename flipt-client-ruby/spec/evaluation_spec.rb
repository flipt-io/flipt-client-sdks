# frozen_string_literal: true

require_relative '../lib/flipt_client'

RSpec.describe Flipt::EvaluationClient do
  before(:all) do
    url = ENV.fetch('FLIPT_URL', 'http://localhost:8080')
    auth_token = ENV.fetch('FLIPT_AUTH_TOKEN', 'secret')
    @client = Flipt::EvaluationClient.new('default', { url: url, auth_token: auth_token })
  end

  describe '#evaluate_variant' do
    it 'returns a variant result' do
      resp = @client.evaluate_variant({ flag_key: 'flag1', entity_id: 'someentity', context: { "fizz": 'buzz' } })

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

  describe '#evaluate_boolean' do
    it 'returns a boolean result' do
      resp = @client.evaluate_boolean({ flag_key: 'flag_boolean', entity_id: 'someentity', context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['status']).to eq('success')
      expect(resp['error_message']).to be_nil
      expect(resp['result']['flag_key']).to eq('flag_boolean')
      expect(resp['result']['enabled']).to eq(true)
      expect(resp['result']['reason']).to eq('MATCH_EVALUATION_REASON')
    end
  end

  describe '#evaluate_variant failure' do
    it 'gracefully handles failures for variant flag evaluation' do
      resp = @client.evaluate_variant({ flag_key: 'nonexistent', entity_id: 'someentity', context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['result']).to be_nil
      expect(resp['status']).to eq('failure')
      expect(resp['error_message']).to eq('invalid request: failed to get flag information default/nonexistent')
    end
  end

  describe '#evaluate_boolean failure' do
    it 'gracefully handles failures for boolean flag evaluation' do
      resp = @client.evaluate_boolean({ flag_key: 'nonexistent', entity_id: 'someentity', context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['result']).to be_nil
      expect(resp['status']).to eq('failure')
      expect(resp['error_message']).to eq('invalid request: failed to get flag information default/nonexistent')
    end
  end
end
