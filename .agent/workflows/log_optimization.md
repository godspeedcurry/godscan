---
description: Manage and optimize log output consistency and file handling
---
1. Review Log Configuration
   - Check `utils/logs.go`.
   - Verify log levels (DEBUG, INFO, ERROR) are respected.

2. Clean up Noise
   - Identify redundant logs in `cmd/*.go`.
   - Move verbose debugging info to `utils.Debug()`.

3. Standardize Output Paths
   - Ensure logs are written to `output/[date]/[target]/`.
   - Avoid creating files in project root.

4. Check Log Formatting
   - Ensure timestamps are present in file logs but clean in console logs.
   - Verify color stripping for file outputs.
