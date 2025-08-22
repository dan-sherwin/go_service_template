#!/bin/bash

FILE="cmd/app/consts/consts.go"

# Extract the current version using awk
CURRENT_VERSION=$(awk -F'"' '/VERSION/ {print $2}' "$FILE")

if [ -z "$CURRENT_VERSION" ]; then
  echo "Error: Current version not found."
  exit 1
fi

# Extract the number after the hyphen and increment it
PREFIX=$(echo "$CURRENT_VERSION" | sed -E 's/-[0-9]+$//')
SUFFIX=$(echo "$CURRENT_VERSION" | grep -oE '[0-9]+$')

NEW_SUFFIX=$((SUFFIX + 1))
NEW_VERSION="${PREFIX}-${NEW_SUFFIX}"

# Replace the old version with the new one using sed
sed -i '' -E "s|VERSION = \"$CURRENT_VERSION\"|VERSION = \"$NEW_VERSION\"|" "$FILE"

echo "Version updated to $NEW_VERSION"