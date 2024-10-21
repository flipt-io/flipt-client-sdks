import os
import json
import re
import yaml
import toml
import argparse
import subprocess
from semver import VersionInfo

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

def git_tag_and_push(sdk_dir, new_version, dry_run=True):
    tag = f"{sdk_dir}-v{new_version}"
    if dry_run:
        print(f"[DRY RUN] Would create git tag: {tag}")
        print(f"[DRY RUN] Would push git tag: {tag}")
    else:
        try:
            subprocess.run(['git', 'tag', tag], check=True)
            print(f"Created git tag: {tag}")
            subprocess.run(['git', 'push', 'origin', tag], check=True)
            print(f"Pushed git tag: {tag}")
        except subprocess.CalledProcessError as e:
            print(f"Error during git operations: {e}")

def release_sdk_versions(bump_type='patch', sdks_to_update=None, dry_run=True):
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

        print(f"Processing {sdk_dir}...")

        new_version = None

        if sdk_dir in ['flipt-client-node', 'flipt-client-browser', 'flipt-client-react']:
            package_json = os.path.join(sdk_path, 'package.json')
            if os.path.exists(package_json):
                with open(package_json, 'r') as f:
                    data = json.load(f)
                current_version = data['version']
                new_version = bump_version(current_version, bump_type)
                if not dry_run:
                    update_package_json(package_json, new_version)
                print(f"{'[DRY RUN] Would update' if dry_run else 'Updated'} {sdk_dir} version to {new_version}")
                if not dry_run:
                    os.system(f"cd {sdk_path} && npm install")

        elif sdk_dir == 'flipt-client-ruby':
            version_rb = os.path.join(sdk_path, 'lib', 'flipt', 'version.rb')
            if os.path.exists(version_rb):
                with open(version_rb, 'r') as f:
                    content = f.read()
                current_version = re.search(r'VERSION\s*=\s*["\'](.+)["\']', content).group(1)
                new_version = bump_version(current_version, bump_type)
                if not dry_run:
                    update_version_rb(version_rb, new_version)
                print(f"{'[DRY RUN] Would update' if dry_run else 'Updated'} {sdk_dir} version to {new_version}")

        elif sdk_dir == 'flipt-client-python':
            pyproject_toml = os.path.join(sdk_path, 'pyproject.toml')
            if os.path.exists(pyproject_toml):
                data = toml.load(pyproject_toml)
                current_version = data['tool']['poetry']['version']
                new_version = bump_version(current_version, bump_type)
                if not dry_run:
                    update_pyproject_toml(pyproject_toml, new_version)
                print(f"{'[DRY RUN] Would update' if dry_run else 'Updated'} {sdk_dir} version to {new_version}")

        elif sdk_dir == 'flipt-client-java':
            gradle_files = ['build.gradle', 'build.gradle.kts']
            for gradle_file in gradle_files:
                gradle_path = os.path.join(sdk_path, gradle_file)
                if os.path.exists(gradle_path):
                    with open(gradle_path, 'r') as f:
                        content = f.read()
                    current_version = re.search(r'version\s*=\s*["\'](.+)["\']', content).group(1)
                    new_version = bump_version(current_version, bump_type)
                    if not dry_run:
                        update_gradle_build(gradle_path, new_version)
                    print(f"{'[DRY RUN] Would update' if dry_run else 'Updated'} {sdk_dir} version to {new_version} in {gradle_file}")

        elif sdk_dir == 'flipt-client-dart':
            pubspec_yaml = os.path.join(sdk_path, 'pubspec.yaml')
            if os.path.exists(pubspec_yaml):
                with open(pubspec_yaml, 'r') as f:
                    data = yaml.safe_load(f)
                current_version = data['version']
                new_version = bump_version(current_version, bump_type)
                if not dry_run:
                    update_pubspec_yaml(pubspec_yaml, new_version)
                print(f"{'[DRY RUN] Would update' if dry_run else 'Updated'} {sdk_dir} version to {new_version}")

        elif sdk_dir == 'flipt-client-go':
            print(f"Note: Version bumping for {sdk_dir} is not implemented. Please update manually if needed.")

        if new_version:
            git_tag_and_push(sdk_dir, new_version, dry_run)

        print(f"Finished processing {sdk_dir}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Release Flipt client SDKs")
    parser.add_argument("--bump", choices=['patch', 'minor', 'major'], default='patch', help="Version bump type")
    parser.add_argument("--sdks", nargs='*', help="Specific SDKs to update (space-separated). If not provided, all SDKs will be updated.")
    parser.add_argument("--dry-run", action="store_true", help="Perform a dry run without making any changes")
    
    args = parser.parse_args()

    release_sdk_versions(args.bump, args.sdks, args.dry_run)
