# frozen_string_literal: true

require_relative '../lib/flipt_client'

RSpec.describe Flipt::Client do
  before(:all) do
    url = ENV.fetch('FLIPT_URL', 'http://localhost:8080')
    auth_token = ENV.fetch('FLIPT_AUTH_TOKEN', 'secret')
    @client = Flipt::Client.new(url: url, authentication: Flipt::ClientTokenAuthentication.new(auth_token))
  end

  describe '#evaluate_variant' do
    it 'returns a variant result' do
      resp = @client.evaluate_variant(flag_key: 'flag1', entity_id: 'someentity', context: { 'fizz' => 'buzz' })
      expect(resp).to be_a(Flipt::VariantEvaluationResponse)
      expect(resp.flag_key).to eq('flag1')
      expect(resp.match).to eq(true)
      expect(resp.reason).to eq('MATCH_EVALUATION_REASON')
      expect(resp.variant_key).to eq('variant1')
      expect(resp.segment_keys).to eq(['segment1'])
    end
  end

  describe '#evaluate_boolean' do
    it 'returns a boolean result' do
      resp = @client.evaluate_boolean(flag_key: 'flag_boolean', entity_id: 'someentity', context: { 'fizz' => 'buzz' })
      expect(resp).to be_a(Flipt::BooleanEvaluationResponse)
      expect(resp.flag_key).to eq('flag_boolean')
      expect(resp.enabled).to eq(true)
      expect(resp.reason).to eq('MATCH_EVALUATION_REASON')
    end
  end

  describe '#evaluate_batch' do
    it 'returns a batch result' do
      resp = @client.evaluate_batch(requests: [
                                      { flag_key: 'flag1', entity_id: 'someentity', context: { 'fizz' => 'buzz' } },
                                      { flag_key: 'flag_boolean', entity_id: 'someentity', context: { 'fizz' => 'buzz' } },
                                      { flag_key: 'notfound', entity_id: 'someentity', context: { 'fizz' => 'buzz' } }
                                    ])
      expect(resp).to be_a(Flipt::BatchEvaluationResponse)
      expect(resp.responses.length).to eq(3)
      variant = resp.responses[0]
      expect(variant).to be_a(Flipt::VariantEvaluationResponse)
      expect(variant.flag_key).to eq('flag1')
      expect(variant.match).to eq(true)
      expect(variant.reason).to eq('MATCH_EVALUATION_REASON')
      expect(variant.variant_key).to eq('variant1')
      expect(variant.segment_keys).to eq(['segment1'])
      boolean = resp.responses[1]
      expect(boolean).to be_a(Flipt::BooleanEvaluationResponse)
      expect(boolean.flag_key).to eq('flag_boolean')
      expect(boolean.enabled).to eq(true)
      expect(boolean.reason).to eq('MATCH_EVALUATION_REASON')
      error = resp.responses[2]
      expect(error).to be_a(Flipt::ErrorEvaluationResponse)
      expect(error.flag_key).to eq('notfound')
      expect(error.namespace_key).to eq('default')
      expect(error.reason).to eq('NOT_FOUND_ERROR_EVALUATION_REASON')
    end
  end

  describe '#evaluate_variant failure' do
    it 'gracefully handles failures for variant flag evaluation' do
      expect { @client.evaluate_variant(flag_key: 'nonexistent', entity_id: 'someentity', context: { 'fizz' => 'buzz' }) }
        .to raise_error(::Flipt::EvaluationError, 'invalid request: failed to get flag information default/nonexistent')
    end
  end

  describe '#evaluate_boolean failure' do
    it 'gracefully handles failures for boolean flag evaluation' do
      expect { @client.evaluate_boolean(flag_key: 'nonexistent', entity_id: 'someentity', context: { 'fizz' => 'buzz' }) }
        .to raise_error(::Flipt::EvaluationError, 'invalid request: failed to get flag information default/nonexistent')
    end
  end

  describe '#list_flags' do
    it 'returns a list of flags' do
      resp = @client.list_flags
      expect(resp).to_not be_nil
      expect(resp).to include({ 'description' => 'flag description', 'enabled' => true, 'key' => 'flag_boolean', 'type' => 'BOOLEAN_FLAG_TYPE' })
    end
  end

  describe '#snapshot and snapshot restore' do
    it 'can get a snapshot and use it to initialize a new client with fallback' do
      # Get a snapshot from a working client
      snapshot = @client.snapshot
      expect(snapshot).not_to be_nil
      expect(snapshot).to be_a(String)
      expect(snapshot.length).to be > 0

      # Create a client with the previous snapshot and an invalid URL
      invalid_url = 'http://invalid.flipt.com'
      fallback_client = Flipt::Client.new(
        url: invalid_url,
        error_strategy: :fallback,
        snapshot: snapshot
      )

      3.times do
        variant = fallback_client.evaluate_variant(flag_key: 'flag1', entity_id: 'someentity', context: { 'fizz' => 'buzz' })
        expect(variant.flag_key).to eq('flag1')
        expect(variant.match).to eq(true)
        expect(variant.reason).to eq('MATCH_EVALUATION_REASON')
        expect(variant.variant_key).to eq('variant1')
        expect(variant.segment_keys).to eq(['segment1'])

        boolean = fallback_client.evaluate_boolean(flag_key: 'flag_boolean', entity_id: 'someentity', context: { 'fizz' => 'buzz' })
        expect(boolean.flag_key).to eq('flag_boolean')
        expect(boolean.enabled).to eq(true)
        expect(boolean.reason).to eq('MATCH_EVALUATION_REASON')

        flags = fallback_client.list_flags
        expect(flags).to_not be_nil
        expect(flags).to include({ 'description' => 'flag description', 'enabled' => true, 'key' => 'flag_boolean', 'type' => 'BOOLEAN_FLAG_TYPE' })

        new_snapshot = fallback_client.snapshot
        expect(new_snapshot).not_to be_nil
        expect(new_snapshot).to be_a(String)
        expect(new_snapshot.length).to be > 0
      end
    end
  end
end
