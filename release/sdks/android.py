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

        # Find the original quote character used for version
        version_match = re.search(r'version\s*=\s*(["\'])([^"\']+)\1', content)
        if not version_match:
            raise ValueError("Could not find version with quotes in build.gradle")
        version_quote = version_match.group(1)

        # First update version
        updated_content = re.sub(
            r'version\s*=\s*["\'][^"\']+["\']',
            f'version = {version_quote}{new_version}{version_quote}',
            content
        )

        # Find the original quote character used for versionName
        version_name_match = re.search(r'versionName\s+(["\'])([^"\']+)\1', updated_content)
        if version_name_match:
            version_name_quote = version_name_match.group(1)
            # Second update versionName
            updated_content = re.sub(
                r'versionName\s+["\'][^"\']+["\']',
                f'versionName {version_name_quote}{new_version}{version_name_quote}',
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
