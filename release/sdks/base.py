from abc import ABC, abstractmethod
import subprocess


class SDK(ABC):
    def __init__(self, name, path):
        self.name = name
        self.path = path

    @abstractmethod
    def get_current_version(self) -> str:
        pass

    @abstractmethod
    def update_version(self, new_version: str):
        pass

    def tag_and_push(self, new_version: str):
        tag = f"{self.name}-v{new_version}"
        try:
            subprocess.run(["git", "tag", tag], check=True)
            subprocess.run(["git", "push", "origin", tag], check=True)
            print(f"Created and pushed tag: {tag}")
        except subprocess.CalledProcessError as e:
            print(f"Error during git operations: {e}")


class MuslSupportSDK(SDK):
    @abstractmethod
    def get_current_musl_version(self) -> str:
        pass

    @abstractmethod
    def update_musl_version(self, new_version: str):
        pass

    def tag_and_push_musl(self, new_version: str):
        musl_tag = f"{self.name}-musl-v{new_version}"
        try:
            subprocess.run(["git", "tag", musl_tag], check=True)
            subprocess.run(["git", "push", "origin", musl_tag], check=True)
            print(f"Created and pushed MUSL tag: {musl_tag}")
        except subprocess.CalledProcessError as e:
            print(f"Error during MUSL git operations: {e}")
