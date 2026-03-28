#!/usr/bin/env python3
"""Non-interactive release script for use by CI or AI agents.

Dry-run by default. Pass --publish to actually release.

Usage:
    python release_cli.py --sdk flipt-client-go --bump patch              # dry-run
    python release_cli.py --sdk flipt-client-go --bump patch --publish    # real release
    python release_cli.py --sdk flipt-client-js --version 1.0.0 --publish
    python release_cli.py --sdk flipt-client-go --bump patch --publish --pr
"""

import argparse
import sys
from semver import VersionInfo
from colorama import init, Fore, Style
from sdks import AndroidSDK, GoSDK, JavaSDK, JavaScriptSDK, RubySDK, PythonSDK, DartSDK, SwiftSDK, CSharpSDK
from sdks.base import SDK, TagBasedSDK

init(autoreset=True)

SDK_CLASSES = {
    "flipt-client-js": JavaScriptSDK,
    "flipt-client-csharp": CSharpSDK,
    "flipt-client-dart": DartSDK,
    "flipt-client-go": GoSDK,
    "flipt-client-java": JavaSDK,
    "flipt-client-kotlin-android": AndroidSDK,
    "flipt-client-python": PythonSDK,
    "flipt-client-react": JavaScriptSDK,
    "flipt-client-ruby": RubySDK,
    "flipt-client-swift": SwiftSDK,
}


def get_sdk(name: str) -> SDK:
    if name not in SDK_CLASSES:
        print(f"{Fore.RED}Unknown SDK: {name}{Style.RESET_ALL}")
        print(f"Available SDKs: {', '.join(sorted(SDK_CLASSES.keys()))}")
        sys.exit(1)
    return SDK_CLASSES[name](name)


def bump_version(version: str, bump_type: str) -> str:
    v = VersionInfo.parse(version)
    if bump_type == "major":
        return str(v.bump_major())
    elif bump_type == "minor":
        return str(v.bump_minor())
    else:
        return str(v.bump_patch())


def main():
    parser = argparse.ArgumentParser(description="Non-interactive SDK release")
    parser.add_argument("--sdk", required=True, help="SDK name (e.g. flipt-client-go)")
    parser.add_argument("--bump", choices=["patch", "minor", "major"], help="Version bump type")
    parser.add_argument("--version", help="Explicit version (e.g. 2.0.0)")
    parser.add_argument("--publish", action="store_true", help="Actually release (default is dry-run)")
    parser.add_argument("--pr", action="store_true", help="Create a pull request instead of pushing to main")
    args = parser.parse_args()

    if not args.bump and not args.version:
        print(f"{Fore.RED}Either --bump or --version is required{Style.RESET_ALL}")
        sys.exit(1)

    if args.bump and args.version:
        print(f"{Fore.RED}Cannot specify both --bump and --version{Style.RESET_ALL}")
        sys.exit(1)

    sdk_instance = get_sdk(args.sdk)
    current_version = sdk_instance.get_current_version()

    if args.version:
        try:
            VersionInfo.parse(args.version)
        except ValueError:
            print(f"{Fore.RED}Invalid version: {args.version}. Use semver (e.g. 1.2.3){Style.RESET_ALL}")
            sys.exit(1)
        new_version = args.version
    else:
        new_version = bump_version(current_version, args.bump)

    print(f"{args.sdk}: {current_version} → {new_version}")

    if not args.publish:
        print(f"{Fore.YELLOW}[DRY RUN] No changes made. Pass --publish to release.{Style.RESET_ALL}")
        return

    sdk_instance.update_version(new_version)
    print(f"{Fore.CYAN}Updated version files.{Style.RESET_ALL}")

    opts = {
        "create_pull_request": args.pr,
        "dry_run": False,
    }

    sdk_instance.tag_and_push(new_version, opts)
    print(f"{Fore.GREEN}Released {args.sdk} v{new_version}{Style.RESET_ALL}")


if __name__ == "__main__":
    main()
