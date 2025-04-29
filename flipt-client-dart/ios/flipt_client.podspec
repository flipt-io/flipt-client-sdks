Pod::Spec.new do |s|
  s.name             = 'flipt_client'
  s.version          = '0.0.1'
  s.summary          = 'Dart FFI bindings for Flipt engine'
  s.description      = 'Dart FFI bindings to the Flipt native engine, including static and dynamic libraries.'
  s.homepage         = 'https://github.com/flipt-io/flipt-client-sdks'
  s.license          = { :file => '../LICENSE' }
  s.author           = { 'Flipt Devs' => 'devs@flipt.io' }
  s.source           = { :path => '.' }

  s.platform         = :ios, '11.0'
  s.vendored_frameworks = 'FliptEngineFFI.xcframework'
  s.preserve_paths   = 'FliptEngineFFI.xcframework'
  s.public_header_files = 'FliptEngineFFI.xcframework/**/*.h'
  s.requires_arc     = false
end
