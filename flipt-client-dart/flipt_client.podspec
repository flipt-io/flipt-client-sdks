Pod::Spec.new do |s|
  s.name             = 'flipt_client'
  s.version          = '0.9.0-rc.18' # must match pubspec.yaml
  s.summary          = 'Dart FFI bindings for Flipt engine'
  s.description      = 'Precompiled Flipt engine as a static .xcframework for use via Dart FFI.'
  s.homepage         = 'https://github.com/flipt-io/flipt-client-sdks'
  s.license          = { :file => 'LICENSE' }
  s.author           = { 'Flipt Devs' => 'devs@flipt.io' }
  s.source           = { :path => '.' }

  s.platform         = :ios, '11.0'
  s.vendored_frameworks = 'ios/FliptEngineFFI.xcframework'
  s.source_files = 'ios/Classes/**/*'
end
