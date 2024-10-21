import os
import json
from .base import SDK

class JavaScriptSDK(SDK):
    def get_current_version(self) -> str:
        package_json = os.path.join(self.path, 'package.json')
        with open(package_json, 'r') as f:
            data = json.load(f)
        return data['version']

    def update_version(self, new_version: str):
        package_json = os.path.join(self.path, 'package.json')
        with open(package_json, 'r') as f:
            data = json.load(f)
        data['version'] = new_version
        with open(package_json, 'w') as f:
            json.dump(data, f, indent=2)
        print(f"Updated {self.name} version to {new_version} in package.json")