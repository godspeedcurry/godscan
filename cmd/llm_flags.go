package cmd

import (
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// LLMCLIOptions collects CLI-level flags for optional LLM summaries.
type LLMCLIOptions struct {
	Prompt   string
	MaxChars int
	DryRun   bool
	Profile  string
	APIKey   string
	BaseURL  string
}

func addLLMFlags(cmd *cobra.Command, target *LLMCLIOptions) {
	if target == nil {
		return
	}
	cmd.Flags().StringVar(&target.Prompt, "llm-prompt", utils.DefaultLLMPrompt, "Prompt used to summarize crawler text")
	cmd.Flags().IntVar(&target.MaxChars, "llm-max-chars", utils.DefaultLLMMaxChars, "Max characters sent to LLM (after trimming)")
	cmd.Flags().BoolVar(&target.DryRun, "llm-dry-run", false, "Do not call LLM; embed the would-be request preview in report for debugging")
	cmd.Flags().StringVar(&target.Profile, "llm-profile", "", "Load provider/model/key from encrypted profile")
	cmd.Flags().StringVar(&target.APIKey, "llm-key", "", "Directly provide API key (skips profile loading)")
	cmd.Flags().StringVar(&target.BaseURL, "llm-base-url", "", "Custom LLM API base URL (e.g. http://localhost:8080/v1)")
}

// ToConfig converts CLI flags + env into runtime config. Returns nil when disabled/no key.
func (o *LLMCLIOptions) ToConfig() *utils.LLMConfig {
	if o == nil {
		return nil
	}
	secret := strings.TrimSpace(o.Profile)
	var profile *utils.LLMProfile

	// Move defaults to top scope
	provider := utils.DefaultLLMProvider
	model := utils.DefaultLLMModel
	key := strings.TrimSpace(o.APIKey)
	baseURL := strings.TrimSpace(o.BaseURL)

	if o.Profile != "" {
		path := utils.DefaultLLMProfilePath(viper.GetString("output-dir"))
		if profs, err := utils.LoadLLMProfiles(path, secret); err == nil {
			for _, p := range profs {
				if p.Name == o.Profile {
					profile = &p
					break
				}
			}
		} else {
			utils.Warning("load llm profile failed (use profile name as secret): %v", err)
		}
	}
	if profile != nil {
		if strings.TrimSpace(profile.Provider) != "" {
			provider = strings.TrimSpace(profile.Provider)
		}
		if strings.TrimSpace(profile.Model) != "" {
			model = strings.TrimSpace(profile.Model)
		}
		if key == "" {
			key = profile.APIKey
		}
		if baseURL == "" {
			baseURL = profile.BaseURL
		}
	}

	enabled := o.DryRun || key != ""
	if !enabled {
		return nil
	}
	if !o.DryRun && key == "" {
		utils.Warning("LLM skipped: no key provided (use --llm-key, --llm-profile or --llm-dry-run)")
		return nil
	}

	prompt := strings.TrimSpace(o.Prompt)
	if prompt == "" {
		prompt = utils.DefaultLLMPrompt
	}
	maxChars := o.MaxChars
	if maxChars <= 0 {
		maxChars = utils.DefaultLLMMaxChars
	}
	if profile != nil {
		utils.Info("LLM profile loaded: name=%s provider=%s model=%s", profile.Name, provider, model)
	} else if key != "" {
		utils.Info("LLM using ephemeral key (provider=%s model=%s)", provider, model)
	}
	return &utils.LLMConfig{
		Provider:      strings.ToLower(provider),
		Model:         model,
		Prompt:        prompt,
		APIKey:        key,
		BaseURL:       baseURL,
		MaxInputChars: maxChars,
		DryRun:        o.DryRun,
	}
}
