import os
import json
import re
import yaml
import toml
import argparse
import subprocess
from semver import VersionInfo
from prompt_toolkit import prompt
from prompt_toolkit.completion import WordCompleter
from prompt_toolkit.shortcuts import checkboxlist_dialog, radiolist_dialog
from colorama import init, Fore, Back, Style

# Initialize colorama
init(autoreset=True)

def bump_version(version, bump_type='patch'):
    v = VersionInfo.parse(version)
    if bump_type == 'major':
        return str(v.bump_major())
    elif bump_type == 'minor':
        return str(v.bump_minor())
    else:
        return str(v.bump_patch())

def update_package_json(file_path, new_version):
    with open(file_path, 'r') as f:
        data = json.load(f)
    data['version'] = new_version
    with open(file_path, 'w') as f:
        json.dump(data, f, indent=2)

def update_version_rb(file_path, new_version):
    with open(file_path, 'r') as f:
        content = f.read()
    updated_content = re.sub(r'VERSION\s*=\s*["\'].*["\']', f'VERSION = "{new_version}"', content)
    with open(file_path, 'w') as f:
        f.write(updated_content)

def update_pyproject_toml(file_path, new_version):
    data = toml.load(file_path)
    data['tool']['poetry']['version'] = new_version
    with open(file_path, 'w') as f:
        toml.dump(data, f)

def update_gradle_build(file_path, new_version):
    with open(file_path, 'r') as f:
        content = f.read()
    updated_content = re.sub(r'version\s*=\s*["\'].*["\']', f'version = "{new_version}"', content)
    with open(file_path, 'w') as f:
        f.write(updated_content)

def update_pubspec_yaml(file_path, new_version):
    with open(file_path, 'r') as f:
        data = yaml.safe_load(f)
    data['version'] = new_version
    with open(file_path, 'w') as f:
        yaml.dump(data, f)

def git_tag_and_push(sdk_dir, new_version, run=False):
    tag = f"{sdk_dir}-v{new_version}"
    if not run:
        print(f"{Fore.YELLOW}[DRY RUN]{Style.RESET_ALL} Would git add, commit, create tag: {tag}, and push")
    else:
        try:
            # Git add
            subprocess.run(['git', 'add', '.'], check=True)
            print(f"Added changes for {sdk_dir}")
            
            # Git commit
            commit_message = f"chore: Bump {sdk_dir} version to {new_version}"
            subprocess.run(['git', 'commit', '-m', commit_message], check=True)
            print(f"Committed changes: {commit_message}")
            
            # Git tag
            subprocess.run(['git', 'tag', tag], check=True)
            print(f"Created git tag: {tag}")
            
            # Git push commits and tags
            subprocess.run(['git', 'push', 'origin', 'HEAD'], check=True)
            subprocess.run(['git', 'push', 'origin', tag], check=True)
            print(f"Pushed commits and tag: {tag}")
        except subprocess.CalledProcessError as e:
            print(f"Error during git operations: {e}")

def print_version_update(sdk_dir, new_version, file_name=None, run=False):
    sdk_name = f"{Fore.CYAN}{Style.BRIGHT}{sdk_dir}{Style.RESET_ALL}"
    version = f"{Fore.YELLOW}{Style.BRIGHT}{new_version}{Style.RESET_ALL}"
    file_info = f" in {file_name}" if file_name else ""
    
    if run:
        action = f"{Fore.GREEN}Updated{Style.RESET_ALL}"
    else:
        action = f"{Fore.YELLOW}[DRY RUN]{Style.RESET_ALL} Would update"
    
    print(f"{action} {sdk_name} version to {version}{file_info}")

