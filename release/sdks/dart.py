from .base import SDK
import os
import yaml


class DartSDK(SDK):
    def get_current_version(self) -> str:
        pubspec_yaml = os.path.join(self.path, "pubspec.yaml")
        with open(pubspec_yaml, "r") as f:
            data = yaml.safe_load(f)
        return data["version"]

    def update_version(self, new_version: str):
        pubspec_yaml = os.path.join(self.path, "pubspec.yaml")
        with open(pubspec_yaml, "r") as f:
            data = yaml.safe_load(f)
        data["version"] = new_version
        with open(pubspec_yaml, "w") as f:
            yaml.dump(data, f)
        print(f"Updated {self.name} version to {new_version} in pubspec.yaml")
