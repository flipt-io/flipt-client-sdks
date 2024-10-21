import os
import re
from .base import MuslSupportSDK


class JavaSDK(MuslSupportSDK):
    def get_current_version(self):
        return self._get_version_from_file("build.gradle")

    def get_current_musl_version(self):
        return self._get_version_from_file("build.musl.gradle")

    def _get_version_from_file(self, filename):
        with open(os.path.join(self.path, filename), "r") as f:
            content = f.read()
            match = re.search(r"version\s*=\s*'([\d.]+)'", content)
            if match:
                return match.group(1)
        raise ValueError(f"Version not found in {filename}")

    def update_version(self, new_version):
        self._update_version_in_file("build.gradle", new_version)

    def update_musl_version(self, new_version):
        self._update_version_in_file("build.musl.gradle", new_version)

    def _update_version_in_file(self, filename, new_version):
        file_path = os.path.join(self.path, filename)
        with open(file_path, "r") as f:
            content = f.read()

        updated_content = re.sub(
            r"(version\s*=\s*')[\d.]+'", f"\\g<1>{new_version}'", content
        )

        with open(file_path, "w") as f:
            f.write(updated_content)
