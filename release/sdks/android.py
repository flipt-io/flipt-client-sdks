import os
import re
from .base import SDK

class AndroidSDK(SDK):
    def get_current_version(self):
        f = os.path.join(self.path, "build.gradle")
        if not os.path.exists(f):
            raise ValueError(f"build.gradle file not found in {self.path}")
        with open(f, "r") as f:
            content = f.read()
            match = re.search(r'version\s*=\s*["\']([^"\']+)["\']', content)
            if match:
                return match.group(1)
        raise ValueError(f"Version not found in build.gradle")

    def update_version(self, new_version):
        file_path = os.path.join(self.path, "build.gradle")
        with open(file_path, "r") as f:
            content = f.read()

        # First update version
        updated_content = re.sub(
            r'(version\s*=\s*["\'])[^"\']+["\']', 
            f"\\g<1>{new_version}\"", 
            content
        )

        # Second update versionName
        updated_content = re.sub(
            r'(versionName\s+)["\']([^"\']+)["\']', 
            f"\\g<1>\"{new_version}\"", 
            updated_content
        )

        # Then increment versionCode
        updated_content = re.sub(
            r'(versionCode\s+)(\d+)',
            lambda m: f"{m.group(1)}{int(m.group(2)) + 1}",
            updated_content
        )

        with open(file_path, "w") as f:
            f.write(updated_content)
