from abc import ABC, abstractmethod
import subprocess
import os
from packaging import version

class SDK(ABC):
    def __init__(self, name):
        self.name = name

    @property
    def path(self) -> str:
        name = self.name.split("-musl")[0]
        return os.path.join("..", name)

    @abstractmethod
    def get_current_version(self) -> str:
        pass

    @abstractmethod
    def update_version(self, new_version: str):
        pass

    def tag_and_push(self, new_version: str, opts: dict):
        tag = f"{self.name}-v{new_version}"
        try:
            if opts["create_pull_request"]:
                subprocess.run(["git", "checkout", "-b", f"release/{tag}"], check=True, cwd=self.path)
                subprocess.run(["git", "commit", "-s", "-a", "--allow-empty", "-m", f"Release {tag}"], check=True, cwd=self.path)
                subprocess.run(
                    ["gh", "pr", "create", "-R", "flipt-io/flipt-client-sdks", "--title", f"Release {tag}", "--body", f"Release {tag}"],
                    cwd=self.path
                )
            else:
                subprocess.run(["git", "commit", "-s", "-a", "--allow-empty", "-m", f"Release {tag}"], check=True, cwd=self.path)
                if opts["push_to_main"]:
                    subprocess.run(["git", "push", "origin", "main"], check=True, cwd=self.path)
            
            # Create and push tag
            subprocess.run(["git", "tag", tag], check=True, cwd=self.path)
            subprocess.run(["git", "push", "origin", tag], check=True, cwd=self.path)

            print(f"Created and pushed tag: {tag}")

        except subprocess.CalledProcessError as e:
            print(f"Error during git operations: {e}")

class TagBasedSDK(SDK):
    def get_current_version(self) -> str:
        try:
            # Get all tags matching the SDK name pattern
            result = subprocess.run(
                ["git", "tag", "--list", f"{self.name}-v*"],
                check=True,
                cwd=self.path,
                capture_output=True,
                text=True
            )
            
            # Parse versions from tags and sort them
            versions = []
            for tag in result.stdout.splitlines():
                # Extract version number from tag (e.g., "flipt-client-swift-v1.0.0" -> "1.0.0")
                version_str = tag.split('-v')[-1]
                versions.append(version_str)
            
            # Return the latest version, or "0.0.0" if no tags exist
            found = "0.0.0"
            if versions:
                # Sort versions using packaging.version for proper semver comparison
                found = str(max(version.parse(v) for v in versions))

            return found
        except subprocess.CalledProcessError:
            return "0.0.0"  # Return default version if git command fails
    
    def update_version(self, new_version: str):
        # No file updates needed for tag-based versioning
        pass
