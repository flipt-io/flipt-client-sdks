import os
import re
from .base import SDK


class CSharpSDK(SDK):
    def get_current_version(self):
        return self._get_version_from_file("src/FliptClient/FliptClient.csproj")

    def _get_version_from_file(self, filename):
        with open(os.path.join(self.path, filename), "r") as f:
            content = f.read()
            # <Version>0.0.1</Version>
            match = re.search(r"<Version>([\d.]+)</Version>", content)
            if match:
                return match.group(1)
        raise ValueError(f"Version not found in {filename}")

    def update_version(self, new_version):
        self._update_version_in_file("src/FliptClient/FliptClient.csproj", new_version)

    def _update_version_in_file(self, filename, new_version):
        file_path = os.path.join(self.path, filename)
        with open(file_path, "r") as f:
            content = f.read()

        updated_content = re.sub(
            r"<Version>([\d.]+)</Version>", f"<Version>{new_version}</Version>", content
        )

        with open(file_path, "w") as f:
            f.write(updated_content)
