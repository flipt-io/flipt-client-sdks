import os
import re
from .base import SDK


class RubySDK(SDK):
    def get_current_version(self) -> str:
        version_rb = os.path.join(self.path, "lib", "flipt_client", "version.rb")
        with open(version_rb, "r") as f:
            content = f.read()
        return re.search(r'VERSION\s*=\s*["\'](.+)["\']', content).group(1)

    def update_version(self, new_version: str):
        version_rb = os.path.join(self.path, "lib", "flipt_client", "version.rb")
        with open(version_rb, "r") as f:
            content = f.read()
        updated_content = re.sub(
            r'VERSION\s*=\s*["\'].*["\']', f'VERSION = \'{new_version}\'', content
        )
        with open(version_rb, "w") as f:
            f.write(updated_content)
        print(f"Updated {self.name} version to {new_version} in version.rb")

        # Gemfile.lock records the gem's own version under PATH specs, so it goes stale (and
        # bundle install rewrites it) unless it moves with version.rb.
        gemfile_lock = os.path.join(self.path, "Gemfile.lock")
        with open(gemfile_lock) as f:
            content = f.read()
        updated_content, count = re.subn(
            r"^(    flipt_client \()[^)]+(\))$",
            rf"\g<1>{new_version}\g<2>",
            content,
            flags=re.MULTILINE,
        )
        if count != 1:
            raise ValueError("flipt_client version not found in Gemfile.lock")
        with open(gemfile_lock, "w") as f:
            f.write(updated_content)
        print(f"Updated {self.name} version to {new_version} in Gemfile.lock")
