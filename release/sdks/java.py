import os
import re
from .base import SDK

class JavaSDK(SDK):
    def get_current_version(self):
        filename = self._get_update_file()
        with open(os.path.join(self.path, filename), "r") as f:
            content = f.read()
            match = re.search(r"version\s*=\s*'([\d.]+)'", content)
            if match:
                return match.group(1)
        raise ValueError(f"Version not found in {filename}")

    def update_version(self, new_version):
        filename = self._get_update_file()
        file_path = os.path.join(self.path, filename)
        with open(file_path, "r") as f:
            content = f.read()

        updated_content = re.sub(
            r"(version\s*=\s*')[\d.]+'", f"\\g<1>{new_version}'", content
        )

        with open(file_path, "w") as f:
            f.write(updated_content)

    def _get_update_file(self) -> str:
        if self.name.endswith("-musl"):
            return "build.musl.gradle"
        return "build.gradle"
