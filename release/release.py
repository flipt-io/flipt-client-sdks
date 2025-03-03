import os
import argparse
from semver import VersionInfo
from prompt_toolkit.shortcuts import radiolist_dialog, yes_no_dialog, input_dialog
from colorama import init, Fore, Style
from sdks import GoSDK, JavaSDK, JavaScriptSDK, RubySDK, PythonSDK, DartSDK, SwiftSDK, CSharpSDK
from sdks.base import SDK, TagBasedSDK

# Initialize colorama
init(autoreset=True)

def get_sdk(name: str) -> SDK:
    sdk_classes = {
        "flipt-client-go": GoSDK,
        "flipt-client-java": JavaSDK,
        "flipt-client-node": JavaScriptSDK,
        "flipt-client-browser": JavaScriptSDK,
        "flipt-client-react": JavaScriptSDK,
        "flipt-client-dart": DartSDK,
        "flipt-client-python": PythonSDK,
        "flipt-client-ruby": RubySDK,
        "flipt-client-swift": SwiftSDK,
        "flipt-client-csharp": CSharpSDK,
    }
    return sdk_classes[name](name)


def bump_version(version: str, bump_type: str) -> str:
    v = VersionInfo.parse(version)
    if bump_type == "major":
        return str(v.bump_major())
    elif bump_type == "minor":
        return str(v.bump_minor())
    else:
        return str(v.bump_patch())


def update_sdk_version(bump_type="patch", sdk=None):
    if not sdk:
        print("No SDK specified. Exiting...")
        return {}

    print(f"\nProcessing {sdk}...")
    
    sdk_instance = get_sdk(sdk)
    current_version = sdk_instance.get_current_version()
    new_version = bump_version(current_version, bump_type)
    sdk_instance.update_version(new_version)
    
    # Return dictionary with SDK name as key, not the SDK instance
    updated_versions = {sdk: new_version}

    print(f"{Fore.CYAN}Finished processing {sdk}{Style.RESET_ALL}")

    return updated_versions


def tag_and_push_version(versions, opts: dict):
    for sdk, new_version in versions.items():
        sdk_instance = get_sdk(sdk)
        if opts.get("dry_run"):
            print(f"{Fore.YELLOW}[DRY RUN] Would tag and push version {new_version} for {sdk}{Style.RESET_ALL}")
            return
        sdk_instance.tag_and_push(new_version, opts)


def get_sdk_selection() -> str | None:
    sdk_display_names = {
        "flipt-client-go": "Go",
        "flipt-client-java": "Java",
        "flipt-client-node": "Node.js",
        "flipt-client-browser": "Browser",
        "flipt-client-react": "React",
        "flipt-client-dart": "Dart",
        "flipt-client-python": "Python",
        "flipt-client-ruby": "Ruby",
        "flipt-client-swift": "Swift",
        "flipt-client-csharp": "C#",
    }

    selected_sdk = radiolist_dialog(
        title="Select SDK to release",
        text="Choose the SDK you want to release:",
        values=[(sdk, display) for sdk, display in sdk_display_names.items()],
    ).run()

    if selected_sdk is None:
        return None  # User cancelled the selection
    return selected_sdk


def get_bump_type():
    bump_type = radiolist_dialog(
        title="Select version bump type",
        text="Choose the type of version bump:",
        values=[
            ("patch", "Patch (0.0.X)"),
            ("minor", "Minor (0.X.0)"),
            ("major", "Major (X.0.0)"),
            ("manual", "Manual version input"),
        ],
    ).run()

    if bump_type == "manual":
        version = input_dialog(
            title="Enter version",
            text="Enter the new version (e.g. 1.2.3):",
        ).run()
        if version:
            try:
                # Validate the version format
                VersionInfo.parse(version)
                return version
            except ValueError:
                print(f"{Fore.RED}Invalid version format. Please use semantic versioning (e.g. 1.2.3){Style.RESET_ALL}")
                return None
        return None

    return bump_type

def get_pull_request():
    pull_request = yes_no_dialog(
        title="Do you want to create a pull request?",
    ).run()
    return pull_request

def main():
    # Add argument parser
    parser = argparse.ArgumentParser(description='Release SDK versions')
    parser.add_argument('--dry-run', action='store_true', help='Show what would happen without making any changes')
    args = parser.parse_args()

    selected_sdk = get_sdk_selection()
    if selected_sdk is None:
        print("SDK selection cancelled. Exiting.")
        return

    sdk_instance = get_sdk(selected_sdk)
    current_version = sdk_instance.get_current_version()

    bump_type = get_bump_type()
    if bump_type is None:
        print("Version selection cancelled. Exiting.")
        return

    # Determine the new version
    if isinstance(bump_type, str) and bump_type in ["major", "minor", "patch"]:
        new_version = bump_version(current_version, bump_type)
    else:
        new_version = bump_type  # Manual version input

    if args.dry_run:
        print(f"{Fore.YELLOW}[DRY RUN] Would update {selected_sdk} from {current_version} to {new_version}{Style.RESET_ALL}")
    else:
        sdk_instance.update_version(new_version)
        print(f"{Fore.CYAN}Updated {selected_sdk} to version {new_version}{Style.RESET_ALL}")
        
    # Check if the SDK supports pull requests
    create_pull_request = False if isinstance(sdk_instance, TagBasedSDK) else get_pull_request()
    # Check if we should push straight to main
    push_to_main = not (isinstance(sdk_instance, TagBasedSDK) or create_pull_request)
    
    # Add confirmation prompt for tag
    tag_confirmation = yes_no_dialog(
        title="Confirm Tag Creation",
        text=f"This will create and push tag v{new_version} for {selected_sdk}.\nDo you want to continue?"
    ).run()
    
    if not tag_confirmation:
        print("Tag creation cancelled. Exiting.")
        return

    opts = {
        "create_pull_request": create_pull_request,
        "push_to_main": push_to_main,
        "dry_run": args.dry_run
    }
        
    tag_and_push_version({selected_sdk: new_version}, opts)

    if args.dry_run:
        print(f"{Fore.GREEN}Dry run completed successfully. No changes were made.{Style.RESET_ALL}")
    else:
        print(f"{Fore.GREEN}Release completed successfully.{Style.RESET_ALL}")


if __name__ == "__main__":
    main()