def release_sdk_versions(bump_type='patch', sdks_to_update=None, run=False):
    all_sdk_dirs = [
        'flipt-client-go', 'flipt-client-java', 'flipt-client-node',
        'flipt-client-browser', 'flipt-client-react', 'flipt-client-dart',
        'flipt-client-python', 'flipt-client-ruby'
    ]

    sdk_dirs = sdks_to_update if sdks_to_update else all_sdk_dirs

    for sdk_dir in sdk_dirs:
        if sdk_dir not in all_sdk_dirs:
            print(f"Warning: {sdk_dir} is not a recognized SDK. Skipping...")
            continue

        sdk_path = os.path.join('..', sdk_dir)
        if not os.path.exists(sdk_path):
            print(f"Warning: {sdk_dir} not found. Skipping...")
            continue

        print(f"\nProcessing {sdk_dir}...")

        new_version = None

        if sdk_dir in ['flipt-client-node', 'flipt-client-browser', 'flipt-client-react']:
            package_json = os.path.join(sdk_path, 'package.json')
            if os.path.exists(package_json):
                with open(package_json, 'r') as f:
                    data = json.load(f)
                current_version = data['version']
                new_version = bump_version(current_version, bump_type)
                update_package_json(package_json, new_version)
                print_version_update(sdk_dir, new_version, 'package.json', run)
                if run:
                    os.system(f"cd {sdk_path} && npm install")

        elif sdk_dir == 'flipt-client-ruby':
            version_rb = os.path.join(sdk_path, 'lib', 'flipt_client', 'version.rb')
            if os.path.exists(version_rb):
                with open(version_rb, 'r') as f:
                    content = f.read()
                current_version = re.search(r'VERSION\s*=\s*["\'](.+)["\']', content).group(1)
                new_version = bump_version(current_version, bump_type)
                update_version_rb(version_rb, new_version)
                print_version_update(sdk_dir, new_version, 'version.rb', run)

        elif sdk_dir == 'flipt-client-python':
            pyproject_toml = os.path.join(sdk_path, 'pyproject.toml')
            if os.path.exists(pyproject_toml):
                data = toml.load(pyproject_toml)
                current_version = data['tool']['poetry']['version']
                new_version = bump_version(current_version, bump_type)
                update_pyproject_toml(pyproject_toml, new_version)
                print_version_update(sdk_dir, new_version, 'pyproject.toml', run)

        elif sdk_dir == 'flipt-client-java':
            gradle_files = ['build.gradle', 'build.gradle.kts']
            for gradle_file in gradle_files:
                gradle_path = os.path.join(sdk_path, gradle_file)
                if os.path.exists(gradle_path):
                    with open(gradle_path, 'r') as f:
                        content = f.read()
                    current_version = re.search(r'version\s*=\s*["\'](.+)["\']', content).group(1)
                    new_version = bump_version(current_version, bump_type)
                    update_gradle_build(gradle_path, new_version)
                    print_version_update(sdk_dir, new_version, gradle_file, run)

        elif sdk_dir == 'flipt-client-dart':
            pubspec_yaml = os.path.join(sdk_path, 'pubspec.yaml')
            if os.path.exists(pubspec_yaml):
                with open(pubspec_yaml, 'r') as f:
                    data = yaml.safe_load(f)
                current_version = data['version']
                new_version = bump_version(current_version, bump_type)
                update_pubspec_yaml(pubspec_yaml, new_version)
                print_version_update(sdk_dir, new_version, 'pubspec.yaml', run)

        elif sdk_dir == 'flipt-client-go':
            # For Go, we don't update any files, just tag and push
            go_mod = os.path.join(sdk_path, 'go.mod')
            if os.path.exists(go_mod):
                with open(go_mod, 'r') as f:
                    content = f.read()
                current_version = re.search(r'v(\d+\.\d+\.\d+)', content).group(1)
                new_version = bump_version(current_version, bump_type)
                print_version_update(sdk_dir, new_version, 'go.mod (tag only)', run)
            else:
                print(f"{Fore.YELLOW}Warning: go.mod not found for {sdk_dir}. Skipping...{Style.RESET_ALL}")

        if new_version and run:
            git_tag_and_push(sdk_dir, new_version, run)

        print(f"{Fore.CYAN}Finished processing {sdk_dir}{Style.RESET_ALL}")

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

def print_colored_summary(selected_sdks, bump_type, run_mode):
    print(f"\n{Style.BRIGHT}Selected SDKs:{Style.RESET_ALL} {Fore.CYAN}{', '.join(selected_sdks)}")
    print(f"{Style.BRIGHT}Bump type:{Style.RESET_ALL} {Fore.YELLOW}{bump_type}")
    if run_mode:
        print(f"{Style.BRIGHT}Run mode:{Style.RESET_ALL} {Back.GREEN}{Fore.BLACK} Active {Style.RESET_ALL}")
    else:
        print(f"{Style.BRIGHT}Run mode:{Style.RESET_ALL} {Back.YELLOW}{Fore.BLACK} Dry run {Style.RESET_ALL}")

def main():
    parser = argparse.ArgumentParser(description="Release Flipt client SDKs")
    parser.add_argument("--run", action="store_true", help="Actually perform the changes (default is dry run)")
    args = parser.parse_args()

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

    print_colored_summary(selected_sdks, bump_type, args.run)

    confirm = prompt("Do you want to proceed? (y/n): ")
    if confirm.lower() != 'y':
        print(f"{Fore.RED}Operation cancelled.{Style.RESET_ALL}")
        return

    release_sdk_versions(bump_type, selected_sdks, args.run)

if __name__ == "__main__":
    main()
