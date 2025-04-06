from .base import TagBasedSDK
import os
import re

class GoSDK(TagBasedSDK):
  def get_current_version(self) -> str:
    return super().get_current_version()

  def update_version(self, new_version: str):
    version_file = os.path.join(self.path, "version.go")
    with open(version_file, "r") as f:
      data = f.read()

    # Use regex to ensure we only replace the version string
    pattern = r'const Version = "(.*?)"'
    if not re.search(pattern, data):
      raise ValueError(f"Could not find version string in {version_file}")

    updated_data = re.sub(pattern, f'const Version = "{new_version}"', data)

    with open(version_file, "w") as f:
      f.write(updated_data)