#!/bin/bash

# This script updates the README.md file in a specified 'flipt-client-<language>' directory with the current UTC date and time on the first line.
# Usage: ./bump.sh <language|all>
#
# This is used as a workaround for `release-please` to be able to create a release PR for the client libraries independently
# See: https://github.com/googleapis/release-please/issues/1905

#!/bin/bash

# Function to prepend text to a file
prepend() {
    local file=$1
    local text=$2
    local temp_file=$(mktemp)

    echo "$text" > "$temp_file"
    cat "$file" >> "$temp_file"
    mv "$temp_file" "$file"
}

# Function to update a single README.md file
update_readme() {
    local directory=$1
    local file="$directory/README.md"
    local date_comment="<!-- Last published: $(date -u) -->"  # Using UTC date and time

    # Create the directory if it doesn't exist
    mkdir -p "$directory"

    # Check if the README.md exists
    if [ -f "$file" ]; then
        # Check if the first line contains the date stamp
        if head -n 1 "$file" | grep -q "<!-- Last published:"; then
            # Replace the first line with the new date
            tail -n +2 "$file" > "$file.tmp" && mv "$file.tmp" "$file"
        fi
    else
        # Touch the file to ensure it exists for the prepend operation
        touch "$file"
    fi

    # Prepend the date stamp
    prepend "$file" "$date_comment"

    echo "Updated $file"
}

# Check the number of arguments provided
if [ "$#" -lt 1 ] ; then
    echo "Usage: ./bump.sh <language|all>"
    exit 1
fi

NAME=$1
VERSION=$2  # Optional version argument

if [ "$NAME" == "all" ]; then
    # Loop through each directory starting with 'flipt-client-'
    for directory in flipt-client-*/; do
        if [ -d "$directory" ]; then
            update_readme "$directory"
        fi
    done
else
    DIRECTORY="flipt-client-$NAME"
    update_readme "$DIRECTORY"
fi
