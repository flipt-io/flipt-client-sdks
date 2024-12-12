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

    def _create_and_push_tag(self, tag: str):
        try:
            subprocess.run(["git", "checkout", "-b", f"release/{tag}"], check=True, cwd=self.path)
            subprocess.run(["git", "commit", "-a", "--allow-empty", "-m", f"Release {tag}"], check=True, cwd=self.path)
            subprocess.run(["gh", "pr", "create", "--title", f"Release {tag}", "--body", f"Release {tag}"], check=True, cwd=self.path)

            subprocess.run(["git", "tag", tag], check=True, cwd=self.path)
            subprocess.run(["git", "push", "origin", tag], check=True, cwd=self.path)
            print(f"Created and pushed tag: {tag}")
        except subprocess.CalledProcessError as e:
            print(f"Error during git operations: {e}")

    def tag_and_push(self, new_version: str):
        tag = f"{self.name}-v{new_version}"
        self._create_and_push_tag(tag)


class MuslSupportSDK(SDK):
    @abstractmethod
    def get_current_musl_version(self) -> str:
        pass

    @abstractmethod
    def update_musl_version(self, new_version: str):
        pass

    def tag_and_push_musl(self, new_version: str):
        musl_tag = f"{self.name}-musl-v{new_version}"
        self._create_and_push_tag(musl_tag)
