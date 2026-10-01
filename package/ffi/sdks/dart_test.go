package sdks

import "testing"

func TestDartSDKSupportsAndroidArmv7(t *testing.T) {
	sdk := &DartSDK{}

	for _, platform := range sdk.SupportedPlatforms() {
		if platform.ID == "Android-armv7" && platform.Target == "armv7-linux-androideabi" {
			return
		}
	}

	t.Fatalf("Dart SDK supported platforms must include Android-armv7/armv7-linux-androideabi for armeabi-v7a support; got %#v", sdk.SupportedPlatforms())
}
