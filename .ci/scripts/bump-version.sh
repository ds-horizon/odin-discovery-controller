#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "${SCRIPT_DIR}")")"
# Extract current version from version.go
RELEASE_VERSION=$(grep 'const Version = "v' "${PROJECT_ROOT}/version.go" | sed -E 's/.*const Version = "v(.*)"/\1/')

if [ -z "${RELEASE_VERSION}" ]; then
	echo "Error: Could not extract version from version.go"
	exit 1
fi

echo "Current release version: v${RELEASE_VERSION}"

# Parse version components (assuming semantic versioning: MAJOR.MINOR.PATCH)
IFS='.' read -ra VERSION_PARTS <<<"${RELEASE_VERSION}"
MAJOR="${VERSION_PARTS[0]}"
MINOR="${VERSION_PARTS[1]}"
PATCH="${VERSION_PARTS[2]}"

# Bump patch version by default
PATCH=$((PATCH + 1))
BUMP_VERSION="${MAJOR}.${MINOR}.${PATCH}"

echo "Bumping version to: v${BUMP_VERSION}"

# Update version.go
sed -i.bak "s/const Version = \"v.*\"/const Version = \"v${BUMP_VERSION}\"/" "${PROJECT_ROOT}/version.go"
rm -f "${PROJECT_ROOT}/version.go.bak"

# Stage changes
git add "${PROJECT_ROOT}/version.go"

# Commit and push
git commit -m "chore: release version v${RELEASE_VERSION} and bump version to v${BUMP_VERSION}"
git push origin HEAD:master

echo "Successfully released v${RELEASE_VERSION} and bumped version to v${BUMP_VERSION}"
