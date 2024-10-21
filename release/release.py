import os
import json
import re
import yaml
import toml
import argparse
import subprocess
from abc import ABC, abstractmethod
from semver import VersionInfo
from prompt_toolkit import prompt
from prompt_toolkit.shortcuts import checkboxlist_dialog, radiolist_dialog
from colorama import init, Fore, Back, Style

# Initialize colorama
init(autoreset=True)

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
            subprocess.run(['git', 'tag', tag], check=True)
            subprocess.run(['git', 'push', 'origin', tag], check=True)
            print(f"Created and pushed tag: {tag}")
        except subprocess.CalledProcessError as e:
            print(f"Error during git operations: {e}")

class NodeSDK(SDK):
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

class RubySDK(SDK):
    def get_current_version(self) -> str:
        version_rb = os.path.join(self.path, 'lib', 'flipt_client', 'version.rb')
        with open(version_rb, 'r') as f:
            content = f.read()
        return re.search(r'VERSION\s*=\s*["\'](.+)["\']', content).group(1)

    def update_version(self, new_version: str):
        version_rb = os.path.join(self.path, 'lib', 'flipt_client', 'version.rb')
        with open(version_rb, 'r') as f:
            content = f.read()
        updated_content = re.sub(r'VERSION\s*=\s*["\'].*["\']', f'VERSION = "{new_version}"', content)
        with open(version_rb, 'w') as f:
            f.write(updated_content)
        print(f"Updated {self.name} version to {new_version} in version.rb")

class PythonSDK(SDK):
    def get_current_version(self) -> str:
        pyproject_toml = os.path.join(self.path, 'pyproject.toml')
        data = toml.load(pyproject_toml)
        return data['tool']['poetry']['version']

    def update_version(self, new_version: str):
        pyproject_toml = os.path.join(self.path, 'pyproject.toml')
        data = toml.load(pyproject_toml)
        data['tool']['poetry']['version'] = new_version
        with open(pyproject_toml, 'w') as f:
            toml.dump(data, f)
        print(f"Updated {self.name} version to {new_version} in pyproject.toml")

class JavaSDK(SDK):
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

class DartSDK(SDK):
    def get_current_version(self) -> str:
        pubspec_yaml = os.path.join(self.path, 'pubspec.yaml')
        with open(pubspec_yaml, 'r') as f:
            data = yaml.safe_load(f)
        return data['version']

    def update_version(self, new_version: str):
        pubspec_yaml = os.path.join(self.path, 'pubspec.yaml')
        with open(pubspec_yaml, 'r') as f:
            data = yaml.safe_load(f)
        data['version'] = new_version
        with open(pubspec_yaml, 'w') as f:
            yaml.dump(data, f)
        print(f"Updated {self.name} version to {new_version} in pubspec.yaml")

class GoSDK(SDK):
    def get_current_version(self) -> str:
        try:
            result = subprocess.run(['git', 'describe', '--tags', '--abbrev=0', '--match', f'{self.name}-v*'],
                                    capture_output=True, text=True, check=True)
            latest_tag = result.stdout.strip()
            return latest_tag.split('-v')[-1]
        except subprocess.CalledProcessError:
            print(f"{Fore.YELLOW}Warning: No tags found for {self.name}. Using 0.0.0 as the base version.{Style.RESET_ALL}")
            return "0.0.0"

    def update_version(self, new_version: str):
        # Go doesn't typically update version in files, so we just print a message
        print(f"{Fore.YELLOW}Note: Version for {self.name} is managed via Git tags. No files updated.{Style.RESET_ALL}")

def get_sdk(name: str, path: str) -> SDK:
    sdk_classes = {
        'flipt-client-go': GoSDK,
        'flipt-client-java': JavaSDK,
        'flipt-client-node': NodeSDK,
        'flipt-client-browser': NodeSDK,
        'flipt-client-react': NodeSDK,
        'flipt-client-dart': DartSDK,
        'flipt-client-python': PythonSDK,
        'flipt-client-ruby': RubySDK
    }
    return sdk_classes[name](name, path)

