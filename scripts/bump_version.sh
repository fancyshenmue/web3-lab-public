#!/bin/bash

# Usage: ./bump_version.sh [major|minor|patch]

VERSION_FILE="VERSION"

if [ ! -f "$VERSION_FILE" ]; then
    echo "0.0.1" > "$VERSION_FILE"
fi

CURRENT_VERSION=$(cat "$VERSION_FILE")
IFS='.' read -r -a parts <<< "$CURRENT_VERSION"
MAJOR=${parts[0]}
MINOR=${parts[1]}
PATCH=${parts[2]}

case "$1" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    *)
        echo "Current Version: $CURRENT_VERSION"
        exit 0
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"
echo "$NEW_VERSION" > "$VERSION_FILE"

if [ -f "frontend/app/package.json" ]; then
    (cd frontend/app && npm version "$NEW_VERSION" --no-git-tag-version --allow-same-version >/dev/null 2>&1)
fi

echo "Bumped version to $NEW_VERSION"
