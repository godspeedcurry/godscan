---
description: Create a new git release tag and push to remote
---
1. Check current status
git status

2. Ask user for version number
// wait-for-input
echo "Enter version tag (e.g. v1.2.5):"

3. Create Tag
git tag -a $USER_INPUT -m "release $USER_INPUT"

4. Push to remote
git push origin $USER_INPUT
