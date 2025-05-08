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

        podspec_path = os.path.join(self.path, "flipt_client.podspec")
        with open(podspec_path, "r") as f:
            lines = f.readlines()
        with open(podspec_path, "w") as f:
            for line in lines:
                if line.strip().startswith("s.version"):
                    f.write(f"  s.version          = '{new_version}' # must match pubspec.yaml\n")
                else:
                    f.write(line)
        print(f"Updated flipt_client.podspec version to {new_version}")