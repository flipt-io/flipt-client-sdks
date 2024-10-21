import os
import toml
from .base import SDK


class PythonSDK(SDK):
    def get_current_version(self) -> str:
        pyproject_toml = os.path.join(self.path, "pyproject.toml")
        data = toml.load(pyproject_toml)
        return data["tool"]["poetry"]["version"]

    def update_version(self, new_version: str):
        pyproject_toml = os.path.join(self.path, "pyproject.toml")
        data = toml.load(pyproject_toml)
        data["tool"]["poetry"]["version"] = new_version
        with open(pyproject_toml, "w") as f:
            toml.dump(data, f)
        print(f"Updated {self.name} version to {new_version} in pyproject.toml")
