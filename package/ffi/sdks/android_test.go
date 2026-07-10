package sdks

import "testing"

func TestAndroidSDKSupportsAndroidArmv7(t *testing.T) {
	sdk := &AndroidSDK{}

	for _, platform := range sdk.SupportedPlatforms() {
		if platform.ID == "Android-armv7" && platform.Target == "armv7-linux-androideabi" {
			return
		}
	}

	t.Fatalf("Android SDK supported platforms must include Android-armv7/armv7-linux-androideabi for armeabi-v7a support; got %#v", sdk.SupportedPlatforms())
}
