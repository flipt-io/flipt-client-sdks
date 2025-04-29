Pod::Spec.new do |s|
    s.name             = 'flipt_client_dart'
    s.version          = '0.0.1'
    s.summary          = 'Dart FFI bindings for Flipt Engine'
    s.source           = { :path => '.' }
  
    s.vendored_frameworks = 'ios/FliptEngineFFI.xcframework'
    s.requires_arc     = false
  end
  