def bump_version(version: str, bump_type: str) -> str:
    v = VersionInfo.parse(version)
    if bump_type == 'major':
        return str(v.bump_major())
    elif bump_type == 'minor':
        return str(v.bump_minor())
    else:
        return str(v.bump_patch())

def update_sdk_versions(bump_type='patch', sdks_to_update=None):
    all_sdk_dirs = [
        'flipt-client-go', 'flipt-client-java', 'flipt-client-node',
        'flipt-client-browser', 'flipt-client-react', 'flipt-client-dart',
        'flipt-client-python', 'flipt-client-ruby'
    ]

    sdk_dirs = sdks_to_update if sdks_to_update else all_sdk_dirs
    updated_versions = {}

    for sdk_dir in sdk_dirs:
        if sdk_dir not in all_sdk_dirs:
            print(f"Warning: {sdk_dir} is not a recognized SDK. Skipping...")
            continue

        sdk_path = os.path.join('..', sdk_dir)
        if not os.path.exists(sdk_path):
            print(f"Warning: {sdk_dir} not found. Skipping...")
            continue

        print(f"\nProcessing {sdk_dir}...")

        sdk = get_sdk(sdk_dir, sdk_path)
        current_version = sdk.get_current_version()
        new_version = bump_version(current_version, bump_type)
        sdk.update_version(new_version)
        updated_versions[sdk_dir] = new_version

        print(f"{Fore.CYAN}Finished processing {sdk_dir}{Style.RESET_ALL}")

    return updated_versions

def tag_and_push_versions(versions):
    for sdk_dir, new_version in versions.items():
        sdk_path = os.path.join('..', sdk_dir)
        sdk = get_sdk(sdk_dir, sdk_path)
        sdk.tag_and_push(new_version)

def get_sdk_selection(all_sdk_dirs):
    selected_sdks = checkboxlist_dialog(
        title="Select SDKs to release",
        text="Choose the SDKs you want to release (Space to select, Enter to confirm):",
        values=[(sdk, sdk) for sdk in all_sdk_dirs] + [('all', 'All SDKs')],
    ).run()
    
    if 'all' in selected_sdks:
        return all_sdk_dirs
    return selected_sdks

def get_bump_type():
    bump_type = radiolist_dialog(
        title="Select version bump type",
        text="Choose the type of version bump:",
        values=[
            ("patch", "Patch (0.0.X)"),
            ("minor", "Minor (0.X.0)"),
            ("major", "Major (X.0.0)"),
        ],
    ).run()
    return bump_type

def get_action():
    action = radiolist_dialog(
        title="Select action",
        text="Choose the action to perform:",
        values=[
            ("update", "Update versions"),
            ("push", "Tag and push"),
            ("both", "Update versions and push"),
        ],
    ).run()
    return action

def main():
    all_sdk_dirs = [
        'flipt-client-go', 'flipt-client-java', 'flipt-client-node',
        'flipt-client-browser', 'flipt-client-react', 'flipt-client-dart',
        'flipt-client-python', 'flipt-client-ruby'
    ]

    selected_sdks = get_sdk_selection(all_sdk_dirs)
    if not selected_sdks:
        print("No SDKs selected. Exiting.")
        return

    bump_type = get_bump_type()
    if not bump_type:
        print("No bump type selected. Exiting.")
        return

    action = get_action()
    if not action:
        print("No action selected. Exiting.")
        return

    updated_versions = {}

    if action in ['update', 'both']:
        updated_versions = update_sdk_versions(bump_type, selected_sdks)
    
    if action in ['push', 'both']:
        if action == 'push':
            # If only pushing, we need to get the current versions
            for sdk_dir in selected_sdks:
                sdk_path = os.path.join('..', sdk_dir)
                sdk = get_sdk(sdk_dir, sdk_path)
                updated_versions[sdk_dir] = sdk.get_current_version()
        tag_and_push_versions(updated_versions)

    print(f"{Fore.GREEN}Action '{action}' completed successfully.{Style.RESET_ALL}")

if __name__ == "__main__":
    main()
