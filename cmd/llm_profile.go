package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/godspeedcurry/godscan/utils"
	"github.com/spf13/cobra"
)

var (
	profileName        string
	profileKey         string
	profileModel       string
	profileProvider    string
	profileBaseURL     string
	profileDelete      string
	profileList        bool
	profileInteractive bool
)

// llmCmd manages encrypted LLM profiles (stored under output/config).
var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage encrypted LLM profiles (provider/model/key)",
	Run: func(cmd *cobra.Command, args []string) {
		interactiveRequested := profileInteractive || (profileName == "" && profileDelete == "" && !profileList)
		if interactiveRequested {
			runInteractiveProfile()
			return
		}
		secret := strings.TrimSpace(profileName)
		if secret == "" {
			utils.Info("please set --name for list/save/delete")
			return
		}
		path := utils.DefaultLLMProfilePath("")
		profiles, err := utils.LoadLLMProfiles(path, secret)
		if err != nil && !os.IsNotExist(err) {
			utils.Warning("load profiles failed (use profile name as secret): %v", err)
		}

		if profileList {
			if len(profiles) == 0 {
				utils.Info("No LLM profiles saved.")
				return
			}
			for _, p := range profiles {
				utils.Info("name=%s provider=%s model=%s", p.Name, p.Provider, p.Model)
			}
			return
		}

		if profileDelete != "" {
			profiles = utils.DeleteProfile(profiles, profileDelete)
			if err := utils.SaveLLMProfiles(path, secret, profiles); err != nil {
				utils.Error("delete profile failed: %v", err)
				return
			}
			utils.Success("Profile %s deleted", profileDelete)
			return
		}

		if profileName != "" {
			p := utils.LLMProfile{
				Name:     profileName,
				Provider: firstNonEmpty(profileProvider, utils.DefaultLLMProvider),
				Model:    firstNonEmpty(profileModel, utils.DefaultLLMModel),
				APIKey:   strings.TrimSpace(profileKey),
				BaseURL:  strings.TrimSpace(profileBaseURL),
			}
			if p.APIKey == "" {
				utils.Error("profile key is empty, use --key")
				return
			}
			profiles = utils.UpsertProfile(profiles, p)
			if err := utils.SaveLLMProfiles(path, secret, profiles); err != nil {
				utils.Error("save profile failed: %v", err)
				return
			}
			utils.Success("Profile %s saved (path=%s)", p.Name, path)
			return
		}

		utils.Info("No action taken. Use --list, --name, --delete, or run with -h for help.")
	},
}

func init() {
	rootCmd.AddCommand(llmCmd)
	llmCmd.Flags().StringVar(&profileName, "name", "", "Profile name (also used as encryption secret)")
	llmCmd.Flags().StringVar(&profileKey, "key", "", "API key for profile (required for save)")
	llmCmd.Flags().StringVar(&profileModel, "model", "", "Model id, e.g. gemini-3-flash-preview")
	llmCmd.Flags().StringVar(&profileProvider, "provider", "", "LLM provider, default google")
	llmCmd.Flags().StringVar(&profileBaseURL, "base-url", "", "Custom LLM API base URL")
	llmCmd.Flags().BoolVar(&profileList, "list", false, "List saved profiles")
	llmCmd.Flags().StringVar(&profileDelete, "delete", "", "Delete profile by name")
	llmCmd.Flags().BoolVarP(&profileInteractive, "interactive", "i", false, "Interactive mode: prompt for name/model/key")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func runInteractiveProfile() {
	reader := bufio.NewReader(os.Stdin)
	name := strings.TrimSpace(prompt(reader, "Profile name (required): ", ""))
	if name == "" {
		utils.Error("Profile name is required")
		return
	}
	secret := name
	path := utils.DefaultLLMProfilePath("")
	profiles, _ := utils.LoadLLMProfiles(path, secret)
	defaultProvider := utils.DefaultLLMProvider
	defaultModel := utils.DefaultLLMModel
	defaultKey := ""

	baseURL := strings.TrimSpace(prompt(reader, "Base URL (optional): ", ""))
	provider := defaultProvider
	if baseURL != "" {
		provider = "openai"
	}

	model := strings.TrimSpace(promptModel(reader, defaultModel))
	key := strings.TrimSpace(prompt(reader, "API key (leave blank to keep existing): ", defaultKey))
	if key == "" {
		utils.Error("API key is required")
		return
	}
	p := utils.LLMProfile{
		Name:     name,
		Provider: provider,
		Model:    model,
		APIKey:   key,
		BaseURL:  baseURL,
	}
	profiles = utils.UpsertProfile(profiles, p)
	if err := utils.SaveLLMProfiles(path, secret, profiles); err != nil {
		utils.Error("save profile failed: %v", err)
		return
	}
	utils.Success("Profile %s saved (path=%s)", p.Name, path)
}

func prompt(r *bufio.Reader, msg, def string) string {
	fmt.Print(msg)
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func promptModel(r *bufio.Reader, def string) string {
	models := []struct {
		Label string
		ID    string
	}{
		{"Gemini 3 Flash", "gemini-3-flash"},
		{"Gemini 3 Pro High", "gemini-3-pro-high"},
		{"Gemini 3 Pro Low", "gemini-3-pro-low"},
		{"Gemini 2.5 Flash", "gemini-2.5-flash"},
		{"Gemini 2.5 Pro", "gemini-2.5-pro"},
		{"Gemini 2.5 Flash Thinking", "gemini-2.5-flash-thinking"},
		{"Claude 4.5 Sonnet", "claude-sonnet-4-5"},
		{"Claude 4.5 Sonnet Thinking", "claude-sonnet-4-5-thinking"},
		{"GPT-4o", "gpt-4o"},
	}

	fmt.Printf("Model options:\n")
	for i, m := range models {
		fmt.Printf(" %d) %s (%s)\n", i+1, m.Label, m.ID)
	}
	fmt.Printf("Choose model (enter number or custom ID) [default %s]: ", def)
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}

	// Try to parse as index
	var idx int
	if _, err := fmt.Sscanf(text, "%d", &idx); err == nil {
		if idx >= 1 && idx <= len(models) {
			return models[idx-1].ID
		}
	}
	// Fallback: assume user typed custom model ID
	return text
}

func getOrPromptSecret(allowPrompt bool) string {
	return strings.TrimSpace(profileName)
}
