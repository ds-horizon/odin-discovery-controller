#!/usr/bin/env bash

set -e

# Extract current version from version.go
RELEASE_VERSION=$(grep 'const Version = "v' version.go | sed 's/.*const Version = "v\(.*\)"/\1/')

if [ -z "$RELEASE_VERSION" ]; then
  echo "Error: Could not extract version from version.go"
  exit 1
fi

echo "Current release version: v${RELEASE_VERSION}"

# Parse version components (assuming semantic versioning: MAJOR.MINOR.PATCH)
IFS='.' read -ra VERSION_PARTS <<< "$RELEASE_VERSION"
MAJOR="${VERSION_PARTS[0]}"
MINOR="${VERSION_PARTS[1]}"
PATCH="${VERSION_PARTS[2]}"

# Bump patch version by default (can be customized with argument)
BUMP_TYPE="${1:-patch}"

case "$BUMP_TYPE" in
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
    echo "Error: Invalid bump type '$BUMP_TYPE'. Use: major, minor, or patch"
    exit 1
    ;;
esac

BUMP_VERSION="${MAJOR}.${MINOR}.${PATCH}"

echo "Bumping version to: v${BUMP_VERSION}"

# Update version.go
sed -i.bak "s/const Version = \"v.*\"/const Version = \"v${BUMP_VERSION}\"/" version.go
rm -f version.go.bak

# Stage changes
git add version.go

# Commit and push
git commit -m "chore: release version v${RELEASE_VERSION} and bump version to v${BUMP_VERSION}"
git push origin HEAD:master

echo "Successfully released v${RELEASE_VERSION} and bumped version to v${BUMP_VERSION}"
