---
description: Polish the Command Line Interface (CLI) for better user experience
---
1. Audit Command Flags & Help
   - Run `go run main.go [command] --help`
   - Ensure flags have clear descriptions and shorthand versions.

2. Check Output consistency
   - Use `utils.Info()`, `utils.Success()`, `utils.Warning()` instead of `fmt.Println()`.
   - Ensure color coding is used meaningfully (Green=Good, Red=Bad, Cyan=Info).

3. Optimize Progress Bars
   - Ensure `cheggaaa/pb/v3` is used for long-running tasks.
   - Verify progress bars do not conflict with log output (use separate lines or clear lines).

4. Verify Interactive Prompts
   - Ensure prompts have defaults.
   - Support `Ctrl+C` graceful exit.
