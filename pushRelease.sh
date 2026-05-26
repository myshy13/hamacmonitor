#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# 1. Check if a version argument was provided
if [ -z "$1" ]; then
    echo "❌ Error: No version provided."
    echo "Usage: ./release.sh v1.0.0"
    exit 1
fi

VERSION=$1
echo "Creating commit automatically"
git add .
git commit --allow-empty -m "Push release for version: $VERSION"


# 2. Optional: Ensure you're on main before tagging
echo "🔄 Checking out main branch and pulling latest changes..."
git checkout main
git pull origin main

# 3. Create the tag locally
echo "🏷️  Creating local tag: $VERSION"
git tag "$VERSION"

# 4. Push the tag to GitHub (this triggers your release workflow)
echo "🚀 Pushing tag $VERSION to GitHub..."
git push origin "$VERSION"

echo "🎉 Success! Check your GitHub Actions tab to see the release building."