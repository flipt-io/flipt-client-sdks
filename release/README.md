# SDK Release Script

This script automates the process of updating versions and creating tags for multiple SDKs in this project.

![image](./release.png)

## Overview

The release script is designed to:

1. Allow users to select which SDKs to update
2. Choose the type of version bump (patch, minor, or major)
3. Update version numbers in SDK-specific files
4. Create and push Git tags for the new versions

## Requirements

- Python 3.7+
- Required Python packages (install using `pip install -r requirements.txt`):
  - semver
  - PyYAML
  - toml
  - prompt_toolkit
  - colorama

## Setup

```
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

## Usage

Run the script from the command line:

```
python release.py
```

## How It Works

1. **SDK Selection**: The script presents a list of available SDKs and allows the user to select which ones to update.

2. **Version Bump Selection**: The user selects the type of version bump (patch, minor, or major).

3. **Version Update**: For each selected SDK, the script:

   - Reads the current version
   - Calculates the new version based on the bump type
   - Updates the version in the SDK-specific file (e.g., package.json, build.gradle)

4. **Tagging and Pushing**: The script creates Git tags for the new versions and pushes them to the remote repository.

5. **Pull Request**: The script optionally creates a pull request with the versioning changes.

## Notes

- The script assumes it's run from the `release` directory within the project structure.
- It interacts with Git, so make sure you have the necessary permissions to create and push tags.

For more detailed information on each SDK's specific implementation, refer to the individual SDK class files in the `sdks` directory.
