#!/bin/bash

# This script updates the README.md file in a specified 'flipt-client-<name>' directory with the current UTC date and time on the first line.
# It optionally takes a semantic version number to commit the changes with a specific message format for release purposes.
# If a version is provided, it stages and commits the file with the message "chore: release as {version}" and "Release-As: {version}".
#
# Usage: ./bump.sh <name> [version]
#
# This is used as a workaround for `release-please` to be able to create a release PR for the client libraries independently
# See: https://github.com/googleapis/release-please/issues/1905

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "Usage: ./bump.sh <name> [version]"
    exit 1
fi

NAME=$1
DIRECTORY="flipt-client-$NAME"
FILE="$DIRECTORY/README.md"
DATE_COMMENT="<!-- Last updated: $(date -u) -->"
VERSION=$2  # Optional version argument

# Create the directory if it doesn't exist
mkdir -p "$DIRECTORY"

# Function to prepend text to a file
prepend() {
    local file=$1
    local text=$2
    local temp_file=$(mktemp)

    echo "$text" > "$temp_file"
    cat "$file" >> "$temp_file"
    mv "$temp_file" "$file"
}

# Check if the README.md exists
if [ -f "$FILE" ]; then
    # Check if the first line contains the date stamp
    if head -n 1 "$FILE" | grep -q "<!-- Last updated:"; then
        # Replace the first line with the new date
        tail -n +2 "$FILE" > "$FILE.tmp" && mv "$FILE.tmp" "$FILE"
        prepend "$FILE" "$DATE_COMMENT"
    else
        # Prepend the date stamp if the first line doesn't contain it
        prepend "$FILE" "$DATE_COMMENT"
    fi
else
    # Create README.md and write the date stamp if the file doesn't exist
    echo "$DATE_COMMENT" > "$FILE"
fi


# Commit changes with a specific message format
if [ -n "$VERSION" ]; then
    # Add the file to the staging area
    git add "$FILE"
    git commit -m "chore: release as $VERSION" -m "Release-As: $VERSION"
fi

echo "Updated $FILE"
