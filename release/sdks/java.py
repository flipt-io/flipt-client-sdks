import os
import re
from .base import SDK

class JavaSDK(SDK):
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

        # Find the original quote character used
        match = re.search(r'version\s*=\s*(["\'])([^"\']+)\1', content)
        if not match:
            raise ValueError("Could not find version with quotes in build.gradle")
        
        quote = match.group(1)  # Capture the quote character used
        updated_content = re.sub(
            r'version\s*=\s*["\'][^"\']+["\']', 
            f'version = {quote}{new_version}{quote}', 
            content
        )

        with open(file_path, "w") as f:
            f.write(updated_content)
