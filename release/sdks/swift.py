import subprocess
from colorama import Fore, Style
from .base import SDK


class SwiftSDK(SDK):
    def get_current_version(self) -> str:
        return self._get_version_from_tag(f"{self.name}-v*")

    def _get_version_from_tag(self, tag_pattern: str) -> str:
        try:
            result = subprocess.run(
                ["git", "describe", "--tags", "--abbrev=0", "--match", tag_pattern],
                capture_output=True,
                text=True,
                check=True,
            )
            latest_tag = result.stdout.strip()
            return latest_tag.split("-v")[-1]
        except subprocess.CalledProcessError:
            print(
                f"{Fore.YELLOW}Warning: No tags found for {tag_pattern}. Using 0.0.0 as the base version.{Style.RESET_ALL}"
            )
            return "0.0.0"

    def update_version(self, new_version: str):
        print(
            f"{Fore.YELLOW}Note: Version for {self.name} is managed via Git tags. No files updated.{Style.RESET_ALL}"
        )