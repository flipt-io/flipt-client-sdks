# frozen_string_literal: true

require_relative 'lib/flipt_client/version'

Gem::Specification.new do |spec|
  spec.name          = 'flipt_client'
  spec.version       = Flipt::VERSION
  spec.authors       = ['Flipt Devs']
  spec.email         = ['dev@flipt.io']
  spec.summary       = 'Flipt Client Evaluation SDK'
  spec.description   = 'Flipt Client Evaluation SDK'
  spec.homepage      = 'https://www.flipt.io'
  spec.license       = 'MIT'
  spec.required_ruby_version = Gem::Requirement.new('>= 2.3.0')

  # spec.metadata['allowed_push_host'] = "TODO: Set to 'http://mygemserver.com'"

  spec.metadata['homepage_uri'] = spec.homepage
  spec.metadata['source_code_uri'] = 'https://github.com/flipt-io/flipt-client-sdks'

  spec.files = Dir.glob('{lib}/**/*') + ['README.md', 'flipt-client-ruby.gemspec']
  spec.bindir        = 'exe'
  spec.executables   = spec.files.grep(%r{^exe/}) { |f| File.basename(f) }
  spec.require_paths = ['lib']
end
