import os
from semver import VersionInfo
from prompt_toolkit.shortcuts import checkboxlist_dialog, radiolist_dialog
from colorama import init, Fore, Style
from sdks import GoSDK, JavaSDK, JavaScriptSDK, RubySDK, PythonSDK, DartSDK
from sdks.base import SDK, MuslSupportSDK

# Initialize colorama
init(autoreset=True)

def get_sdk(name: str, path: str) -> SDK:
    sdk_classes = {
        'flipt-client-go': GoSDK,
        'flipt-client-java': JavaSDK,
        'flipt-client-node': JavaScriptSDK,
        'flipt-client-browser': JavaScriptSDK,
        'flipt-client-react': JavaScriptSDK,
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
        'flipt-client-go',
        'flipt-client-java',
        'flipt-client-node',
        'flipt-client-browser',
        'flipt-client-react',
        'flipt-client-dart',
        'flipt-client-python',
        'flipt-client-ruby'
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

        if isinstance(sdk, MuslSupportSDK):
            current_musl_version = sdk.get_current_musl_version()
            new_musl_version = bump_version(current_musl_version, bump_type)
            sdk.update_musl_version(new_musl_version)
            updated_versions[f"{sdk_dir}-musl"] = new_musl_version

        print(f"{Fore.CYAN}Finished processing {sdk_dir}{Style.RESET_ALL}")

    return updated_versions

def tag_and_push_versions(versions):
    for sdk_dir, new_version in versions.items():
        sdk_path = os.path.join('..', sdk_dir)
        sdk = get_sdk(sdk_dir, sdk_path)
        sdk.tag_and_push(new_version)

def get_sdk_selection(all_sdk_dirs):
    sdk_display_names = {
        'flipt-client-go': 'Go',
        'flipt-client-java': 'Java',
        'flipt-client-node': 'Node.js',
        'flipt-client-browser': 'Browser',
        'flipt-client-react': 'React',
        'flipt-client-dart': 'Dart',
        'flipt-client-python': 'Python',
        'flipt-client-ruby': 'Ruby'
    }
    
    selected_sdks = checkboxlist_dialog(
        title="Select SDKs to release",
        text="Choose the SDKs you want to release (Space to select, Enter to confirm):",
        values=[(sdk, sdk_display_names.get(sdk, sdk)) for sdk in all_sdk_dirs] + [('all', 'All SDKs')],
    ).run()
    
    if 'all' in selected_sdks:
        return all_sdk_dirs
    return selected_sdks

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

    action = get_action()
    if not action:
        print("No action selected. Exiting.")
        return

    updated_versions = {}

    if action in ['update', 'both']:
        bump_type = get_bump_type()
        if not bump_type:
            print("No bump type selected. Exiting.")
            return
        updated_versions = update_sdk_versions(bump_type, selected_sdks)
    
    if action in ['push', 'both']:
        if action == 'push':
            # If only pushing, we need to get the current versions
            for sdk_dir in selected_sdks:
                sdk_path = os.path.join('..', sdk_dir)
                sdk = get_sdk(sdk_dir, sdk_path)
                updated_versions[sdk_dir] = sdk.get_current_version()
                if isinstance(sdk, MuslSupportSDK):
                    updated_versions[f"{sdk_dir}-musl"] = sdk.get_current_musl_version()
        tag_and_push_versions(updated_versions)

    print(f"{Fore.GREEN}Action '{action}' completed successfully.{Style.RESET_ALL}")

if __name__ == "__main__":
    main()
