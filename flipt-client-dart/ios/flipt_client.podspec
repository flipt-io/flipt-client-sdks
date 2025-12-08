Pod::Spec.new do |s|
  s.name             = 'flipt_client'
  s.version          = '1.1.1' # must match pubspec.yaml
  s.summary          = 'Dart FFI bindings for Flipt engine'
  s.description      = 'Precompiled Flipt engine as a static .xcframework for use via Dart FFI.'
  s.homepage         = 'https://github.com/flipt-io/flipt-client-sdks'
  s.license          = { :file => 'LICENSE' }
  s.author           = { 'Flipt Devs' => 'devs@flipt.io' }
  s.source           = { :path => '.' }
  s.dependency 'Flutter'

  s.platform         = :ios, '11.0'
  s.vendored_frameworks = 'FliptEngineFFI.xcframework'
  s.source_files = 'Classes/**/*.{h,m}'
  s.pod_target_xcconfig = { 'EXCLUDED_ARCHS[sdk=iphonesimulator*]' => 'x86_64' }
end
