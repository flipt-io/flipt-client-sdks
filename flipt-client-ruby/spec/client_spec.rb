# frozen_string_literal: true

require_relative '../lib/flipt_client'

RSpec.describe Flipt::EvaluationClient do
  before(:all) do
    url = ENV.fetch('FLIPT_URL', 'http://localhost:8080')
    auth_token = ENV.fetch('FLIPT_AUTH_TOKEN', 'secret')
    @client = Flipt::EvaluationClient.new('default',
                                          { url: url,
                                            authentication: Flipt::ClientTokenAuthentication.new(auth_token) })
  end

  describe '#evaluate_variant' do
    it 'returns a variant result' do
      resp = @client.evaluate_variant({ flag_key: 'flag1', entity_id: 'someentity', context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['flag_key']).to eq('flag1')
      expect(resp['match']).to eq(true)
      expect(resp['reason']).to eq('MATCH_EVALUATION_REASON')
      expect(resp['variant_key']).to eq('variant1')
      expect(resp['segment_keys']).to eq(['segment1'])
    end
  end

  describe '#evaluate_boolean' do
    it 'returns a boolean result' do
      resp = @client.evaluate_boolean({ flag_key: 'flag_boolean', entity_id: 'someentity',
                                        context: { "fizz": 'buzz' } })

      expect(resp).to_not be_nil
      expect(resp['flag_key']).to eq('flag_boolean')
      expect(resp['enabled']).to eq(true)
      expect(resp['reason']).to eq('MATCH_EVALUATION_REASON')
    end
  end

  describe '#evaluate_batch' do
    it 'returns a batch result' do
      resp = @client.evaluate_batch([{ flag_key: 'flag1', entity_id: 'someentity', context: { "fizz": 'buzz' } },
                                     { flag_key: 'flag_boolean', entity_id: 'someentity', context: { "fizz": 'buzz' } }, { flag_key: 'notfound', entity_id: 'someentity', context: { "fizz": 'buzz' } }])

      expect(resp).to_not be_nil

      expect(resp['responses'].length).to be == 3

      variant = resp['responses'][0]
      expect(variant['type']).to eq('VARIANT_EVALUATION_RESPONSE_TYPE')
      expect(variant['variant_evaluation_response']['flag_key']).to eq('flag1')
      expect(variant['variant_evaluation_response']['match']).to eq(true)
      expect(variant['variant_evaluation_response']['reason']).to eq('MATCH_EVALUATION_REASON')
      expect(variant['variant_evaluation_response']['variant_key']).to eq('variant1')
      expect(variant['variant_evaluation_response']['segment_keys']).to eq(['segment1'])

      boolean = resp['responses'][1]
      expect(boolean['type']).to eq('BOOLEAN_EVALUATION_RESPONSE_TYPE')
      expect(boolean['boolean_evaluation_response']['flag_key']).to eq('flag_boolean')
      expect(boolean['boolean_evaluation_response']['enabled']).to eq(true)
      expect(boolean['boolean_evaluation_response']['reason']).to eq('MATCH_EVALUATION_REASON')

      error = resp['responses'][2]
      expect(error['type']).to eq('ERROR_EVALUATION_RESPONSE_TYPE')
      expect(error['error_evaluation_response']['flag_key']).to eq('notfound')
      expect(error['error_evaluation_response']['namespace_key']).to eq('default')
      expect(error['error_evaluation_response']['reason']).to eq('NOT_FOUND_ERROR_EVALUATION_REASON')
    end
  end

  describe '#evaluate_variant failure' do
    it 'gracefully handles failures for variant flag evaluation' do
      expect { @client.evaluate_variant({ flag_key: 'nonexistent', entity_id: 'someentity', context: { "fizz": 'buzz' } }) }
        .to raise_error(::Flipt::Error, 'invalid request: failed to get flag information default/nonexistent')
    end
  end

  describe '#evaluate_boolean failure' do
    it 'gracefully handles failures for boolean flag evaluation' do
      expect { @client.evaluate_boolean({ flag_key: 'nonexistent', entity_id: 'someentity', context: { "fizz": 'buzz' } }) }
        .to raise_error(::Flipt::Error, 'invalid request: failed to get flag information default/nonexistent')
    end
  end

  describe '#list_flags' do
    it 'returns a list of flags' do
      resp = @client.list_flags

      expect(resp).to_not be_nil
      expect(resp).to include({ 'description' => '', 'enabled' => true, 'key' => 'flag_boolean', 'type' => 'BOOLEAN_FLAG_TYPE' })
    end
  end
end
