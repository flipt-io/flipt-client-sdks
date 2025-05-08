#import <Foundation/Foundation.h>
#import <Flutter/Flutter.h>

@interface FliptClient : NSObject<FlutterPlugin>
@end

@implementation FliptClient
+ (void)registerWithRegistrar:(NSObject<FlutterPluginRegistrar>*)registrar {
  // No-op: This is just to ensure the framework is bundled.
}
@end