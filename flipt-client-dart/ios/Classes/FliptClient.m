#import <Flutter/Flutter.h>

// Declare the symbol from your static library
extern void initialize_engine(void);

@interface FliptClient : NSObject<FlutterPlugin>
@end

@implementation FliptClient

// Dummy function to force the linker to include initialize_engine
__attribute__((used))
static void force_link_initialize_engine(void) {
    // This call is never actually executed, but it forces the linker to include the symbol
    initialize_engine();
}

+ (void)registerWithRegistrar:(NSObject<FlutterPluginRegistrar>*)registrar {
  // No-op: This is just to ensure the framework is bundled and plugin is registered.
}

@end