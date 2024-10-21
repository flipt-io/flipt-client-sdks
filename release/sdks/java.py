import os
import re
from .base import MuslSupportSDK

class JavaSDK(MuslSupportSDK):
    def get_current_version(self) -> str:
        gradle_files = ['build.gradle', 'build.gradle.kts']
        for gradle_file in gradle_files:
            gradle_path = os.path.join(self.path, gradle_file)
            if os.path.exists(gradle_path):
                with open(gradle_path, 'r') as f:
                    content = f.read()
                return re.search(r'version\s*=\s*["\'](.+)["\']', content).group(1)
        raise FileNotFoundError("No Gradle file found")

    def update_version(self, new_version: str):
        gradle_files = ['build.gradle', 'build.gradle.kts']
        for gradle_file in gradle_files:
            gradle_path = os.path.join(self.path, gradle_file)
            if os.path.exists(gradle_path):
                with open(gradle_path, 'r') as f:
                    content = f.read()
                updated_content = re.sub(r'version\s*=\s*["\'].*["\']', f'version = "{new_version}"', content)
                with open(gradle_path, 'w') as f:
                    f.write(updated_content)
                print(f"Updated {self.name} version to {new_version} in {gradle_file}")

    def get_current_musl_version(self) -> str:
        gradle_files = ['build.musl.gradle', 'build.musl.gradle.kts']
        for gradle_file in gradle_files:
            gradle_path = os.path.join(self.path, gradle_file)
            if os.path.exists(gradle_path):
                with open(gradle_path, 'r') as f:
                    content = f.read()
                return re.search(r'version\s*=\s*["\'](.+)["\']', content).group(1)
        raise FileNotFoundError("No MUSL Gradle file found")

    def update_musl_version(self, new_version: str):
        musl_gradle_files = ['build.musl.gradle', 'build.musl.gradle.kts']
        for gradle_file in musl_gradle_files:
            gradle_path = os.path.join(self.path, gradle_file)
            if os.path.exists(gradle_path):
                with open(gradle_path, 'r') as f:
                    content = f.read()
                updated_content = re.sub(r'version\s*=\s*["\'].*["\']', f'version = "{new_version}"', content)
                with open(gradle_path, 'w') as f:
                    f.write(updated_content)
                print(f"Updated MUSL {self.name} version to {new_version} in {gradle_file}")