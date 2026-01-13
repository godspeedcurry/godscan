---
description: Maintain and update LLM integration (models, providers, API handling)
---
1. Update Model List
   - Edit `cmd/llm_profile.go`: Update `promptModel` struct/case list.
   - Edit `utils/llm.go`: Update `DefaultLLMModel` if needed.

2. Add/Update Providers
   - Edit `utils/llm.go`.
   - Implement `call[Provider]Name` function.
   - Update dispatch logic in `SummarizeReport`.

3. Configure Request Parameters
   - Check `LLMConfig` struct in `utils/llm.go`.
   - Update `cmd/llm_flags.go` to expose new parameters (e.g. `top_k`, `temperature`).

4. Test Integration
   - Run `go run main.go report --llm-dry-run` to verify request construction.
   - Test with actual key if available.
